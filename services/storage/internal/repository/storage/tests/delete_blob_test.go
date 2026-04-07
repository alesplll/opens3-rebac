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
