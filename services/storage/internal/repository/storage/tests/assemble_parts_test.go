package tests

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	domainerrors "github.com/alesplll/opens3-rebac/services/storage/internal/errors/domain_errors"
	"github.com/alesplll/opens3-rebac/services/storage/internal/model"
	"github.com/alesplll/opens3-rebac/services/storage/internal/repository"
	storageRepo "github.com/alesplll/opens3-rebac/services/storage/internal/repository/storage"
	"github.com/stretchr/testify/require"
)

func setupMultipartSinglePart(t *testing.T, repository repository.StorageRepository, uploadID string, part []byte) string {
	t.Helper()

	err := repository.CreateMultipartSession(context.Background(), uploadID, 1, "video/mp4")
	require.NoError(t, err)

	_, err = repository.StorePart(context.Background(), uploadID, 1, bytes.NewReader(part))
	require.NoError(t, err)

	return fmt.Sprintf("%x", md5.Sum(part))
}

func TestAssembleParts_Success(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	multipartDir := t.TempDir()
	repository := storageRepo.NewRepository(testStorageConfig{
		dataDir:      dataDir,
		multipartDir: multipartDir,
	})
	err := repository.CreateMultipartSession(context.Background(), "upload-1", 2, "video/mp4")
	require.NoError(t, err)

	partOne := []byte("hello ")
	partTwo := []byte("world")
	_, err = repository.StorePart(context.Background(), "upload-1", 1, bytes.NewReader(partOne))
	require.NoError(t, err)
	_, err = repository.StorePart(context.Background(), "upload-1", 2, bytes.NewReader(partTwo))
	require.NoError(t, err)

	parts := []model.PartInfo{
		{PartNumber: 1, ChecksumMD5: fmt.Sprintf("%x", md5.Sum(partOne))},
		{PartNumber: 2, ChecksumMD5: fmt.Sprintf("%x", md5.Sum(partTwo))},
	}

	meta, err := repository.AssembleParts(context.Background(), "upload-1", parts, "blob-1")
	require.NoError(t, err)
	require.Equal(t, "upload-1", meta.BlobID)
	require.Equal(t, "video/mp4", meta.ContentType)
	require.Equal(t, int64(len(partOne)+len(partTwo)), meta.SizeBytes)

	reader, totalSize, err := repository.RetrieveBlob(context.Background(), "upload-1")
	require.NoError(t, err)
	t.Cleanup(func() { _ = reader.Close() })
	require.Equal(t, int64(len(partOne)+len(partTwo)), totalSize)

	body, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.Equal(t, append(partOne, partTwo...), body)
	require.Equal(t, fmt.Sprintf("%x", md5.Sum(body)), meta.ChecksumMD5)

	_, err = os.Stat(multipartSessionPath(multipartDir, "upload-1"))
	require.ErrorIs(t, err, os.ErrNotExist)

	retryMeta, err := repository.AssembleParts(context.Background(), "upload-1", parts, "blob-retry")
	require.NoError(t, err)
	require.Equal(t, meta, retryMeta)
}

func TestAssembleParts_ChecksumMismatch(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	multipartDir := t.TempDir()
	repository := storageRepo.NewRepository(testStorageConfig{
		dataDir:      dataDir,
		multipartDir: multipartDir,
	})
	err := repository.CreateMultipartSession(context.Background(), "upload-1", 1, "video/mp4")
	require.NoError(t, err)

	part := []byte("hello")
	_, err = repository.StorePart(context.Background(), "upload-1", 1, bytes.NewReader(part))
	require.NoError(t, err)

	meta, err := repository.AssembleParts(context.Background(), "upload-1", []model.PartInfo{
		{PartNumber: 1, ChecksumMD5: "bad-checksum"},
	}, "blob-1")
	require.ErrorIs(t, err, domainerrors.ErrChecksumMismatch)
	require.Nil(t, meta)

	_, err = os.Stat(blobFilePath(dataDir, "upload-1"))
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestAssembleParts_InvalidExpectedParts(t *testing.T) {
	t.Parallel()

	repository := storageRepo.NewRepository(testStorageConfig{
		dataDir:      t.TempDir(),
		multipartDir: t.TempDir(),
	})
	err := repository.CreateMultipartSession(context.Background(), "upload-1", 2, "video/mp4")
	require.NoError(t, err)

	part := []byte("hello")
	_, err = repository.StorePart(context.Background(), "upload-1", 1, bytes.NewReader(part))
	require.NoError(t, err)

	meta, err := repository.AssembleParts(context.Background(), "upload-1", []model.PartInfo{
		{PartNumber: 1, ChecksumMD5: fmt.Sprintf("%x", md5.Sum(part))},
	}, "blob-1")
	require.ErrorIs(t, err, domainerrors.ErrInvalidParts)
	require.Nil(t, meta)
}

func TestAssembleParts_SucceedsWhenCleanupFails(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	multipartDir := t.TempDir()
	repository := storageRepo.NewRepository(testStorageConfig{
		dataDir:      dataDir,
		multipartDir: multipartDir,
	})

	part := []byte("hello world")
	checksum := setupMultipartSinglePart(t, repository, "upload-1", part)

	require.NoError(t, os.MkdirAll(filepath.Dir(completedMetaPath(multipartDir, "upload-1")), 0o755))
	require.NoError(t, os.Chmod(stagingUploadsPath(multipartDir), 0o555))
	t.Cleanup(func() {
		_ = os.Chmod(stagingUploadsPath(multipartDir), 0o755)
	})

	meta, err := repository.AssembleParts(context.Background(), "upload-1", []model.PartInfo{
		{PartNumber: 1, ChecksumMD5: checksum},
	}, "blob-1")
	require.NoError(t, err)
	require.NotNil(t, meta)
	require.Equal(t, "upload-1", meta.BlobID)

	body, readErr := os.ReadFile(blobFilePath(dataDir, "upload-1"))
	require.NoError(t, readErr)
	require.Equal(t, part, body)

	_, statErr := os.Stat(multipartSessionPath(multipartDir, "upload-1"))
	require.NoError(t, statErr)
}

func TestAssembleParts_IdempotentRetryAfterCleanupFailure(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	multipartDir := t.TempDir()
	repository := storageRepo.NewRepository(testStorageConfig{
		dataDir:      dataDir,
		multipartDir: multipartDir,
	})

	part := []byte("hello world")
	checksum := setupMultipartSinglePart(t, repository, "upload-1", part)

	require.NoError(t, os.MkdirAll(filepath.Dir(completedMetaPath(multipartDir, "upload-1")), 0o755))
	require.NoError(t, os.Chmod(stagingUploadsPath(multipartDir), 0o555))
	t.Cleanup(func() {
		_ = os.Chmod(stagingUploadsPath(multipartDir), 0o755)
	})

	firstMeta, err := repository.AssembleParts(context.Background(), "upload-1", []model.PartInfo{
		{PartNumber: 1, ChecksumMD5: checksum},
	}, "blob-first")
	require.NoError(t, err)
	require.NotNil(t, firstMeta)
	require.Equal(t, "upload-1", firstMeta.BlobID)

	secondMeta, err := repository.AssembleParts(context.Background(), "upload-1", []model.PartInfo{
		{PartNumber: 1, ChecksumMD5: checksum},
	}, "blob-second")
	require.NoError(t, err)
	require.Equal(t, firstMeta, secondMeta)

	entries, readErr := os.ReadDir(dataDir)
	require.NoError(t, readErr)
	require.Len(t, entries, 1)
	require.Equal(t, blobShardDirName("upload-1"), entries[0].Name())
}

func TestAssembleParts_FallsBackWhenCompletedMetaIsCorrupted(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	multipartDir := t.TempDir()
	repository := storageRepo.NewRepository(testStorageConfig{
		dataDir:      dataDir,
		multipartDir: multipartDir,
	})

	part := []byte("hello world")
	checksum := setupMultipartSinglePart(t, repository, "upload-1", part)

	require.NoError(t, os.MkdirAll(filepath.Dir(completedMetaPath(multipartDir, "upload-1")), 0o755))
	require.NoError(t, os.WriteFile(completedMetaPath(multipartDir, "upload-1"), []byte("{"), 0o644))

	meta, err := repository.AssembleParts(context.Background(), "upload-1", []model.PartInfo{
		{PartNumber: 1, ChecksumMD5: checksum},
	}, "blob-ignored")
	require.NoError(t, err)
	require.NotNil(t, meta)
	require.Equal(t, "upload-1", meta.BlobID)

	body, readErr := os.ReadFile(blobFilePath(dataDir, "upload-1"))
	require.NoError(t, readErr)
	require.Equal(t, part, body)

	completedMetaBytes, readMetaErr := os.ReadFile(completedMetaPath(multipartDir, "upload-1"))
	require.NoError(t, readMetaErr)

	var completedMeta model.BlobMeta
	require.NoError(t, json.Unmarshal(completedMetaBytes, &completedMeta))
	require.Equal(t, *meta, completedMeta)
}

func TestAssembleParts_RebuildsWhenCompletedMetaExistsButBlobIsMissing(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	multipartDir := t.TempDir()
	repository := storageRepo.NewRepository(testStorageConfig{
		dataDir:      dataDir,
		multipartDir: multipartDir,
	})

	part := []byte("hello world")
	checksum := setupMultipartSinglePart(t, repository, "upload-1", part)

	require.NoError(t, os.MkdirAll(filepath.Dir(completedMetaPath(multipartDir, "upload-1")), 0o755))

	completedMetaBytes, err := json.Marshal(model.BlobMeta{
		BlobID:      "upload-1",
		ChecksumMD5: "stale-checksum",
		SizeBytes:   999,
		ContentType: "video/mp4",
	})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(completedMetaPath(multipartDir, "upload-1"), completedMetaBytes, 0o644))

	meta, err := repository.AssembleParts(context.Background(), "upload-1", []model.PartInfo{
		{PartNumber: 1, ChecksumMD5: checksum},
	}, "blob-ignored")
	require.NoError(t, err)
	require.NotNil(t, meta)
	require.Equal(t, "upload-1", meta.BlobID)
	require.Equal(t, int64(len(part)), meta.SizeBytes)

	body, readErr := os.ReadFile(blobFilePath(dataDir, "upload-1"))
	require.NoError(t, readErr)
	require.Equal(t, part, body)

	retryMeta, err := repository.AssembleParts(context.Background(), "upload-1", []model.PartInfo{
		{PartNumber: 1, ChecksumMD5: checksum},
	}, "blob-retry")
	require.NoError(t, err)
	require.Equal(t, meta, retryMeta)
}

func TestAssembleParts_CorruptedCompletedMetaWithoutSessionReturnsUploadNotFound(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	multipartDir := t.TempDir()
	repository := storageRepo.NewRepository(testStorageConfig{
		dataDir:      dataDir,
		multipartDir: multipartDir,
	})

	require.NoError(t, os.MkdirAll(filepath.Dir(completedMetaPath(multipartDir, "upload-1")), 0o755))
	require.NoError(t, os.WriteFile(completedMetaPath(multipartDir, "upload-1"), []byte("{"), 0o644))

	meta, err := repository.AssembleParts(context.Background(), "upload-1", []model.PartInfo{
		{PartNumber: 1, ChecksumMD5: "unused"},
	}, "blob-ignored")
	require.ErrorIs(t, err, domainerrors.ErrUploadNotFound)
	require.Nil(t, meta)
}

func TestAssembleParts_IgnoresStaleTempFileOnRetry(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	multipartDir := t.TempDir()
	repository := storageRepo.NewRepository(testStorageConfig{
		dataDir:      dataDir,
		multipartDir: multipartDir,
	})

	part := []byte("hello world")
	checksum := setupMultipartSinglePart(t, repository, "upload-1", part)

	require.NoError(t, os.MkdirAll(filepath.Dir(blobFilePath(dataDir, "upload-1")), 0o755))
	require.NoError(t, os.WriteFile(blobFilePath(dataDir, "upload-1")+".tmp", []byte("stale temp"), 0o644))

	meta, err := repository.AssembleParts(context.Background(), "upload-1", []model.PartInfo{
		{PartNumber: 1, ChecksumMD5: checksum},
	}, "blob-ignored")
	require.NoError(t, err)
	require.NotNil(t, meta)
	require.Equal(t, "upload-1", meta.BlobID)

	body, readErr := os.ReadFile(blobFilePath(dataDir, "upload-1"))
	require.NoError(t, readErr)
	require.Equal(t, part, body)
}
