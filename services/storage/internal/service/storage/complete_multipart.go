package storage

import (
	"context"

	domainerrors "github.com/alesplll/opens3-rebac/services/storage/internal/errors/domain_errors"
	"github.com/alesplll/opens3-rebac/services/storage/internal/model"
	"github.com/alesplll/opens3-rebac/services/storage/internal/observability"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/logger"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func (s *storageService) CompleteMultipartUpload(ctx context.Context, uploadID string, parts []model.PartInfo) (*model.BlobMeta, error) {
	if err := validateCompletedParts(parts); err != nil {
		return nil, err
	}

	meta, err := s.repo.AssembleParts(ctx, uploadID, parts, uuid.New().String())
	if err != nil {
		logger.Error(
			ctx,
			"failed to complete multipart upload",
			zap.Error(err),
			zap.String("upload_id", uploadID),
			zap.Int("parts_count", len(parts)),
		)
		return nil, err
	}

	observability.AddWriteBytes(ctx, meta.SizeBytes)
	logger.Info(
		ctx,
		"completed multipart upload",
		zap.String("upload_id", uploadID),
		zap.String("blob_id", meta.BlobID),
		zap.Int64("size_bytes", meta.SizeBytes),
	)
	return meta, nil
}

func validateCompletedParts(parts []model.PartInfo) error {
	if len(parts) == 0 {
		return domainerrors.ErrInvalidParts
	}

	var previous int32
	for i, part := range parts {
		if part.PartNumber < 1 {
			return domainerrors.ErrInvalidPartNumber
		}
		if part.ChecksumMD5 == "" {
			return domainerrors.ErrChecksumMismatch
		}
		if i > 0 && part.PartNumber <= previous {
			return domainerrors.ErrInvalidParts
		}

		previous = part.PartNumber
	}

	return nil
}
