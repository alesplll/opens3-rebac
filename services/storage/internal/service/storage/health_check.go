package storage

import (
	"context"

	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/logger"
	"go.uber.org/zap"
)

func (s *storageService) HealthCheck(ctx context.Context, _ string) (bool, error) {
	if err := s.repo.HealthCheck(ctx); err != nil {
		logger.Error(ctx, "storage health check failed", zap.Error(err))
		return false, err
	}
	return true, nil
}
