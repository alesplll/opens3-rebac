package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	domainerrors "github.com/alesplll/opens3-rebac/services/storage/internal/errors/domain_errors"
)

func (r *repo) RetrieveBlob(ctx context.Context, blobID string, rangeStart, rangeEnd int64) (io.ReadCloser, int64, error) {
	if err := ctx.Err(); err != nil {
		return nil, 0, err
	}

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

	totalSize := stat.Size()
	if rangeStart <= 0 && (rangeEnd == 0 || rangeEnd == -1) {
		return file, totalSize, nil
	}

	if rangeStart < 0 {
		rangeStart = 0
	}

	if rangeStart > totalSize {
		rangeStart = totalSize
	}

	if _, err := file.Seek(rangeStart, io.SeekStart); err != nil {
		_ = file.Close()
		return nil, 0, fmt.Errorf("seek blob file: %w", err)
	}

	effectiveEnd := totalSize - 1
	if rangeEnd >= 0 && rangeEnd < effectiveEnd {
		effectiveEnd = rangeEnd
	}

	if effectiveEnd < rangeStart {
		effectiveEnd = rangeStart - 1
	}

	limitedReader := io.LimitReader(file, effectiveEnd-rangeStart+1)
	return &readCloser{
		Reader: limitedReader,
		Closer: file,
	}, totalSize, nil
}

type readCloser struct {
	io.Reader
	io.Closer
}
