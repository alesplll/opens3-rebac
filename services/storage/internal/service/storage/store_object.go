package storage

import (
	"context"
	"io"

	domainerrors "github.com/alesplll/opens3-rebac/services/storage/internal/errors/domain_errors"
	"github.com/alesplll/opens3-rebac/services/storage/internal/model"
	"github.com/alesplll/opens3-rebac/services/storage/internal/observability"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/logger"
	"go.uber.org/zap"
)

const defaultContentType = "application/octet-stream"

func (s *storageService) StoreObject(ctx context.Context, reader io.Reader, size int64, contentType string) (*model.BlobMeta, error) {
	if size < 0 {
		return nil, domainerrors.ErrInvalidBlobSize
	}

	if contentType == "" {
		contentType = defaultContentType
	}

	meta, err := s.repo.StoreBlob(ctx, reader)
	if err != nil {
		logger.Error(ctx, "failed to store blob", zap.Error(err), zap.String("content_type", contentType))
		return nil, err
	}

	if size > 0 && meta.SizeBytes != size {
		if deleteErr := s.repo.DeleteBlob(ctx, meta.BlobID); deleteErr != nil {
			logger.Error(ctx, "failed to cleanup blob after size mismatch", zap.Error(deleteErr), zap.String("blob_id", meta.BlobID))
		}

		logger.Error(
			ctx,
			"stored blob size mismatch",
			zap.String("blob_id", meta.BlobID),
			zap.Int64("expected_size", size),
			zap.Int64("actual_size", meta.SizeBytes),
		)
		return nil, domainerrors.ErrInvalidBlobSize
	}

	meta.ContentType = contentType
	observability.AddWriteBytes(ctx, meta.SizeBytes)
	logger.Info(
		ctx,
		"stored blob",
		zap.String("blob_id", meta.BlobID),
		zap.Int64("size_bytes", meta.SizeBytes),
		zap.String("content_type", meta.ContentType),
	)
	return meta, nil
}
