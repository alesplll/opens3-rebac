package grpc

import (
	"context"
	"io"

	authzv1 "github.com/alesplll/opens3-rebac/shared/pkg/go/authz/v1"
	metadatav1 "github.com/alesplll/opens3-rebac/shared/pkg/go/metadata/v1"
	storagev1 "github.com/alesplll/opens3-rebac/shared/pkg/go/storage/v1"
)

type AuthZClient interface {
	Check(ctx context.Context, req *authzv1.CheckRequest) (*authzv1.CheckResponse, error)
	WriteTuple(ctx context.Context, req *authzv1.WriteTupleRequest) (*authzv1.WriteTupleResponse, error)
	HealthCheck(ctx context.Context, req *authzv1.HealthCheckRequest) (*authzv1.HealthCheckResponse, error)
}

type MetadataClient interface {
	CreateBucket(ctx context.Context, req *metadatav1.CreateBucketRequest) (*metadatav1.CreateBucketResponse, error)
	DeleteBucket(ctx context.Context, req *metadatav1.DeleteBucketRequest) (*metadatav1.DeleteBucketResponse, error)
	ListBuckets(ctx context.Context, req *metadatav1.ListBucketsRequest) (*metadatav1.ListBucketsResponse, error)
	HeadBucket(ctx context.Context, req *metadatav1.HeadBucketRequest) (*metadatav1.HeadBucketResponse, error)
	CreateObjectVersion(ctx context.Context, req *metadatav1.CreateObjectVersionRequest) (*metadatav1.CreateObjectVersionResponse, error)
	GetObjectMeta(ctx context.Context, req *metadatav1.GetObjectMetaRequest) (*metadatav1.GetObjectMetaResponse, error)
	DeleteObjectMeta(ctx context.Context, req *metadatav1.DeleteObjectMetaRequest) (*metadatav1.DeleteObjectMetaResponse, error)
	ListObjects(ctx context.Context, req *metadatav1.ListObjectsRequest) (*metadatav1.ListObjectsResponse, error)
	HealthCheck(ctx context.Context, req *metadatav1.HealthCheckRequest) (*metadatav1.HealthCheckResponse, error)
}

type StorageClient interface {
	StoreObject(ctx context.Context, chunks <-chan *storagev1.StoreObjectRequest) (*storagev1.StoreObjectResponse, error)
	RetrieveObject(ctx context.Context, req *storagev1.RetrieveObjectRequest, writer io.Writer) (*storagev1.RetrieveObjectResponse, error)
	DeleteObject(ctx context.Context, req *storagev1.DeleteObjectRequest) (*storagev1.DeleteObjectResponse, error)
	InitiateMultipartUpload(ctx context.Context, req *storagev1.InitiateMultipartUploadRequest) (*storagev1.InitiateMultipartUploadResponse, error)
	UploadPart(ctx context.Context, chunks <-chan *storagev1.UploadPartRequest) (*storagev1.UploadPartResponse, error)
	CompleteMultipartUpload(ctx context.Context, req *storagev1.CompleteMultipartUploadRequest) (*storagev1.CompleteMultipartUploadResponse, error)
	AbortMultipartUpload(ctx context.Context, req *storagev1.AbortMultipartUploadRequest) (*storagev1.AbortMultipartUploadResponse, error)
	HealthCheck(ctx context.Context, req *storagev1.HealthCheckRequest) (*storagev1.HealthCheckResponse, error)
}
