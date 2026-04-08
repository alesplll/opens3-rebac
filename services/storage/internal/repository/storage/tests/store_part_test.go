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

	partPath := multipartPartPath(multipartDir, "upload-1", 2)
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

	partPath := multipartPartPath(multipartDir, "upload-1", 2)
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

	partPath := multipartPartPath(multipartDir, "upload-1", 2)
	stored, err := os.ReadFile(partPath)
	require.NoError(t, err)
	require.Equal(t, secondContent, stored)
}

func TestStorePart_CanceledDuringWriteCleansUpTempFile(t *testing.T) {
	t.Parallel()

	multipartDir := t.TempDir()
	repository := storageRepo.NewRepository(testStorageConfig{
		dataDir:      t.TempDir(),
		multipartDir: multipartDir,
	})
	err := repository.CreateMultipartSession(context.Background(), "upload-1", 2, "video/mp4")
	require.NoError(t, err)

	reader := &cancelAwareReader{
		firstChunk:        []byte("partial"),
		secondReadStarted: make(chan struct{}),
		unblock:           make(chan struct{}),
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)

	go func() {
		_, storeErr := repository.StorePart(ctx, "upload-1", 2, reader)
		done <- storeErr
	}()

	<-reader.secondReadStarted
	cancel()
	close(reader.unblock)

	err = <-done
	require.ErrorIs(t, err, context.Canceled)

	partPath := multipartPartPath(multipartDir, "upload-1", 2)
	_, statErr := os.Stat(partPath)
	require.ErrorIs(t, statErr, os.ErrNotExist)

	tempMatches, globErr := filepath.Glob(partPath + ".*.tmp")
	require.NoError(t, globErr)
	require.Empty(t, tempMatches)
}

type cancelAwareReader struct {
	firstChunk        []byte
	secondReadStarted chan struct{}
	unblock           chan struct{}
	readIndex         int
}

func (r *cancelAwareReader) Read(p []byte) (int, error) {
	switch r.readIndex {
	case 0:
		r.readIndex++
		return copy(p, r.firstChunk), nil
	case 1:
		r.readIndex++
		close(r.secondReadStarted)
		<-r.unblock
		return 0, nil
	default:
		return 0, io.EOF
	}
}
