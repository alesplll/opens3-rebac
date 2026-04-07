package storage

import (
	"context"
	"io"

	"github.com/alesplll/opens3-rebac/services/storage/internal/model"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/logger"
	"go.uber.org/zap"
)

func (s *storageService) StoreObject(ctx context.Context, reader io.Reader, _ int64, contentType string) (*model.BlobMeta, error) {
	meta, err := s.repo.StoreBlob(ctx, reader)
	if err != nil {
		logger.Error(ctx, "failed to store blob", zap.Error(err), zap.String("content_type", contentType))
		return nil, err
	}

	meta.ContentType = contentType
	logger.Info(
		ctx,
		"stored blob",
		zap.String("blob_id", meta.BlobID),
		zap.Int64("size_bytes", meta.SizeBytes),
		zap.String("content_type", meta.ContentType),
	)
	return meta, nil
}
