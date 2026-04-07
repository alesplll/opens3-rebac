package storage

import (
	"context"

	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/logger"
	"go.uber.org/zap"
)

func (s *storageService) AbortMultipartUpload(ctx context.Context, uploadID string) error {
	if err := s.repo.CleanupMultipart(ctx, uploadID); err != nil {
		logger.Error(ctx, "failed to abort multipart upload", zap.Error(err), zap.String("upload_id", uploadID))
		return err
	}

	logger.Info(ctx, "aborted multipart upload", zap.String("upload_id", uploadID))
	return nil
}
