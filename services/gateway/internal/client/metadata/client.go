package metadata

import (
	"context"

	metadatav1 "github.com/alesplll/opens3-rebac/shared/pkg/go/metadata/v1"
)

type ObjectMeta struct {
	BlobID      string
	VersionID   string
	Size        int64
	ETag        string
	ContentType string
}

type Client interface {
	HeadBucket(ctx context.Context, bucket string) (bool, error)
	CreateVersion(ctx context.Context, bucket, key, blobID string, size int64, etag, contentType string) (string, error)
	GetObject(ctx context.Context, bucket, key string) (ObjectMeta, error)
}

type client struct {
	grpc metadatav1.MetadataServiceClient
}

func NewClient(grpcClient metadatav1.MetadataServiceClient) Client {
	return &client{grpc: grpcClient}
}

func (c *client) HeadBucket(ctx context.Context, bucket string) (bool, error) {
	response, err := c.grpc.HeadBucket(ctx, &metadatav1.HeadBucketRequest{BucketName: bucket})
	if err != nil {
		return false, err
	}
	return response.GetExists(), nil
}

func (c *client) CreateVersion(ctx context.Context, bucket, key, blobID string, size int64, etag, contentType string) (string, error) {
	response, err := c.grpc.CreateObjectVersion(ctx, &metadatav1.CreateObjectVersionRequest{
		BucketName:  bucket,
		Key:         key,
		BlobId:      blobID,
		SizeBytes:   size,
		Etag:        etag,
		ContentType: contentType,
	})
	if err != nil {
		return "", err
	}
	return response.GetVersionId(), nil
}

func (c *client) GetObject(ctx context.Context, bucket, key string) (ObjectMeta, error) {
	response, err := c.grpc.GetObjectMeta(ctx, &metadatav1.GetObjectMetaRequest{BucketName: bucket, Key: key})
	if err != nil {
		return ObjectMeta{}, err
	}
	return ObjectMeta{
		BlobID:      response.GetBlobId(),
		VersionID:   response.GetVersionId(),
		Size:        response.GetSizeBytes(),
		ETag:        response.GetEtag(),
		ContentType: response.GetContentType(),
	}, nil
}
