package bucket

import (
	"context"

	"github.com/alesplll/opens3-rebac/services/metadata/internal/model"
)

func (s *bucketService) GetBucket(ctx context.Context, name string) (*model.Bucket, error) {
	return s.repo.Get(ctx, name)
}
