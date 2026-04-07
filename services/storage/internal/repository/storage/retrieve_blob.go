package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	domainerrors "github.com/alesplll/opens3-rebac/services/storage/internal/errors/domain_errors"
)

func (r *repo) RetrieveBlob(ctx context.Context, blobID string) (io.ReadCloser, int64, error) {
	if err := ctx.Err(); err != nil {
		return nil, 0, err
	}

	file, totalSize, err := r.openBlob(blobID)
	if err != nil {
		return nil, 0, err
	}

	return file, totalSize, nil
}

func (r *repo) RetrieveBlobRange(ctx context.Context, blobID string, offset, length int64) (io.ReadCloser, int64, error) {
	if err := ctx.Err(); err != nil {
		return nil, 0, err
	}

	file, totalSize, err := r.openBlob(blobID)
	if err != nil {
		return nil, 0, err
	}

	if offset > totalSize {
		offset = totalSize
	}

	if _, err := file.Seek(offset, io.SeekStart); err != nil {
		_ = file.Close()
		return nil, 0, fmt.Errorf("seek blob file: %w", err)
	}

	remaining := totalSize - offset
	if remaining < 0 {
		remaining = 0
	}

	if length > remaining {
		length = remaining
	}

	limitedReader := io.LimitReader(file, length)
	return &readCloser{
		Reader: limitedReader,
		Closer: file,
	}, totalSize, nil
}

func (r *repo) openBlob(blobID string) (*os.File, int64, error) {
	file, err := os.Open(r.blobPath(blobID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, 0, domainerrors.ErrBlobNotFound
		}

		return nil, 0, fmt.Errorf("open blob file: %w", err)
	}

	stat, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, 0, fmt.Errorf("stat blob file: %w", err)
	}

	return file, stat.Size(), nil
}

type readCloser struct {
	io.Reader
	io.Closer
}
