package bucket

import (
	"context"

	"github.com/alesplll/opens3-rebac/services/metadata/internal/model"
)

func (s *bucketService) ListBuckets(ctx context.Context, ownerID string) ([]*model.Bucket, error) {
	return s.repo.List(ctx, ownerID)
}
