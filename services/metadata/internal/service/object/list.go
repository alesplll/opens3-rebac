package object

import (
	"context"

	"github.com/alesplll/opens3-rebac/services/metadata/internal/model"
)

func (s *objectService) ListObjects(ctx context.Context, bucketName, prefix, continuationToken string, maxKeys int32) ([]*model.ObjectListItem, string, bool, error) {
	return s.repo.List(ctx, bucketName, prefix, continuationToken, maxKeys)
}
