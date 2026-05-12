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
	quotav1 "github.com/alesplll/opens3-rebac/shared/pkg/go/quota/v1"
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
	createBucketFn        func(ctx context.Context, req *metadatav1.CreateBucketRequest) (*metadatav1.CreateBucketResponse, error)
	deleteBucketFn        func(ctx context.Context, req *metadatav1.DeleteBucketRequest) (*metadatav1.DeleteBucketResponse, error)
	listBucketsFn         func(ctx context.Context, req *metadatav1.ListBucketsRequest) (*metadatav1.ListBucketsResponse, error)
	healthCheckFn         func(ctx context.Context, req *metadatav1.HealthCheckRequest) (*metadatav1.HealthCheckResponse, error)
	createObjectVersionFn func(ctx context.Context, req *metadatav1.CreateObjectVersionRequest) (*metadatav1.CreateObjectVersionResponse, error)
	getObjectMetaFn       func(ctx context.Context, req *metadatav1.GetObjectMetaRequest) (*metadatav1.GetObjectMetaResponse, error)
	deleteObjectMetaFn    func(ctx context.Context, req *metadatav1.DeleteObjectMetaRequest) (*metadatav1.DeleteObjectMetaResponse, error)
	listObjectsFn         func(ctx context.Context, req *metadatav1.ListObjectsRequest) (*metadatav1.ListObjectsResponse, error)
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

func (s *stubMetadataClient) GetObjectMeta(ctx context.Context, req *metadatav1.GetObjectMetaRequest) (*metadatav1.GetObjectMetaResponse, error) {
	if s.getObjectMetaFn == nil {
		panic("unexpected call")
	}
	return s.getObjectMetaFn(ctx, req)
}

func (s *stubMetadataClient) DeleteObjectMeta(ctx context.Context, req *metadatav1.DeleteObjectMetaRequest) (*metadatav1.DeleteObjectMetaResponse, error) {
	if s.deleteObjectMetaFn == nil {
		panic("unexpected call")
	}
	return s.deleteObjectMetaFn(ctx, req)
}

func (s *stubMetadataClient) ListObjects(ctx context.Context, req *metadatav1.ListObjectsRequest) (*metadatav1.ListObjectsResponse, error) {
	if s.listObjectsFn == nil {
		panic("unexpected call")
	}
	return s.listObjectsFn(ctx, req)
}

func (s *stubMetadataClient) HealthCheck(ctx context.Context, req *metadatav1.HealthCheckRequest) (*metadatav1.HealthCheckResponse, error) {
	if s.healthCheckFn == nil {
		panic("unexpected call")
	}
	return s.healthCheckFn(ctx, req)
}

type stubQuotaClient struct {
	checkQuotaFn  func(ctx context.Context, req *quotav1.CheckQuotaRequest) (*quotav1.CheckQuotaResponse, error)
	updateUsageFn func(ctx context.Context, req *quotav1.UpdateUsageRequest) (*quotav1.UpdateUsageResponse, error)
	healthCheckFn func(ctx context.Context, req *quotav1.HealthCheckRequest) (*quotav1.HealthCheckResponse, error)
}

func (s *stubQuotaClient) CheckQuota(ctx context.Context, req *quotav1.CheckQuotaRequest) (*quotav1.CheckQuotaResponse, error) {
	if s.checkQuotaFn == nil {
		panic("unexpected call")
	}
	return s.checkQuotaFn(ctx, req)
}

func (s *stubQuotaClient) UpdateUsage(ctx context.Context, req *quotav1.UpdateUsageRequest) (*quotav1.UpdateUsageResponse, error) {
	if s.updateUsageFn == nil {
		panic("unexpected call")
	}
	return s.updateUsageFn(ctx, req)
}

func (s *stubQuotaClient) HealthCheck(ctx context.Context, req *quotav1.HealthCheckRequest) (*quotav1.HealthCheckResponse, error) {
	if s.healthCheckFn == nil {
		panic("unexpected call")
	}
	return s.healthCheckFn(ctx, req)
}

type stubStorageClient struct {
	storeObjectFn             func(ctx context.Context, chunks <-chan *storagev1.StoreObjectRequest) (*storagev1.StoreObjectResponse, error)
	retrieveObjectFn          func(ctx context.Context, req *storagev1.RetrieveObjectRequest, writer io.Writer) (*storagev1.RetrieveObjectResponse, error)
	deleteObjectFn            func(ctx context.Context, req *storagev1.DeleteObjectRequest) (*storagev1.DeleteObjectResponse, error)
	initiateMultipartUploadFn func(ctx context.Context, req *storagev1.InitiateMultipartUploadRequest) (*storagev1.InitiateMultipartUploadResponse, error)
	uploadPartFn              func(ctx context.Context, chunks <-chan *storagev1.UploadPartRequest) (*storagev1.UploadPartResponse, error)
	completeMultipartUploadFn func(ctx context.Context, req *storagev1.CompleteMultipartUploadRequest) (*storagev1.CompleteMultipartUploadResponse, error)
	abortMultipartUploadFn    func(ctx context.Context, req *storagev1.AbortMultipartUploadRequest) (*storagev1.AbortMultipartUploadResponse, error)
	healthCheckFn             func(ctx context.Context, req *storagev1.HealthCheckRequest) (*storagev1.HealthCheckResponse, error)
}

func (s *stubStorageClient) StoreObject(ctx context.Context, chunks <-chan *storagev1.StoreObjectRequest) (*storagev1.StoreObjectResponse, error) {
	if s.storeObjectFn == nil {
		panic("unexpected call")
	}
	return s.storeObjectFn(ctx, chunks)
}

func (s *stubStorageClient) RetrieveObject(ctx context.Context, req *storagev1.RetrieveObjectRequest, writer io.Writer) (*storagev1.RetrieveObjectResponse, error) {
	if s.retrieveObjectFn == nil {
		panic("unexpected call")
	}
	return s.retrieveObjectFn(ctx, req, writer)
}

func (s *stubStorageClient) DeleteObject(ctx context.Context, req *storagev1.DeleteObjectRequest) (*storagev1.DeleteObjectResponse, error) {
	if s.deleteObjectFn == nil {
		panic("unexpected call")
	}
	return s.deleteObjectFn(ctx, req)
}

func (s *stubStorageClient) InitiateMultipartUpload(ctx context.Context, req *storagev1.InitiateMultipartUploadRequest) (*storagev1.InitiateMultipartUploadResponse, error) {
	if s.initiateMultipartUploadFn == nil {
		panic("unexpected call")
	}
	return s.initiateMultipartUploadFn(ctx, req)
}

func (s *stubStorageClient) UploadPart(ctx context.Context, chunks <-chan *storagev1.UploadPartRequest) (*storagev1.UploadPartResponse, error) {
	if s.uploadPartFn == nil {
		panic("unexpected call")
	}
	return s.uploadPartFn(ctx, chunks)
}

func (s *stubStorageClient) CompleteMultipartUpload(ctx context.Context, req *storagev1.CompleteMultipartUploadRequest) (*storagev1.CompleteMultipartUploadResponse, error) {
	if s.completeMultipartUploadFn == nil {
		panic("unexpected call")
	}
	return s.completeMultipartUploadFn(ctx, req)
}

func (s *stubStorageClient) AbortMultipartUpload(ctx context.Context, req *storagev1.AbortMultipartUploadRequest) (*storagev1.AbortMultipartUploadResponse, error) {
	if s.abortMultipartUploadFn == nil {
		panic("unexpected call")
	}
	return s.abortMultipartUploadFn(ctx, req)
}

func (s *stubStorageClient) HealthCheck(ctx context.Context, req *storagev1.HealthCheckRequest) (*storagev1.HealthCheckResponse, error) {
	if s.healthCheckFn == nil {
		panic("unexpected call")
	}
	return s.healthCheckFn(ctx, req)
}

var _ grpcclient.AuthZClient = (*stubAuthZClient)(nil)
var _ grpcclient.MetadataClient = (*stubMetadataClient)(nil)
var _ grpcclient.QuotaClient = (*stubQuotaClient)(nil)
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
		&stubQuotaClient{
			checkQuotaFn: func(context.Context, *quotav1.CheckQuotaRequest) (*quotav1.CheckQuotaResponse, error) {
				return &quotav1.CheckQuotaResponse{Allowed: true}, nil
			},
			updateUsageFn: func(context.Context, *quotav1.UpdateUsageRequest) (*quotav1.UpdateUsageResponse, error) {
				return &quotav1.UpdateUsageResponse{}, nil
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
			deleteObjectFn: func(context.Context, *storagev1.DeleteObjectRequest) (*storagev1.DeleteObjectResponse, error) {
				return &storagev1.DeleteObjectResponse{Success: true}, nil
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

func TestPutObjectRollsBackQuotaWhenStorageFails(t *testing.T) {
	t.Parallel()

	var quotaRolledBack bool
	svc := NewService(
		&stubAuthZClient{
			checkFn: func(context.Context, *authzv1.CheckRequest) (*authzv1.CheckResponse, error) {
				return &authzv1.CheckResponse{Allowed: true}, nil
			},
		},
		&stubMetadataClient{},
		&stubQuotaClient{
			checkQuotaFn: func(ctx context.Context, req *quotav1.CheckQuotaRequest) (*quotav1.CheckQuotaResponse, error) {
				require.Equal(t, "user:user-1", req.GetSubjectId())
				require.Equal(t, "bucket:bucket-1", req.GetBucketId())
				require.Equal(t, int64(7), req.GetDelta().GetBytes())
				require.Equal(t, int64(1), req.GetDelta().GetObjects())
				return &quotav1.CheckQuotaResponse{Allowed: true}, nil
			},
			updateUsageFn: func(ctx context.Context, req *quotav1.UpdateUsageRequest) (*quotav1.UpdateUsageResponse, error) {
				quotaRolledBack = true
				require.Equal(t, int64(-7), req.GetDelta().GetBytes())
				require.Equal(t, int64(-1), req.GetDelta().GetObjects())
				return &quotav1.UpdateUsageResponse{}, nil
			},
		},
		&stubStorageClient{
			storeObjectFn: func(ctx context.Context, chunks <-chan *storagev1.StoreObjectRequest) (*storagev1.StoreObjectResponse, error) {
				for range chunks {
				}
				return nil, status.Error(codes.ResourceExhausted, "disk full")
			},
		},
	)

	resp, err := svc.PutObject(context.Background(), service.PutObjectRequest{
		UserID:      "user-1",
		Bucket:      "bucket-1",
		Key:         "key-1",
		Body:        strings.NewReader("payload"),
		Size:        7,
		ContentType: "text/plain",
	})

	require.Nil(t, resp)
	require.ErrorIs(t, err, domainerrors.ErrInsufficientSpace)
	require.True(t, quotaRolledBack)
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
			getObjectMetaFn: func(ctx context.Context, req *metadatav1.GetObjectMetaRequest) (*metadatav1.GetObjectMetaResponse, error) {
				require.Equal(t, "bucket-1", req.GetBucketName())
				require.Equal(t, "key-1", req.GetKey())
				return &metadatav1.GetObjectMetaResponse{BlobId: "blob-42", SizeBytes: 64}, nil
			},
			deleteObjectMetaFn: func(ctx context.Context, req *metadatav1.DeleteObjectMetaRequest) (*metadatav1.DeleteObjectMetaResponse, error) {
				require.Equal(t, "bucket-1", req.GetBucketName())
				require.Equal(t, "key-1", req.GetKey())
				return &metadatav1.DeleteObjectMetaResponse{BlobId: "blob-42", Success: true}, nil
			},
		},
		&stubQuotaClient{
			updateUsageFn: func(ctx context.Context, req *quotav1.UpdateUsageRequest) (*quotav1.UpdateUsageResponse, error) {
				require.Equal(t, "user:user-1", req.GetSubjectId())
				require.Equal(t, "bucket:bucket-1", req.GetBucketId())
				require.Equal(t, int64(-64), req.GetDelta().GetBytes())
				require.Equal(t, int64(-1), req.GetDelta().GetObjects())
				return &quotav1.UpdateUsageResponse{}, nil
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

func TestCreateBucketRollsBackMetadataAndQuotaOnAuthzFailure(t *testing.T) {
	t.Parallel()

	var metadataRolledBack bool
	var quotaRolledBack bool
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
				metadataRolledBack = true
				require.Equal(t, "bucket-1", req.GetBucketName())
				return &metadatav1.DeleteBucketResponse{Success: true}, nil
			},
		},
		&stubQuotaClient{
			checkQuotaFn: func(ctx context.Context, req *quotav1.CheckQuotaRequest) (*quotav1.CheckQuotaResponse, error) {
				require.Equal(t, "user:user-1", req.GetSubjectId())
				require.Equal(t, int64(1), req.GetDelta().GetBuckets())
				return &quotav1.CheckQuotaResponse{Allowed: true}, nil
			},
			updateUsageFn: func(ctx context.Context, req *quotav1.UpdateUsageRequest) (*quotav1.UpdateUsageResponse, error) {
				quotaRolledBack = true
				require.Equal(t, int64(-1), req.GetDelta().GetBuckets())
				return &quotav1.UpdateUsageResponse{}, nil
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
	require.True(t, metadataRolledBack)
	require.True(t, quotaRolledBack)
}

func TestCreateBucketReturnsTooManyBucketsWhenQuotaDenies(t *testing.T) {
	t.Parallel()

	svc := NewService(
		&stubAuthZClient{},
		&stubMetadataClient{},
		&stubQuotaClient{
			checkQuotaFn: func(context.Context, *quotav1.CheckQuotaRequest) (*quotav1.CheckQuotaResponse, error) {
				return &quotav1.CheckQuotaResponse{
					Allowed: false,
					Code:    quotav1.DenyCode_DENY_CODE_USER_BUCKET_LIMIT_REACHED,
				}, nil
			},
		},
		&stubStorageClient{},
	)

	resp, err := svc.CreateBucket(context.Background(), service.CreateBucketRequest{
		UserID: "user-1",
		Bucket: "bucket-1",
	})

	require.Nil(t, resp)
	require.ErrorIs(t, err, domainerrors.ErrTooManyBuckets)
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
		&stubQuotaClient{},
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
		&stubQuotaClient{},
		&stubStorageClient{},
	)

	resp, err := svc.ListBuckets(context.Background(), service.ListBucketsRequest{UserID: "user-1"})

	require.NoError(t, err)
	require.Len(t, resp.Buckets, 1)
	require.Equal(t, "bucket-1", resp.Buckets[0].Name)
}

func TestReadyFailsWhenQuotaIsNotServing(t *testing.T) {
	t.Parallel()

	svc := NewService(
		&stubAuthZClient{
			healthCheckFn: func(context.Context, *authzv1.HealthCheckRequest) (*authzv1.HealthCheckResponse, error) {
				return &authzv1.HealthCheckResponse{Status: authzv1.HealthCheckResponse_SERVING}, nil
			},
		},
		&stubMetadataClient{
			healthCheckFn: func(context.Context, *metadatav1.HealthCheckRequest) (*metadatav1.HealthCheckResponse, error) {
				return &metadatav1.HealthCheckResponse{Status: metadatav1.HealthCheckResponse_SERVING}, nil
			},
		},
		&stubQuotaClient{
			healthCheckFn: func(context.Context, *quotav1.HealthCheckRequest) (*quotav1.HealthCheckResponse, error) {
				return &quotav1.HealthCheckResponse{Status: quotav1.HealthCheckResponse_NOT_SERVING}, nil
			},
		},
		&stubStorageClient{
			healthCheckFn: func(context.Context, *storagev1.HealthCheckRequest) (*storagev1.HealthCheckResponse, error) {
				return &storagev1.HealthCheckResponse{Status: storagev1.HealthCheckResponse_SERVING}, nil
			},
		},
	)

	err := svc.Ready(context.Background())

	require.ErrorIs(t, err, domainerrors.ErrServiceUnavailable)
}
