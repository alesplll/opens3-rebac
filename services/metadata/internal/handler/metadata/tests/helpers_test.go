package tests

import (
	"context"
	"time"

	metadatahandler "github.com/alesplll/opens3-rebac/services/metadata/internal/handler/metadata"
	"github.com/alesplll/opens3-rebac/services/metadata/internal/model"
	"github.com/alesplll/opens3-rebac/services/metadata/internal/service"
	metadatav1 "github.com/alesplll/opens3-rebac/shared/pkg/go/metadata/v1"
)

type bucketServiceStub struct {
	createBucketFunc func(ctx context.Context, name, ownerID string) (*model.Bucket, error)
	deleteBucketFunc func(ctx context.Context, name string) error
	getBucketFunc    func(ctx context.Context, name string) (*model.Bucket, error)
	listBucketsFunc  func(ctx context.Context, ownerID string) ([]*model.Bucket, error)
	headBucketFunc   func(ctx context.Context, name string) (bool, string, string, error)
}

func (s *bucketServiceStub) CreateBucket(ctx context.Context, name, ownerID string) (*model.Bucket, error) {
	return s.createBucketFunc(ctx, name, ownerID)
}

func (s *bucketServiceStub) DeleteBucket(ctx context.Context, name string) error {
	return s.deleteBucketFunc(ctx, name)
}

func (s *bucketServiceStub) GetBucket(ctx context.Context, name string) (*model.Bucket, error) {
	return s.getBucketFunc(ctx, name)
}

func (s *bucketServiceStub) ListBuckets(ctx context.Context, ownerID string) ([]*model.Bucket, error) {
	return s.listBucketsFunc(ctx, ownerID)
}

func (s *bucketServiceStub) HeadBucket(ctx context.Context, name string) (bool, string, string, error) {
	return s.headBucketFunc(ctx, name)
}

type objectServiceStub struct {
	createObjectVersionFunc func(ctx context.Context, bucketName, key, blobID string, sizeBytes int64, etag, contentType string) (string, string, time.Time, error)
	getObjectMetaFunc       func(ctx context.Context, bucketName, key, versionID string) (*model.ObjectMeta, error)
	deleteObjectMetaFunc    func(ctx context.Context, bucketName, key string) (string, string, error)
	listObjectsFunc         func(ctx context.Context, bucketName, prefix, continuationToken string, maxKeys int32) ([]*model.ObjectListItem, string, bool, error)
	healthCheckFunc         func(ctx context.Context) (bool, bool)
}

func (s *objectServiceStub) CreateObjectVersion(ctx context.Context, bucketName, key, blobID string, sizeBytes int64, etag, contentType string) (string, string, time.Time, error) {
	return s.createObjectVersionFunc(ctx, bucketName, key, blobID, sizeBytes, etag, contentType)
}

func (s *objectServiceStub) GetObjectMeta(ctx context.Context, bucketName, key, versionID string) (*model.ObjectMeta, error) {
	return s.getObjectMetaFunc(ctx, bucketName, key, versionID)
}

func (s *objectServiceStub) DeleteObjectMeta(ctx context.Context, bucketName, key string) (string, string, error) {
	return s.deleteObjectMetaFunc(ctx, bucketName, key)
}

func (s *objectServiceStub) ListObjects(ctx context.Context, bucketName, prefix, continuationToken string, maxKeys int32) ([]*model.ObjectListItem, string, bool, error) {
	return s.listObjectsFunc(ctx, bucketName, prefix, continuationToken, maxKeys)
}

func (s *objectServiceStub) HealthCheck(ctx context.Context) (bool, bool) {
	return s.healthCheckFunc(ctx)
}

func newHandler(bucketService service.BucketService, objectService service.ObjectService) metadatav1.MetadataServiceServer {
	return metadatahandler.NewHandler(bucketService, objectService)
}
