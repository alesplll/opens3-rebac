package storage

import (
	"context"
	"io"

	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/logger"
	"go.uber.org/zap"
)

func (s *storageService) RetrieveObject(ctx context.Context, blobID string, offset, length int64) (io.ReadCloser, int64, error) {
	normalized := normalizeRange(offset, length)
	if normalized.full {
		reader, totalSize, err := s.repo.RetrieveBlob(ctx, blobID)
		if err != nil {
			logger.Error(ctx, "failed to retrieve full blob", zap.Error(err), zap.String("blob_id", blobID))
			return nil, 0, err
		}

		return reader, totalSize, nil
	}

	reader, totalSize, err := s.repo.RetrieveBlobRange(ctx, blobID, normalized.offset, normalized.length)
	if err != nil {
		logger.Error(
			ctx,
			"failed to retrieve blob range",
			zap.Error(err),
			zap.String("blob_id", blobID),
			zap.Int64("offset", normalized.offset),
			zap.Int64("length", normalized.length),
		)
		return nil, 0, err
	}

	return reader, totalSize, nil
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
