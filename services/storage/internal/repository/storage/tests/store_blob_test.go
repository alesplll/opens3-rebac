package tests

import (
	"bytes"
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	storageRepo "github.com/alesplll/opens3-rebac/services/storage/internal/repository/storage"
	"github.com/stretchr/testify/require"
)

func TestStoreBlob_Success(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	multipartDir := t.TempDir()
	repository := storageRepo.NewRepository(testStorageConfig{
		dataDir:      dataDir,
		multipartDir: multipartDir,
	})
	content := []byte("storage blob content")
	blobID := "blob-1"

	meta, err := repository.StoreBlob(context.Background(), blobID, bytes.NewReader(content))
	require.NoError(t, err)
	require.Equal(t, blobID, meta.BlobID)
	require.Equal(t, int64(len(content)), meta.SizeBytes)
	require.Equal(t, fmt.Sprintf("%x", md5.Sum(content)), meta.ChecksumMD5)

	storedContent, err := os.ReadFile(blobFilePath(dataDir, blobID))
	require.NoError(t, err)
	require.Equal(t, content, storedContent)

	_, err = os.Stat(singlePartUploadPath(multipartDir, blobID))
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestStoreBlob_CleanupTempFileOnReadError(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	repository := storageRepo.NewRepository(testStorageConfig{
		dataDir:      dataDir,
		multipartDir: t.TempDir(),
	})

	_, err := repository.StoreBlob(context.Background(), "blob-read-error", &failingReader{
		chunks: [][]byte{[]byte("partial")},
		err:    errors.New("read failed"),
	})
	require.Error(t, err)

	entries, readErr := os.ReadDir(dataDir)
	require.NoError(t, readErr)
	require.Empty(t, entries)
}

func TestStoreBlob_LargeFile(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	repository := storageRepo.NewRepository(testStorageConfig{
		dataDir:      dataDir,
		multipartDir: t.TempDir(),
	})

	content := make([]byte, 2_621_440) // 2.5 MB
	blobID := "blob-large"
	for i := range content {
		content[i] = byte(i % 256)
	}

	meta, err := repository.StoreBlob(context.Background(), blobID, bytes.NewReader(content))
	require.NoError(t, err)
	require.Equal(t, blobID, meta.BlobID)
	require.Equal(t, int64(len(content)), meta.SizeBytes)
	require.Equal(t, fmt.Sprintf("%x", md5.Sum(content)), meta.ChecksumMD5)

	storedContent, err := os.ReadFile(blobFilePath(dataDir, blobID))
	require.NoError(t, err)
	require.Equal(t, content, storedContent)
}

func TestStoreBlob_EmptyBlob(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	repository := storageRepo.NewRepository(testStorageConfig{
		dataDir:      dataDir,
		multipartDir: t.TempDir(),
	})
	blobID := "blob-empty"

	meta, err := repository.StoreBlob(context.Background(), blobID, bytes.NewReader(nil))
	require.NoError(t, err)
	require.Equal(t, blobID, meta.BlobID)
	require.Equal(t, int64(0), meta.SizeBytes)
	require.Equal(t, "d41d8cd98f00b204e9800998ecf8427e", meta.ChecksumMD5)

	storedContent, err := os.ReadFile(blobFilePath(dataDir, blobID))
	require.NoError(t, err)
	require.Empty(t, storedContent)
}

func TestStoreBlob_ContextCanceled(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	repository := storageRepo.NewRepository(testStorageConfig{
		dataDir:      dataDir,
		multipartDir: t.TempDir(),
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := repository.StoreBlob(ctx, "blob-cancelled", bytes.NewReader([]byte("should not be stored")))
	require.Error(t, err)

	entries, readErr := os.ReadDir(dataDir)
	require.NoError(t, readErr)
	require.Empty(t, entries)
}

func TestStoreBlob_IgnoresStaleTempFileOnRetry(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	repository := storageRepo.NewRepository(testStorageConfig{
		dataDir:      dataDir,
		multipartDir: t.TempDir(),
	})

	blobID := "blob-1"
	require.NoError(t, os.MkdirAll(filepath.Dir(blobFilePath(dataDir, blobID)), 0o755))
	require.NoError(t, os.WriteFile(blobFilePath(dataDir, blobID)+".tmp", []byte("stale temp"), 0o644))

	content := []byte("storage blob content")
	meta, err := repository.StoreBlob(context.Background(), blobID, bytes.NewReader(content))
	require.NoError(t, err)
	require.Equal(t, blobID, meta.BlobID)

	storedContent, err := os.ReadFile(blobFilePath(dataDir, blobID))
	require.NoError(t, err)
	require.Equal(t, content, storedContent)
}

type testStorageConfig struct {
	dataDir      string
	multipartDir string
}

func (c testStorageConfig) DataDir() string {
	return c.dataDir
}

func (c testStorageConfig) MultipartDir() string {
	return c.multipartDir
}

func blobFilePath(dataDir, blobID string) string {
	return filepath.Join(dataDir, blobShardDirName(blobID), blobID)
}

func blobShardDirName(blobID string) string {
	shard := blobID
	if len(blobID) >= 2 {
		shard = blobID[:2]
	}

	return shard
}

func stagingUploadsPath(multipartDir string) string {
	return filepath.Join(multipartDir, "uploads")
}

func singlePartUploadPath(multipartDir, blobID string) string {
	return filepath.Join(stagingUploadsPath(multipartDir), blobID)
}

func multipartSessionPath(multipartDir, uploadID string) string {
	return filepath.Join(stagingUploadsPath(multipartDir), uploadID)
}

func multipartPartPath(multipartDir, uploadID string, partNumber int32) string {
	return filepath.Join(multipartSessionPath(multipartDir, uploadID), fmt.Sprintf("part_%05d", partNumber))
}

func completedMetaPath(multipartDir, uploadID string) string {
	return filepath.Join(multipartDir, "completed", completedMetaShardDirName(uploadID), uploadID+".json")
}

func completedMetaShardDirName(uploadID string) string {
	return blobShardDirName(uploadID)
}

type failingReader struct {
	chunks [][]byte
	err    error
	index  int
}

func (r *failingReader) Read(p []byte) (int, error) {
	if r.index >= len(r.chunks) {
		return 0, r.err
	}

	n := copy(p, r.chunks[r.index])
	r.index++
	return n, nil
}
