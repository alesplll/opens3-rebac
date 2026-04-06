package repository

import (
	"context"
	"io"

	"github.com/alesplll/opens3-rebac/services/storage/internal/model"
)

type StorageRepository interface {
	StoreBlob(ctx context.Context, reader io.Reader) (*model.BlobMeta, error)
	RetrieveBlob(ctx context.Context, blobID string) (io.ReadCloser, int64, error)
	RetrieveBlobRange(ctx context.Context, blobID string, offset, length int64) (io.ReadCloser, int64, error)
	DeleteBlob(ctx context.Context, blobID string) error
	HealthCheck(ctx context.Context) error

	// TODO(phase-2 multipart):
	// CreateMultipartSession(ctx context.Context, uploadID string, expectedParts int32) error
	// StorePart(ctx context.Context, uploadID string, partNumber int32, reader io.Reader) (checksumMD5 string, err error)
	// AssembleParts(ctx context.Context, uploadID string, parts []model.PartInfo, destBlobID string) (*model.BlobMeta, error)
	// CleanupMultipart(ctx context.Context, uploadID string) error
}
