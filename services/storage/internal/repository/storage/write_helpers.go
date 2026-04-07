package storage

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path/filepath"

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
	file, tempPath, err := createAtomicTempFile(finalPath, entity)
	if err != nil {
		return nil, err
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

func createAtomicTempFile(finalPath string, entity string) (*os.File, string, error) {
	dir := filepath.Dir(finalPath)
	base := filepath.Base(finalPath)

	file, err := os.CreateTemp(dir, base+".*.tmp")
	if err != nil {
		return nil, "", fmt.Errorf("create temp %s file: %w", entity, err)
	}

	if err := file.Chmod(0o644); err != nil {
		tempPath := file.Name()
		_ = file.Close()
		_ = os.Remove(tempPath)
		return nil, "", fmt.Errorf("chmod temp %s file: %w", entity, err)
	}

	return file, file.Name(), nil
}

func domainDiskFull(err error) error {
	if isDiskFull(err) {
		return domainerrors.ErrDiskFull
	}

	return err
}
