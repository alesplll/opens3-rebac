package tests

import (
	"bytes"
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	domainerrors "github.com/alesplll/opens3-rebac/services/storage/internal/errors/domain_errors"
	"github.com/alesplll/opens3-rebac/services/storage/internal/model"
	storageRepo "github.com/alesplll/opens3-rebac/services/storage/internal/repository/storage"
	"github.com/stretchr/testify/require"
)

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
	require.Equal(t, "blob-1", meta.BlobID)
	require.Equal(t, "video/mp4", meta.ContentType)
	require.Equal(t, int64(len(partOne)+len(partTwo)), meta.SizeBytes)

	reader, totalSize, err := repository.RetrieveBlob(context.Background(), "blob-1")
	require.NoError(t, err)
	t.Cleanup(func() { _ = reader.Close() })
	require.Equal(t, int64(len(partOne)+len(partTwo)), totalSize)

	body, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.Equal(t, append(partOne, partTwo...), body)
	require.Equal(t, fmt.Sprintf("%x", md5.Sum(body)), meta.ChecksumMD5)

	_, err = os.Stat(filepath.Join(multipartDir, "upload-1"))
	require.ErrorIs(t, err, os.ErrNotExist)
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

	_, err = os.Stat(filepath.Join(dataDir, "blob-1"))
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
