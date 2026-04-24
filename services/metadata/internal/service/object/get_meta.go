package object

import (
	"context"

	"github.com/alesplll/opens3-rebac/services/metadata/internal/model"
)

func (s *objectService) GetObjectMeta(ctx context.Context, bucketName, key, versionID string) (*model.ObjectMeta, error) {
	return s.repo.GetMeta(ctx, bucketName, key, versionID)
}
