package metadata

import (
	"context"
	"time"

	grpcclient "github.com/alesplll/opens3-rebac/services/gateway/internal/client/grpc"
	metadatav1 "github.com/alesplll/opens3-rebac/shared/pkg/go/metadata/v1"
)

type client struct {
	client  metadatav1.MetadataServiceClient
	timeout time.Duration
}

func NewClient(grpcSvc metadatav1.MetadataServiceClient, timeout time.Duration) grpcclient.MetadataClient {
	return &client{client: grpcSvc, timeout: timeout}
}

func (c *client) CreateBucket(ctx context.Context, req *metadatav1.CreateBucketRequest) (*metadatav1.CreateBucketResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	return c.client.CreateBucket(ctx, req)
}

func (c *client) DeleteBucket(ctx context.Context, req *metadatav1.DeleteBucketRequest) (*metadatav1.DeleteBucketResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	return c.client.DeleteBucket(ctx, req)
}

func (c *client) ListBuckets(ctx context.Context, req *metadatav1.ListBucketsRequest) (*metadatav1.ListBucketsResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	return c.client.ListBuckets(ctx, req)
}

func (c *client) HeadBucket(ctx context.Context, req *metadatav1.HeadBucketRequest) (*metadatav1.HeadBucketResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	return c.client.HeadBucket(ctx, req)
}

func (c *client) CreateObjectVersion(ctx context.Context, req *metadatav1.CreateObjectVersionRequest) (*metadatav1.CreateObjectVersionResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	return c.client.CreateObjectVersion(ctx, req)
}

func (c *client) GetObjectMeta(ctx context.Context, req *metadatav1.GetObjectMetaRequest) (*metadatav1.GetObjectMetaResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	return c.client.GetObjectMeta(ctx, req)
}

func (c *client) DeleteObjectMeta(ctx context.Context, req *metadatav1.DeleteObjectMetaRequest) (*metadatav1.DeleteObjectMetaResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	return c.client.DeleteObjectMeta(ctx, req)
}

func (c *client) ListObjects(ctx context.Context, req *metadatav1.ListObjectsRequest) (*metadatav1.ListObjectsResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	return c.client.ListObjects(ctx, req)
}

func (c *client) HealthCheck(ctx context.Context, req *metadatav1.HealthCheckRequest) (*metadatav1.HealthCheckResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	return c.client.HealthCheck(ctx, req)
}
