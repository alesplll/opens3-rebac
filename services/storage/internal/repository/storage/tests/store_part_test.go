package tests

import (
	"bytes"
	"context"
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	domainerrors "github.com/alesplll/opens3-rebac/services/storage/internal/errors/domain_errors"
	storageRepo "github.com/alesplll/opens3-rebac/services/storage/internal/repository/storage"
	"github.com/stretchr/testify/require"
)

func TestStorePart_Success(t *testing.T) {
	t.Parallel()

	multipartDir := t.TempDir()
	repository := storageRepo.NewRepository(testStorageConfig{
		dataDir:      t.TempDir(),
		multipartDir: multipartDir,
	})
	err := repository.CreateMultipartSession(context.Background(), "upload-1", 2, "video/mp4")
	require.NoError(t, err)

	content := []byte("part content")
	checksum, err := repository.StorePart(context.Background(), "upload-1", 2, bytes.NewReader(content))
	require.NoError(t, err)
	require.Equal(t, fmt.Sprintf("%x", md5.Sum(content)), checksum)

	partPath := filepath.Join(multipartDir, "upload-1", "part_2")
	stored, err := os.ReadFile(partPath)
	require.NoError(t, err)
	require.Equal(t, content, stored)
}

func TestStorePart_UploadNotFound(t *testing.T) {
	t.Parallel()

	repository := storageRepo.NewRepository(testStorageConfig{
		dataDir:      t.TempDir(),
		multipartDir: t.TempDir(),
	})

	checksum, err := repository.StorePart(context.Background(), "missing", 1, bytes.NewReader([]byte("part content")))
	require.ErrorIs(t, err, domainerrors.ErrUploadNotFound)
	require.Empty(t, checksum)
}

func TestStorePart_IgnoresStaleTempFileOnRetry(t *testing.T) {
	t.Parallel()

	multipartDir := t.TempDir()
	repository := storageRepo.NewRepository(testStorageConfig{
		dataDir:      t.TempDir(),
		multipartDir: multipartDir,
	})
	err := repository.CreateMultipartSession(context.Background(), "upload-1", 2, "video/mp4")
	require.NoError(t, err)

	partPath := filepath.Join(multipartDir, "upload-1", "part_2")
	require.NoError(t, os.WriteFile(partPath+".tmp", []byte("stale temp"), 0o644))

	content := []byte("part content")
	checksum, err := repository.StorePart(context.Background(), "upload-1", 2, bytes.NewReader(content))
	require.NoError(t, err)
	require.Equal(t, fmt.Sprintf("%x", md5.Sum(content)), checksum)

	stored, err := os.ReadFile(partPath)
	require.NoError(t, err)
	require.Equal(t, content, stored)
}

func TestStorePart_RetryOverwritesExistingPart(t *testing.T) {
	t.Parallel()

	multipartDir := t.TempDir()
	repository := storageRepo.NewRepository(testStorageConfig{
		dataDir:      t.TempDir(),
		multipartDir: multipartDir,
	})
	err := repository.CreateMultipartSession(context.Background(), "upload-1", 2, "video/mp4")
	require.NoError(t, err)

	firstContent := []byte("first")
	_, err = repository.StorePart(context.Background(), "upload-1", 2, bytes.NewReader(firstContent))
	require.NoError(t, err)

	secondContent := []byte("second")
	checksum, err := repository.StorePart(context.Background(), "upload-1", 2, bytes.NewReader(secondContent))
	require.NoError(t, err)
	require.Equal(t, fmt.Sprintf("%x", md5.Sum(secondContent)), checksum)

	partPath := filepath.Join(multipartDir, "upload-1", "part_2")
	stored, err := os.ReadFile(partPath)
	require.NoError(t, err)
	require.Equal(t, secondContent, stored)
}
