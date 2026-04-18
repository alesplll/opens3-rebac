package gateway

import (
	"context"
	"io"
	"strings"
	"testing"

	grpcclient "github.com/alesplll/opens3-rebac/services/gateway/internal/client/grpc"
	domainerrors "github.com/alesplll/opens3-rebac/services/gateway/internal/errors/domain_errors"
	"github.com/alesplll/opens3-rebac/services/gateway/internal/service"
	authzv1 "github.com/alesplll/opens3-rebac/shared/pkg/go/authz/v1"
	metadatav1 "github.com/alesplll/opens3-rebac/shared/pkg/go/metadata/v1"
	storagev1 "github.com/alesplll/opens3-rebac/shared/pkg/go/storage/v1"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type stubAuthZClient struct {
	checkFn       func(ctx context.Context, req *authzv1.CheckRequest) (*authzv1.CheckResponse, error)
	writeTupleFn  func(ctx context.Context, req *authzv1.WriteTupleRequest) (*authzv1.WriteTupleResponse, error)
	healthCheckFn func(ctx context.Context, req *authzv1.HealthCheckRequest) (*authzv1.HealthCheckResponse, error)
}

func (s *stubAuthZClient) Check(ctx context.Context, req *authzv1.CheckRequest) (*authzv1.CheckResponse, error) {
	if s.checkFn == nil {
		panic("unexpected call")
	}
	return s.checkFn(ctx, req)
}

func (s *stubAuthZClient) WriteTuple(ctx context.Context, req *authzv1.WriteTupleRequest) (*authzv1.WriteTupleResponse, error) {
	if s.writeTupleFn == nil {
		panic("unexpected call")
	}
	return s.writeTupleFn(ctx, req)
}

func (s *stubAuthZClient) HealthCheck(ctx context.Context, req *authzv1.HealthCheckRequest) (*authzv1.HealthCheckResponse, error) {
	if s.healthCheckFn == nil {
		panic("unexpected call")
	}
	return s.healthCheckFn(ctx, req)
}

type stubMetadataClient struct {
	createObjectVersionFn func(ctx context.Context, req *metadatav1.CreateObjectVersionRequest) (*metadatav1.CreateObjectVersionResponse, error)
	deleteObjectMetaFn    func(ctx context.Context, req *metadatav1.DeleteObjectMetaRequest) (*metadatav1.DeleteObjectMetaResponse, error)
	createBucketFn        func(ctx context.Context, req *metadatav1.CreateBucketRequest) (*metadatav1.CreateBucketResponse, error)
	deleteBucketFn        func(ctx context.Context, req *metadatav1.DeleteBucketRequest) (*metadatav1.DeleteBucketResponse, error)
	listBucketsFn         func(ctx context.Context, req *metadatav1.ListBucketsRequest) (*metadatav1.ListBucketsResponse, error)
	healthCheckFn         func(ctx context.Context, req *metadatav1.HealthCheckRequest) (*metadatav1.HealthCheckResponse, error)
}

func (s *stubMetadataClient) CreateBucket(ctx context.Context, req *metadatav1.CreateBucketRequest) (*metadatav1.CreateBucketResponse, error) {
	if s.createBucketFn == nil {
		panic("unexpected call")
	}
	return s.createBucketFn(ctx, req)
}

func (s *stubMetadataClient) DeleteBucket(ctx context.Context, req *metadatav1.DeleteBucketRequest) (*metadatav1.DeleteBucketResponse, error) {
	if s.deleteBucketFn == nil {
		panic("unexpected call")
	}
	return s.deleteBucketFn(ctx, req)
}

func (s *stubMetadataClient) ListBuckets(ctx context.Context, req *metadatav1.ListBucketsRequest) (*metadatav1.ListBucketsResponse, error) {
	if s.listBucketsFn == nil {
		panic("unexpected call")
	}
	return s.listBucketsFn(ctx, req)
}

func (s *stubMetadataClient) HeadBucket(context.Context, *metadatav1.HeadBucketRequest) (*metadatav1.HeadBucketResponse, error) {
	panic("unexpected call")
}

func (s *stubMetadataClient) CreateObjectVersion(ctx context.Context, req *metadatav1.CreateObjectVersionRequest) (*metadatav1.CreateObjectVersionResponse, error) {
	if s.createObjectVersionFn == nil {
		panic("unexpected call")
	}
	return s.createObjectVersionFn(ctx, req)
}

func (s *stubMetadataClient) GetObjectMeta(context.Context, *metadatav1.GetObjectMetaRequest) (*metadatav1.GetObjectMetaResponse, error) {
	panic("unexpected call")
}

func (s *stubMetadataClient) DeleteObjectMeta(ctx context.Context, req *metadatav1.DeleteObjectMetaRequest) (*metadatav1.DeleteObjectMetaResponse, error) {
	if s.deleteObjectMetaFn == nil {
		panic("unexpected call")
	}
	return s.deleteObjectMetaFn(ctx, req)
}

func (s *stubMetadataClient) ListObjects(context.Context, *metadatav1.ListObjectsRequest) (*metadatav1.ListObjectsResponse, error) {
	panic("unexpected call")
}

func (s *stubMetadataClient) HealthCheck(ctx context.Context, req *metadatav1.HealthCheckRequest) (*metadatav1.HealthCheckResponse, error) {
	if s.healthCheckFn == nil {
		panic("unexpected call")
	}
	return s.healthCheckFn(ctx, req)
}

type stubStorageClient struct {
	storeObjectFn             func(ctx context.Context, chunks <-chan *storagev1.StoreObjectRequest) (*storagev1.StoreObjectResponse, error)
	deleteObjectFn            func(ctx context.Context, req *storagev1.DeleteObjectRequest) (*storagev1.DeleteObjectResponse, error)
	completeMultipartUploadFn func(ctx context.Context, req *storagev1.CompleteMultipartUploadRequest) (*storagev1.CompleteMultipartUploadResponse, error)
	healthCheckFn             func(ctx context.Context, req *storagev1.HealthCheckRequest) (*storagev1.HealthCheckResponse, error)
}

func (s *stubStorageClient) StoreObject(ctx context.Context, chunks <-chan *storagev1.StoreObjectRequest) (*storagev1.StoreObjectResponse, error) {
	if s.storeObjectFn == nil {
		panic("unexpected call")
	}
	return s.storeObjectFn(ctx, chunks)
}

func (s *stubStorageClient) RetrieveObject(context.Context, *storagev1.RetrieveObjectRequest, io.Writer) (*storagev1.RetrieveObjectResponse, error) {
	panic("unexpected call")
}

func (s *stubStorageClient) DeleteObject(ctx context.Context, req *storagev1.DeleteObjectRequest) (*storagev1.DeleteObjectResponse, error) {
	if s.deleteObjectFn == nil {
		panic("unexpected call")
	}
	return s.deleteObjectFn(ctx, req)
}

func (s *stubStorageClient) InitiateMultipartUpload(context.Context, *storagev1.InitiateMultipartUploadRequest) (*storagev1.InitiateMultipartUploadResponse, error) {
	panic("unexpected call")
}

func (s *stubStorageClient) UploadPart(context.Context, <-chan *storagev1.UploadPartRequest) (*storagev1.UploadPartResponse, error) {
	panic("unexpected call")
}

func (s *stubStorageClient) CompleteMultipartUpload(ctx context.Context, req *storagev1.CompleteMultipartUploadRequest) (*storagev1.CompleteMultipartUploadResponse, error) {
	if s.completeMultipartUploadFn == nil {
		panic("unexpected call")
	}
	return s.completeMultipartUploadFn(ctx, req)
}

func (s *stubStorageClient) AbortMultipartUpload(context.Context, *storagev1.AbortMultipartUploadRequest) (*storagev1.AbortMultipartUploadResponse, error) {
	panic("unexpected call")
}

func (s *stubStorageClient) HealthCheck(ctx context.Context, req *storagev1.HealthCheckRequest) (*storagev1.HealthCheckResponse, error) {
	if s.healthCheckFn == nil {
		panic("unexpected call")
	}
	return s.healthCheckFn(ctx, req)
}

var _ grpcclient.AuthZClient = (*stubAuthZClient)(nil)
var _ grpcclient.MetadataClient = (*stubMetadataClient)(nil)
var _ grpcclient.StorageClient = (*stubStorageClient)(nil)

func TestPutObjectReturnsServiceUnavailableWhenMetadataIsUnimplemented(t *testing.T) {
	t.Parallel()

	svc := NewService(
		&stubAuthZClient{
			checkFn: func(context.Context, *authzv1.CheckRequest) (*authzv1.CheckResponse, error) {
				return &authzv1.CheckResponse{Allowed: true}, nil
			},
		},
		&stubMetadataClient{
			createObjectVersionFn: func(context.Context, *metadatav1.CreateObjectVersionRequest) (*metadatav1.CreateObjectVersionResponse, error) {
				return nil, status.Error(codes.Unimplemented, "metadata stub")
			},
		},
		&stubStorageClient{
			storeObjectFn: func(ctx context.Context, chunks <-chan *storagev1.StoreObjectRequest) (*storagev1.StoreObjectResponse, error) {
				var sentData []string
				for chunk := range chunks {
					switch payload := chunk.GetPayload().(type) {
					case *storagev1.StoreObjectRequest_Header:
						sentData = append(sentData, string(payload.Header.GetData()))
					case *storagev1.StoreObjectRequest_Chunk:
						sentData = append(sentData, string(payload.Chunk.GetData()))
					}
				}
				require.Equal(t, []string{"payload"}, sentData)
				return &storagev1.StoreObjectResponse{BlobId: "blob-1", ChecksumMd5: "etag-1"}, nil
			},
		},
	)

	resp, err := svc.PutObject(context.Background(), service.PutObjectRequest{
		UserID:      "user-1",
		Bucket:      "bucket-1",
		Key:         "key-1",
		Body:        strings.NewReader("payload"),
		Size:        int64(len("payload")),
		ContentType: "text/plain",
	})

	require.Nil(t, resp)
	require.ErrorIs(t, err, domainerrors.ErrServiceUnavailable)
}

func TestDeleteObjectDeletesBlobFromStorage(t *testing.T) {
	t.Parallel()

	var deletedBlobID string
	svc := NewService(
		&stubAuthZClient{
			checkFn: func(context.Context, *authzv1.CheckRequest) (*authzv1.CheckResponse, error) {
				return &authzv1.CheckResponse{Allowed: true}, nil
			},
		},
		&stubMetadataClient{
			deleteObjectMetaFn: func(ctx context.Context, req *metadatav1.DeleteObjectMetaRequest) (*metadatav1.DeleteObjectMetaResponse, error) {
				require.Equal(t, "bucket-1", req.GetBucketName())
				require.Equal(t, "key-1", req.GetKey())
				return &metadatav1.DeleteObjectMetaResponse{BlobId: "blob-42", Success: true}, nil
			},
		},
		&stubStorageClient{
			deleteObjectFn: func(ctx context.Context, req *storagev1.DeleteObjectRequest) (*storagev1.DeleteObjectResponse, error) {
				deletedBlobID = req.GetBlobId()
				return &storagev1.DeleteObjectResponse{Success: true}, nil
			},
		},
	)

	err := svc.DeleteObject(context.Background(), service.DeleteObjectRequest{
		UserID: "user-1",
		Bucket: "bucket-1",
		Key:    "key-1",
	})

	require.NoError(t, err)
	require.Equal(t, "blob-42", deletedBlobID)
}

func TestCreateBucketRollsBackMetadataOnAuthzFailure(t *testing.T) {
	t.Parallel()

	var rolledBack bool
	svc := NewService(
		&stubAuthZClient{
			writeTupleFn: func(context.Context, *authzv1.WriteTupleRequest) (*authzv1.WriteTupleResponse, error) {
				return nil, status.Error(codes.Unavailable, "authz down")
			},
		},
		&stubMetadataClient{
			createBucketFn: func(ctx context.Context, req *metadatav1.CreateBucketRequest) (*metadatav1.CreateBucketResponse, error) {
				require.Equal(t, "bucket-1", req.GetName())
				require.Equal(t, "user-1", req.GetOwnerId())
				return &metadatav1.CreateBucketResponse{BucketId: "bucket-id-1"}, nil
			},
			deleteBucketFn: func(ctx context.Context, req *metadatav1.DeleteBucketRequest) (*metadatav1.DeleteBucketResponse, error) {
				rolledBack = true
				require.Equal(t, "bucket-1", req.GetBucketName())
				return &metadatav1.DeleteBucketResponse{Success: true}, nil
			},
		},
		&stubStorageClient{},
	)

	resp, err := svc.CreateBucket(context.Background(), service.CreateBucketRequest{
		UserID: "user-1",
		Bucket: "bucket-1",
	})

	require.Nil(t, resp)
	require.ErrorIs(t, err, domainerrors.ErrServiceUnavailable)
	require.True(t, rolledBack)
}

func TestCompleteMultipartUploadReturnsServiceUnavailableWhenMetadataIsUnimplemented(t *testing.T) {
	t.Parallel()

	svc := NewService(
		&stubAuthZClient{
			checkFn: func(context.Context, *authzv1.CheckRequest) (*authzv1.CheckResponse, error) {
				return &authzv1.CheckResponse{Allowed: true}, nil
			},
		},
		&stubMetadataClient{
			createObjectVersionFn: func(context.Context, *metadatav1.CreateObjectVersionRequest) (*metadatav1.CreateObjectVersionResponse, error) {
				return nil, status.Error(codes.Unimplemented, "metadata stub")
			},
		},
		&stubStorageClient{
			completeMultipartUploadFn: func(ctx context.Context, req *storagev1.CompleteMultipartUploadRequest) (*storagev1.CompleteMultipartUploadResponse, error) {
				require.Equal(t, "upload-1", req.GetUploadId())
				return &storagev1.CompleteMultipartUploadResponse{BlobId: "blob-1", ChecksumMd5: "etag-1"}, nil
			},
		},
	)

	resp, err := svc.CompleteMultipartUpload(context.Background(), service.CompleteMultipartUploadRequest{
		UserID:   "user-1",
		Bucket:   "bucket-1",
		Key:      "key-1",
		UploadID: "upload-1",
		Parts: []service.CompletedPart{
			{PartNumber: 1, ETag: "\"etag-part-1\""},
		},
	})

	require.Nil(t, resp)
	require.ErrorIs(t, err, domainerrors.ErrServiceUnavailable)
}

func TestListBucketsDoesNotCallAuthz(t *testing.T) {
	t.Parallel()

	svc := NewService(
		&stubAuthZClient{},
		&stubMetadataClient{
			listBucketsFn: func(ctx context.Context, req *metadatav1.ListBucketsRequest) (*metadatav1.ListBucketsResponse, error) {
				require.Equal(t, "user-1", req.GetOwnerId())
				return &metadatav1.ListBucketsResponse{
					Buckets: []*metadatav1.BucketInfo{
						{Name: "bucket-1", CreatedAt: 1700000000000},
					},
				}, nil
			},
		},
		&stubStorageClient{},
	)

	resp, err := svc.ListBuckets(context.Background(), service.ListBucketsRequest{UserID: "user-1"})

	require.NoError(t, err)
	require.Len(t, resp.Buckets, 1)
	require.Equal(t, "bucket-1", resp.Buckets[0].Name)
}

func TestReadyFailsWhenDependencyIsNotServing(t *testing.T) {
	t.Parallel()

	svc := NewService(
		&stubAuthZClient{
			healthCheckFn: func(ctx context.Context, req *authzv1.HealthCheckRequest) (*authzv1.HealthCheckResponse, error) {
				return &authzv1.HealthCheckResponse{Status: authzv1.HealthCheckResponse_SERVING}, nil
			},
		},
		&stubMetadataClient{
			healthCheckFn: func(ctx context.Context, req *metadatav1.HealthCheckRequest) (*metadatav1.HealthCheckResponse, error) {
				return &metadatav1.HealthCheckResponse{Status: metadatav1.HealthCheckResponse_NOT_SERVING}, nil
			},
		},
		&stubStorageClient{
			healthCheckFn: func(ctx context.Context, req *storagev1.HealthCheckRequest) (*storagev1.HealthCheckResponse, error) {
				return &storagev1.HealthCheckResponse{Status: storagev1.HealthCheckResponse_SERVING}, nil
			},
		},
	)

	err := svc.Ready(context.Background())

	require.ErrorIs(t, err, domainerrors.ErrServiceUnavailable)
}
