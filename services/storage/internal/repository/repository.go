package repository

import (
	"context"
	"io"

	"github.com/alesplll/opens3-rebac/services/storage/internal/model"
)

type StorageRepository interface {
	StoreBlob(ctx context.Context, reader io.Reader) (*model.BlobMeta, error)
	RetrieveBlob(ctx context.Context, blobID string, rangeStart, rangeEnd int64) (io.ReadCloser, int64, error)
	DeleteBlob(ctx context.Context, blobID string) error
	HealthCheck(ctx context.Context) error
}
