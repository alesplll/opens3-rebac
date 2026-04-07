package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"syscall"

	domainerrors "github.com/alesplll/opens3-rebac/services/storage/internal/errors/domain_errors"
	"github.com/alesplll/opens3-rebac/services/storage/internal/model"
)

func (r *repo) StoreBlob(ctx context.Context, blobID string, reader io.Reader) (*model.BlobMeta, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if err := ensureDirReady(r.dataDir); err != nil {
		return nil, err
	}

	result, err := writeAtomically(ctx, r.blobPath(blobID), reader, "blob")
	if err != nil {
		return nil, err
	}

	return &model.BlobMeta{
		BlobID:      blobID,
		ChecksumMD5: result.checksumMD5,
		SizeBytes:   result.sizeBytes,
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
