package storage

import (
	"context"
	"io"

	"github.com/alesplll/opens3-rebac/services/storage/internal/model"
)

func (s *storageService) StoreObject(ctx context.Context, reader io.Reader, _ int64, _ string) (*model.BlobMeta, error) {
	return s.repo.StoreBlob(ctx, reader)
}
