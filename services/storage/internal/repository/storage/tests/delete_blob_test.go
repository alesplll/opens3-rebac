package tests

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	storageRepo "github.com/alesplll/opens3-rebac/services/storage/internal/repository/storage"
	"github.com/stretchr/testify/require"
)

func TestDeleteBlob_Success(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	repository := storageRepo.NewRepository(testStorageConfig{
		dataDir:      dataDir,
		multipartDir: t.TempDir(),
	})

	blobPath := filepath.Join(dataDir, "blob-1")
	err := os.WriteFile(blobPath, []byte("content"), 0o644)
	require.NoError(t, err)

	err = repository.DeleteBlob(context.Background(), "blob-1")
	require.NoError(t, err)

	_, err = os.Stat(blobPath)
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestDeleteBlob_Idempotent(t *testing.T) {
	t.Parallel()

	repository := storageRepo.NewRepository(testStorageConfig{
		dataDir:      t.TempDir(),
		multipartDir: t.TempDir(),
	})

	err := repository.DeleteBlob(context.Background(), "missing-blob")
	require.NoError(t, err)
}

func TestDeleteBlob_RemovesCompletedMultipartMeta(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	multipartDir := t.TempDir()
	repository := storageRepo.NewRepository(testStorageConfig{
		dataDir:      dataDir,
		multipartDir: multipartDir,
	})

	blobPath := filepath.Join(dataDir, "upload-1")
	require.NoError(t, os.WriteFile(blobPath, []byte("content"), 0o644))

	completedDir := filepath.Join(multipartDir, "completed")
	require.NoError(t, os.MkdirAll(completedDir, 0o755))
	completedMetaPath := filepath.Join(completedDir, "upload-1.json")
	require.NoError(t, os.WriteFile(completedMetaPath, []byte(`{"blob_id":"upload-1"}`), 0o644))

	err := repository.DeleteBlob(context.Background(), "upload-1")
	require.NoError(t, err)

	_, err = os.Stat(blobPath)
	require.ErrorIs(t, err, os.ErrNotExist)

	_, err = os.Stat(completedMetaPath)
	require.ErrorIs(t, err, os.ErrNotExist)
}
