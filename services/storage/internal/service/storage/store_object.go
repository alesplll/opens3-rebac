package storage

import (
	"context"
	"io"

	"github.com/alesplll/opens3-rebac/services/storage/internal/model"
)

func (s *storageService) StoreObject(ctx context.Context, reader io.Reader, _ int64, contentType string) (*model.BlobMeta, error) {
	meta, err := s.repo.StoreBlob(ctx, reader)
	if err != nil {
		return nil, err
	}

	meta.ContentType = contentType
	return meta, nil
}
