package storage

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"os"

	domainerrors "github.com/alesplll/opens3-rebac/services/storage/internal/errors/domain_errors"
)

type writeResult struct {
	checksumMD5 string
	sizeBytes   int64
}

func ensureDirReady(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}

	if err := ensureWritableDiskSpace(dir); err != nil {
		return err
	}

	return nil
}

func writeAtomically(ctx context.Context, finalPath string, reader io.Reader, entity string) (*writeResult, error) {
	tempPath := finalPath + ".tmp"

	file, err := os.OpenFile(tempPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("create temp %s file: %w", entity, err)
	}

	hasher := md5.New()
	written, copyErr := io.Copy(io.MultiWriter(file, hasher), newContextReader(ctx, reader))
	if copyErr != nil {
		_ = file.Close()
		_ = os.Remove(tempPath)
		if isDiskFull(copyErr) {
			return nil, domainDiskFull(copyErr)
		}

		return nil, fmt.Errorf("write %s data: %w", entity, copyErr)
	}

	if err := file.Sync(); err != nil {
		_ = file.Close()
		_ = os.Remove(tempPath)
		return nil, fmt.Errorf("sync temp %s file: %w", entity, err)
	}

	if err := file.Close(); err != nil {
		_ = os.Remove(tempPath)
		return nil, fmt.Errorf("close temp %s file: %w", entity, err)
	}

	if err := os.Rename(tempPath, finalPath); err != nil {
		_ = os.Remove(tempPath)
		return nil, fmt.Errorf("commit %s file: %w", entity, err)
	}

	return &writeResult{
		checksumMD5: fmt.Sprintf("%x", hasher.Sum(nil)),
		sizeBytes:   written,
	}, nil
}

func domainDiskFull(err error) error {
	if isDiskFull(err) {
		return domainerrors.ErrDiskFull
	}

	return err
}
