package storage

import (
	"context"
	"io"
)

func (s *storageService) RetrieveObject(ctx context.Context, blobID string, rangeStart, rangeEnd int64) (io.ReadCloser, int64, error) {
	normalizedRange := normalizeRange(rangeStart, rangeEnd)
	if normalizedRange.full {
		return s.repo.RetrieveBlob(ctx, blobID)
	}

	return s.repo.RetrieveBlobRange(ctx, blobID, normalizedRange.offset, normalizedRange.length)
}

type normalizedRange struct {
	offset int64
	length int64
	full   bool
}

func normalizeRange(rangeStart, rangeEnd int64) normalizedRange {
	if rangeStart < 0 {
		rangeStart = 0
	}

	// Backward compatibility with the current implementation plan:
	// (0, 0) historically means "read full blob". Internally we normalize
	// that to a full-read branch before repository execution.
	if rangeStart == 0 && rangeEnd == 0 {
		return normalizedRange{full: true}
	}

	if rangeEnd < -1 {
		rangeEnd = -1
	}

	if rangeStart == 0 && rangeEnd == -1 {
		return normalizedRange{full: true}
	}

	if rangeEnd == -1 {
		return normalizedRange{
			offset: rangeStart,
			length: maxInt64,
		}
	}

	if rangeEnd < rangeStart {
		return normalizedRange{
			offset: rangeStart,
			length: 0,
		}
	}

	return normalizedRange{
		offset: rangeStart,
		length: rangeEnd - rangeStart + 1,
	}
}

const maxInt64 = int64(^uint64(0) >> 1)
