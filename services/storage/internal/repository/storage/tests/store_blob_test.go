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
	repository := storageRepo.NewRepository(testStorageConfig{
		dataDir:      dataDir,
		multipartDir: t.TempDir(),
	})
	content := []byte("storage blob content")

	meta, err := repository.StoreBlob(context.Background(), bytes.NewReader(content))
	require.NoError(t, err)
	require.NotEmpty(t, meta.BlobID)
	require.Equal(t, int64(len(content)), meta.SizeBytes)
	require.Equal(t, fmt.Sprintf("%x", md5.Sum(content)), meta.ChecksumMD5)

	storedContent, err := os.ReadFile(filepath.Join(dataDir, meta.BlobID))
	require.NoError(t, err)
	require.Equal(t, content, storedContent)
}

func TestStoreBlob_CleanupTempFileOnReadError(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	repository := storageRepo.NewRepository(testStorageConfig{
		dataDir:      dataDir,
		multipartDir: t.TempDir(),
	})

	_, err := repository.StoreBlob(context.Background(), &failingReader{
		chunks: [][]byte{[]byte("partial")},
		err:    errors.New("read failed"),
	})
	require.Error(t, err)

	entries, readErr := os.ReadDir(dataDir)
	require.NoError(t, readErr)
	require.Empty(t, entries)
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
