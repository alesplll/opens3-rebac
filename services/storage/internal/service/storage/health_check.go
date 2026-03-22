package storage

import (
	"context"
)

func (s *storageService) HealthCheck(ctx context.Context, _ string) (bool, error) {
	if err := s.repo.HealthCheck(ctx); err != nil {
		return false, err
	}
	return true, nil
}
