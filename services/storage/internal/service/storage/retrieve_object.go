package storage

import (
	"context"
	"io"
)

func (s *storageService) RetrieveObject(ctx context.Context, blobID string, rangeStart, rangeEnd int64) (io.ReadCloser, int64, error) {
	return s.repo.RetrieveBlob(ctx, blobID, rangeStart, rangeEnd)
}
