package service

import (
	"context"
	"io"

	"github.com/alesplll/opens3-rebac/services/storage/internal/model"
)

type StorageService interface {
	StoreObject(ctx context.Context, reader io.Reader, size int64, contentType string) (*model.BlobMeta, error)
	RetrieveObject(ctx context.Context, blobID string, offset, length int64) (io.ReadCloser, int64, error)
	DeleteObject(ctx context.Context, blobID string) error

	InitiateMultipartUpload(ctx context.Context, expectedParts int32, contentType string) (string, error)
	UploadPart(ctx context.Context, uploadID string, partNumber int32, reader io.Reader) (string, error)
	CompleteMultipartUpload(ctx context.Context, uploadID string, parts []model.PartInfo) (*model.BlobMeta, error)
	AbortMultipartUpload(ctx context.Context, uploadID string) error

	HealthCheck(ctx context.Context, serviceName string) (bool, error)
}
