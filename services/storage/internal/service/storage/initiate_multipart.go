package storage

import (
	"context"

	domainerrors "github.com/alesplll/opens3-rebac/services/storage/internal/errors/domain_errors"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/logger"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func (s *storageService) InitiateMultipartUpload(ctx context.Context, expectedParts int32, contentType string) (string, error) {
	if expectedParts < 0 {
		return "", domainerrors.ErrInvalidParts
	}

	if contentType == "" {
		contentType = defaultContentType
	}

	uploadID := uuid.New().String()
	if err := s.repo.CreateMultipartSession(ctx, uploadID, expectedParts, contentType); err != nil {
		logger.Error(
			ctx,
			"failed to create multipart session",
			zap.Error(err),
			zap.String("upload_id", uploadID),
			zap.Int32("expected_parts", expectedParts),
		)
		return "", err
	}

	logger.Info(
		ctx,
		"initiated multipart upload",
		zap.String("upload_id", uploadID),
		zap.Int32("expected_parts", expectedParts),
		zap.String("content_type", contentType),
	)
	return uploadID, nil
}
