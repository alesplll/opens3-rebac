package tests

import (
	"bytes"
	"context"
	"io"
	"testing"

	domainerrors "github.com/alesplll/opens3-rebac/services/storage/internal/errors/domain_errors"
	"github.com/alesplll/opens3-rebac/services/storage/internal/repository"
	storageRepo "github.com/alesplll/opens3-rebac/services/storage/internal/repository/storage"
	"github.com/stretchr/testify/require"
)

func TestRetrieveBlob_Success(t *testing.T) {
	t.Parallel()

	repository, blobID, content := prepareStoredBlob(t)

	reader, totalSize, err := repository.RetrieveBlob(context.Background(), blobID)
	require.NoError(t, err)
	require.Equal(t, int64(len(content)), totalSize)
	t.Cleanup(func() { _ = reader.Close() })

	actual, readErr := io.ReadAll(reader)
	require.NoError(t, readErr)
	require.Equal(t, content, actual)
}

func TestRetrieveBlob_NotFound(t *testing.T) {
	t.Parallel()

	repository := storageRepo.NewRepository(testStorageConfig{
		dataDir:      t.TempDir(),
		multipartDir: t.TempDir(),
	})

	reader, totalSize, err := repository.RetrieveBlob(context.Background(), "missing-blob")
	require.ErrorIs(t, err, domainerrors.ErrBlobNotFound)
	require.Nil(t, reader)
	require.Zero(t, totalSize)
}

func TestRetrieveBlob_Range(t *testing.T) {
	t.Parallel()

	repository, blobID, content := prepareStoredBlob(t)

	reader, totalSize, err := repository.RetrieveBlobRange(context.Background(), blobID, 4, 5)
	require.NoError(t, err)
	require.Equal(t, int64(len(content)), totalSize)
	t.Cleanup(func() { _ = reader.Close() })

	actual, readErr := io.ReadAll(reader)
	require.NoError(t, readErr)
	require.Equal(t, content[4:9], actual)
}

func TestRetrieveBlob_RangePastEnd(t *testing.T) {
	t.Parallel()

	repository, blobID, content := prepareStoredBlob(t)

	reader, totalSize, err := repository.RetrieveBlobRange(context.Background(), blobID, 10, 1000)
	require.NoError(t, err)
	require.Equal(t, int64(len(content)), totalSize)
	t.Cleanup(func() { _ = reader.Close() })

	actual, readErr := io.ReadAll(reader)
	require.NoError(t, readErr)
	require.Equal(t, content[10:], actual)
}

func TestRetrieveBlob_RangeOffsetPastEnd(t *testing.T) {
	t.Parallel()

	repository, blobID, content := prepareStoredBlob(t)

	reader, totalSize, err := repository.RetrieveBlobRange(context.Background(), blobID, int64(len(content)+10), 100)
	require.NoError(t, err)
	require.Equal(t, int64(len(content)), totalSize)
	t.Cleanup(func() { _ = reader.Close() })

	actual, readErr := io.ReadAll(reader)
	require.NoError(t, readErr)
	require.Empty(t, actual)
}

func TestRetrieveBlobRange_OffsetZeroFullLength(t *testing.T) {
	t.Parallel()

	repository, blobID, content := prepareStoredBlob(t)

	reader, totalSize, err := repository.RetrieveBlobRange(context.Background(), blobID, 0, int64(len(content)))
	require.NoError(t, err)
	require.Equal(t, int64(len(content)), totalSize)
	t.Cleanup(func() { _ = reader.Close() })

	actual, readErr := io.ReadAll(reader)
	require.NoError(t, readErr)
	require.Equal(t, content, actual)
}

func prepareStoredBlob(t *testing.T) (repository.StorageRepository, string, []byte) {
	t.Helper()

	repo := storageRepo.NewRepository(testStorageConfig{
		dataDir:      t.TempDir(),
		multipartDir: t.TempDir(),
	})

	content := []byte("0123456789abcdef")
	meta, err := repo.StoreBlob(context.Background(), bytes.NewReader(content))
	require.NoError(t, err)

	return repo, meta.BlobID, content
}
