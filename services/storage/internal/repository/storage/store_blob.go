package storage

import (
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"

	"github.com/google/uuid"

	domainerrors "github.com/alesplll/opens3-rebac/services/storage/internal/errors/domain_errors"
	"github.com/alesplll/opens3-rebac/services/storage/internal/model"
)

func (r *repo) StoreBlob(ctx context.Context, reader io.Reader) (*model.BlobMeta, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if err := os.MkdirAll(r.dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	if err := ensureWritableDiskSpace(r.dataDir); err != nil {
		return nil, err
	}

	blobID := uuid.New().String()
	finalPath := r.blobPath(blobID)
	tempPath := finalPath + ".tmp"

	file, err := os.OpenFile(tempPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("create temp blob file: %w", err)
	}

	defer func() {
		_ = file.Close()
	}()

	hasher := md5.New()
	written, copyErr := io.Copy(io.MultiWriter(file, hasher), newContextReader(ctx, reader))
	if copyErr != nil {
		_ = file.Close()
		_ = os.Remove(tempPath)

		if isDiskFull(copyErr) {
			return nil, domainerrors.ErrDiskFull
		}

		return nil, fmt.Errorf("write blob data: %w", copyErr)
	}

	if err := file.Sync(); err != nil {
		_ = file.Close()
		_ = os.Remove(tempPath)
		return nil, fmt.Errorf("sync temp blob file: %w", err)
	}

	if err := file.Close(); err != nil {
		_ = os.Remove(tempPath)
		return nil, fmt.Errorf("close temp blob file: %w", err)
	}

	if err := os.Rename(tempPath, finalPath); err != nil {
		_ = os.Remove(tempPath)
		return nil, fmt.Errorf("commit blob file: %w", err)
	}

	return &model.BlobMeta{
		BlobID:      blobID,
		ChecksumMD5: fmt.Sprintf("%x", hasher.Sum(nil)),
		SizeBytes:   written,
	}, nil
}

func (r *repo) blobPath(blobID string) string {
	return filepath.Join(r.dataDir, blobID)
}

func ensureWritableDiskSpace(dir string) error {
	var stats syscall.Statfs_t
	if err := syscall.Statfs(dir, &stats); err != nil {
		return fmt.Errorf("statfs data dir: %w", err)
	}

	if stats.Bavail == 0 || stats.Bsize == 0 {
		return domainerrors.ErrDiskFull
	}

	return nil
}

func isDiskFull(err error) bool {
	return errors.Is(err, syscall.ENOSPC)
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func newContextReader(ctx context.Context, reader io.Reader) io.Reader {
	return &contextReader{
		ctx:    ctx,
		reader: reader,
	}
}

func (r *contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}

	n, err := r.reader.Read(p)
	if err != nil {
		return n, err
	}

	if err := r.ctx.Err(); err != nil {
		return n, err
	}

	return n, nil
}
