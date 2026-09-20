package object

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/alesplll/opens3-rebac/services/gateway/internal/client/metadata"
	storageclient "github.com/alesplll/opens3-rebac/services/gateway/internal/client/storage"
	"github.com/alesplll/opens3-rebac/services/gateway/internal/service"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/logger"
	"go.uber.org/zap"
)

type objectService struct {
	metadata metadata.Client
	storage  storageclient.Client
}

func NewService(metadataClient metadata.Client, storageClient storageclient.Client) service.ObjectService {
	return &objectService{metadata: metadataClient, storage: storageClient}
}

func (s *objectService) Put(ctx context.Context, bucket, key string, body io.Reader, size *int64, contentType string) (service.PutResult, error) {
	exists, err := s.metadata.HeadBucket(ctx, bucket)
	if err != nil {
		return service.PutResult{}, err
	}
	if !exists {
		return service.PutResult{}, service.ErrBucketNotFound
	}

	stored, err := s.storage.Store(ctx, body, size, contentType)
	if err != nil {
		var bodyReadErr *storageclient.BodyReadError
		if errors.As(err, &bodyReadErr) {
			return service.PutResult{}, fmt.Errorf("%w: %v", service.ErrBodyRead, bodyReadErr.Err)
		}
		return service.PutResult{}, err
	}
	versionID, err := s.metadata.CreateVersion(ctx, bucket, key, stored.BlobID, stored.Size, stored.ETag, contentType)
	if err != nil {
		logger.Error(ctx, "metadata registration failed after blob commit",
			zap.String("blob_id", stored.BlobID), zap.String("bucket", bucket), zap.String("key", key), zap.Error(err))
		return service.PutResult{}, err
	}
	return service.PutResult{ETag: stored.ETag, VersionID: versionID}, nil
}

func (s *objectService) Get(ctx context.Context, bucket, key string) (service.GetResult, error) {
	meta, err := s.metadata.GetObject(ctx, bucket, key)
	if err != nil {
		return service.GetResult{}, err
	}
	body, err := s.storage.Retrieve(ctx, meta.BlobID, meta.Size)
	if err != nil {
		if errors.Is(err, storageclient.ErrEmptyStream) {
			return service.GetResult{}, service.ErrEmptyStorageStream
		}
		return service.GetResult{}, err
	}
	return service.GetResult{
		Body: body, Size: meta.Size, ETag: meta.ETag,
		VersionID: meta.VersionID, ContentType: meta.ContentType,
	}, nil
}
