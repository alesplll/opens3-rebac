package tests

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	storageRepo "github.com/alesplll/opens3-rebac/services/storage/internal/repository/storage"
	"github.com/stretchr/testify/require"
)

func TestCleanupMultipart_Success(t *testing.T) {
	t.Parallel()

	multipartDir := t.TempDir()
	repository := storageRepo.NewRepository(testStorageConfig{
		dataDir:      t.TempDir(),
		multipartDir: multipartDir,
	})
	err := repository.CreateMultipartSession(context.Background(), "upload-1", 1, "video/mp4")
	require.NoError(t, err)

	err = repository.CleanupMultipart(context.Background(), "upload-1")
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(multipartDir, "upload-1"))
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestCleanupMultipart_Idempotent(t *testing.T) {
	t.Parallel()

	repository := storageRepo.NewRepository(testStorageConfig{
		dataDir:      t.TempDir(),
		multipartDir: t.TempDir(),
	})

	err := repository.CleanupMultipart(context.Background(), "missing")
	require.NoError(t, err)
}
