package storage

import (
	"bytes"
	"context"
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/alesplll/opens3-rebac/services/storage/internal/model"
	"github.com/stretchr/testify/require"
)

func TestAssembleParts_CleanupUsesDetachedContext(t *testing.T) {
	previousHook := afterAssemblePartsCommitHook
	t.Cleanup(func() {
		afterAssemblePartsCommitHook = previousHook
	})

	dataDir := t.TempDir()
	multipartDir := t.TempDir()
	repository := &repo{
		dataDir:      dataDir,
		multipartDir: multipartDir,
	}

	err := repository.CreateMultipartSession(context.Background(), "upload-1", 1, "video/mp4")
	require.NoError(t, err)

	part := []byte("hello world")
	_, err = repository.StorePart(context.Background(), "upload-1", 1, bytes.NewReader(part))
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	afterAssemblePartsCommitHook = func(context.Context) {
		cancel()
	}

	meta, err := repository.AssembleParts(ctx, "upload-1", []model.PartInfo{
		{PartNumber: 1, ChecksumMD5: fmt.Sprintf("%x", md5.Sum(part))},
	}, "blob-1")
	require.NoError(t, err)
	require.NotNil(t, meta)

	_, statErr := os.Stat(filepath.Join(multipartDir, stagingUploadsDirname, "upload-1"))
	require.ErrorIs(t, statErr, os.ErrNotExist)
}

func TestAssembleParts_CanceledBeforeNextPartCopyLeavesNoBlob(t *testing.T) {
	previousHook := beforeAssemblePartCopyHook
	t.Cleanup(func() {
		beforeAssemblePartCopyHook = previousHook
	})

	dataDir := t.TempDir()
	multipartDir := t.TempDir()
	repository := &repo{
		dataDir:      dataDir,
		multipartDir: multipartDir,
	}

	err := repository.CreateMultipartSession(context.Background(), "upload-1", 2, "video/mp4")
	require.NoError(t, err)

	partOne := []byte("hello ")
	partTwo := []byte("world")
	_, err = repository.StorePart(context.Background(), "upload-1", 1, bytes.NewReader(partOne))
	require.NoError(t, err)
	_, err = repository.StorePart(context.Background(), "upload-1", 2, bytes.NewReader(partTwo))
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	beforeAssemblePartCopyHook = func(_ context.Context, partNumber int32) {
		if partNumber == 2 {
			cancel()
		}
	}

	meta, err := repository.AssembleParts(ctx, "upload-1", []model.PartInfo{
		{PartNumber: 1, ChecksumMD5: fmt.Sprintf("%x", md5.Sum(partOne))},
		{PartNumber: 2, ChecksumMD5: fmt.Sprintf("%x", md5.Sum(partTwo))},
	}, "blob-1")
	require.ErrorIs(t, err, context.Canceled)
	require.Nil(t, meta)

	_, statErr := os.Stat(filepath.Join(dataDir, "up", "upload-1"))
	require.ErrorIs(t, statErr, os.ErrNotExist)
}
