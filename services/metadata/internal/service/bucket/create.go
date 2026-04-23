package bucket

import (
	"context"

	"github.com/alesplll/opens3-rebac/services/metadata/internal/model"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/sys/validate"
)

func (s *bucketService) CreateBucket(ctx context.Context, name, ownerID string) (*model.Bucket, error) {
	if err := validate.Validate(ctx,
		func(ctx context.Context) error {
			if name == "" {
				return validate.NewValidationErrors("bucket name is required")
			}
			return nil
		},
		func(ctx context.Context) error {
			if ownerID == "" {
				return validate.NewValidationErrors("owner_id is required")
			}
			return nil
		},
	); err != nil {
		return nil, err
	}

	return s.repo.Create(ctx, name, ownerID)
}
