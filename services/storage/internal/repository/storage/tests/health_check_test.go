package tests

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	storageRepo "github.com/alesplll/opens3-rebac/services/storage/internal/repository/storage"
	"github.com/stretchr/testify/require"
)

func TestHealthCheck_DirAccessible(t *testing.T) {
	t.Parallel()

	repository := storageRepo.NewRepository(testStorageConfig{
		dataDir:      t.TempDir(),
		multipartDir: t.TempDir(),
	})

	err := repository.HealthCheck(context.Background())
	require.NoError(t, err)
}

func TestHealthCheck_DirMissing(t *testing.T) {
	t.Parallel()

	missingDir := filepath.Join(t.TempDir(), "missing")
	repository := storageRepo.NewRepository(testStorageConfig{
		dataDir:      missingDir,
		multipartDir: t.TempDir(),
	})

	err := repository.HealthCheck(context.Background())
	require.Error(t, err)
}

func TestHealthCheck_DataDirIsFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	dataFile := filepath.Join(root, "not-a-dir")
	err := os.WriteFile(dataFile, []byte("x"), 0o644)
	require.NoError(t, err)

	repository := storageRepo.NewRepository(testStorageConfig{
		dataDir:      dataFile,
		multipartDir: t.TempDir(),
	})

	err = repository.HealthCheck(context.Background())
	require.Error(t, err)
}
