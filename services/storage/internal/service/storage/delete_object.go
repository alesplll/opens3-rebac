package storage

import (
	"context"

	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/logger"
	"go.uber.org/zap"
)

func (s *storageService) DeleteObject(ctx context.Context, blobID string) error {
	if err := s.repo.DeleteBlob(ctx, blobID); err != nil {
		logger.Error(ctx, "failed to delete blob", zap.Error(err), zap.String("blob_id", blobID))
		return err
	}

	return nil
}
