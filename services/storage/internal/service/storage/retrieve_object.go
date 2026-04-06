package storage

import (
	"context"
	"io"
)

func (s *storageService) RetrieveObject(ctx context.Context, blobID string, offset, length int64) (io.ReadCloser, int64, error) {
	normalized := normalizeRange(offset, length)
	if normalized.full {
		return s.repo.RetrieveBlob(ctx, blobID)
	}

	return s.repo.RetrieveBlobRange(ctx, blobID, normalized.offset, normalized.length)
}

type normalizedRange struct {
	offset int64
	length int64
	full   bool
}

func normalizeRange(offset, length int64) normalizedRange {
	if offset < 0 {
		offset = 0
	}

	if offset == 0 && length == 0 {
		return normalizedRange{full: true}
	}

	if length < 0 {
		length = 0
	}

	if length == 0 {
		return normalizedRange{
			offset: offset,
			length: maxInt64,
		}
	}

	return normalizedRange{
		offset: offset,
		length: length,
	}
}

const maxInt64 = int64(^uint64(0) >> 1)
