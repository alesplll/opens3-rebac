package gateway

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	grpcclient "github.com/alesplll/opens3-rebac/services/gateway/internal/client/grpc"
	domainerrors "github.com/alesplll/opens3-rebac/services/gateway/internal/errors/domain_errors"
	"github.com/alesplll/opens3-rebac/services/gateway/internal/service"
	authzv1 "github.com/alesplll/opens3-rebac/shared/pkg/go/authz/v1"
	metadatav1 "github.com/alesplll/opens3-rebac/shared/pkg/go/metadata/v1"
	storagev1 "github.com/alesplll/opens3-rebac/shared/pkg/go/storage/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	defaultListMaxKeys = 1000
	chunkSize          = 64 * 1024
)

type gatewayService struct {
	authzClient    grpcclient.AuthZClient
	metadataClient grpcclient.MetadataClient
	storageClient  grpcclient.StorageClient
}

func NewService(
	authzClient grpcclient.AuthZClient,
	metadataClient grpcclient.MetadataClient,
	storageClient grpcclient.StorageClient,
) service.GatewayService {
	return &gatewayService{
		authzClient:    authzClient,
		metadataClient: metadataClient,
		storageClient:  storageClient,
	}
}

func (s *gatewayService) CreateBucket(ctx context.Context, req service.CreateBucketRequest) (*service.CreateBucketResponse, error) {
	resp, err := s.metadataClient.CreateBucket(ctx, &metadatav1.CreateBucketRequest{
		Name:    req.Bucket,
		OwnerId: req.UserID,
	})
	if err != nil {
		return nil, mapBucketGRPCError(err)
	}

	_, err = s.authzClient.WriteTuple(ctx, &authzv1.WriteTupleRequest{
		Subject:  subjectUser(req.UserID),
		Relation: authzv1.Relation_RELATION_HAS_PERMISSION,
		Object:   bucketResource(req.Bucket),
		Level:    authzv1.PermissionLevel_PERMISSION_LEVEL_ADMIN,
	})
	if err != nil {
		authzErr := mapGRPCError(err)
		_, rollbackErr := s.metadataClient.DeleteBucket(ctx, &metadatav1.DeleteBucketRequest{BucketName: req.Bucket})
		if rollbackErr != nil {
			return nil, fmt.Errorf("create bucket authz tuple failed: %w; rollback delete bucket failed: %v", authzErr, mapBucketGRPCError(rollbackErr))
		}
		return nil, authzErr
	}

	return &service.CreateBucketResponse{
		BucketID:  resp.GetBucketId(),
		CreatedAt: millisToTime(resp.GetCreatedAt()),
	}, nil
}

func (s *gatewayService) DeleteBucket(ctx context.Context, req service.DeleteBucketRequest) error {
	if err := s.checkAccess(ctx, req.UserID, authzv1.Action_ACTION_ADMIN, bucketResource(req.Bucket)); err != nil {
		return err
	}

	_, err := s.metadataClient.DeleteBucket(ctx, &metadatav1.DeleteBucketRequest{BucketName: req.Bucket})
	if err != nil {
		return mapGRPCError(err)
	}

	return nil
}

func (s *gatewayService) ListBuckets(ctx context.Context, req service.ListBucketsRequest) (*service.ListBucketsResponse, error) {
	resp, err := s.metadataClient.ListBuckets(ctx, &metadatav1.ListBucketsRequest{OwnerId: req.UserID})
	if err != nil {
		return nil, mapGRPCError(err)
	}

	result := &service.ListBucketsResponse{Buckets: make([]service.BucketInfo, 0, len(resp.GetBuckets()))}
	for _, bucket := range resp.GetBuckets() {
		result.Buckets = append(result.Buckets, service.BucketInfo{
			Name:      bucket.GetName(),
			CreatedAt: millisToTime(bucket.GetCreatedAt()),
		})
	}

	return result, nil
}

func (s *gatewayService) HeadBucket(ctx context.Context, req service.HeadBucketRequest) error {
	if err := s.checkAccess(ctx, req.UserID, authzv1.Action_ACTION_READ, bucketResource(req.Bucket)); err != nil {
		return err
	}

	_, err := s.metadataClient.HeadBucket(ctx, &metadatav1.HeadBucketRequest{BucketName: req.Bucket})
	if err != nil {
		return mapGRPCError(err)
	}

	return nil
}

func (s *gatewayService) PutObject(ctx context.Context, req service.PutObjectRequest) (*service.PutObjectResponse, error) {
	if err := s.checkAccess(ctx, req.UserID, authzv1.Action_ACTION_WRITE, objectResource(req.Bucket, req.Key)); err != nil {
		return nil, err
	}

	storeResp, err := s.storeObject(ctx, req.Body, req.Size, req.ContentType)
	if err != nil {
		return nil, err
	}

	metaResp, err := s.metadataClient.CreateObjectVersion(ctx, &metadatav1.CreateObjectVersionRequest{
		BucketName:  req.Bucket,
		Key:         req.Key,
		BlobId:      storeResp.GetBlobId(),
		SizeBytes:   req.Size,
		Etag:        storeResp.GetChecksumMd5(),
		ContentType: req.ContentType,
	})
	if err != nil {
		return nil, mapObjectGRPCError(err)
	}

	if err := s.writeTupleWithRetry(ctx, &authzv1.WriteTupleRequest{
		Subject:  bucketResource(req.Bucket),
		Relation: authzv1.Relation_RELATION_PARENT_OF,
		Object:   objectResource(req.Bucket, req.Key),
	}); err != nil {
		return nil, mapGRPCError(err)
	}

	return &service.PutObjectResponse{
		ETag:      quoteETag(storeResp.GetChecksumMd5()),
		VersionID: metaResp.GetVersionId(),
	}, nil
}

func (s *gatewayService) GetObject(ctx context.Context, req service.GetObjectRequest) (*service.GetObjectResponse, error) {
	if err := s.checkAccess(ctx, req.UserID, authzv1.Action_ACTION_READ, objectResource(req.Bucket, req.Key)); err != nil {
		return nil, err
	}

	meta, err := s.metadataClient.GetObjectMeta(ctx, &metadatav1.GetObjectMetaRequest{
		BucketName: req.Bucket,
		Key:        req.Key,
		VersionId:  req.VersionID,
	})
	if err != nil {
		return nil, mapGRPCError(err)
	}

	retrieveReq := &storagev1.RetrieveObjectRequest{BlobId: meta.GetBlobId()}
	contentLength := meta.GetSizeBytes()
	var responseRange *service.ByteRange
	if req.Range != nil {
		start, end, err := normalizeRange(meta.GetSizeBytes(), req.Range)
		if err != nil {
			return nil, err
		}
		retrieveReq.Offset = start
		retrieveReq.Length = end - start + 1
		contentLength = retrieveReq.Length
		responseRange = &service.ByteRange{Start: start, End: end}
	}

	streamResp, err := s.storageClient.RetrieveObject(ctx, retrieveReq, req.Writer)
	if err != nil {
		return nil, mapGRPCError(err)
	}

	return &service.GetObjectResponse{
		ContentType:   meta.GetContentType(),
		ContentLength: contentLength,
		ETag:          quoteETag(meta.GetEtag()),
		VersionID:     meta.GetVersionId(),
		LastModified:  millisToTime(meta.GetLastModified()),
		Range:         responseRange,
		TotalSize:     streamResp.GetTotalSize(),
	}, nil
}

func (s *gatewayService) HeadObject(ctx context.Context, req service.HeadObjectRequest) (*service.HeadObjectResponse, error) {
	if err := s.checkAccess(ctx, req.UserID, authzv1.Action_ACTION_READ, objectResource(req.Bucket, req.Key)); err != nil {
		return nil, err
	}

	meta, err := s.metadataClient.GetObjectMeta(ctx, &metadatav1.GetObjectMetaRequest{
		BucketName: req.Bucket,
		Key:        req.Key,
		VersionId:  req.VersionID,
	})
	if err != nil {
		return nil, mapGRPCError(err)
	}

	return &service.HeadObjectResponse{
		ContentType:   meta.GetContentType(),
		ContentLength: meta.GetSizeBytes(),
		ETag:          quoteETag(meta.GetEtag()),
		VersionID:     meta.GetVersionId(),
		LastModified:  millisToTime(meta.GetLastModified()),
	}, nil
}

func (s *gatewayService) DeleteObject(ctx context.Context, req service.DeleteObjectRequest) error {
	if err := s.checkAccess(ctx, req.UserID, authzv1.Action_ACTION_DELETE, objectResource(req.Bucket, req.Key)); err != nil {
		return err
	}

	resp, err := s.metadataClient.DeleteObjectMeta(ctx, &metadatav1.DeleteObjectMetaRequest{
		BucketName: req.Bucket,
		Key:        req.Key,
	})
	if err != nil {
		return mapGRPCError(err)
	}
	if strings.TrimSpace(resp.GetBlobId()) == "" {
		return nil
	}

	if _, err := s.storageClient.DeleteObject(ctx, &storagev1.DeleteObjectRequest{BlobId: resp.GetBlobId()}); err != nil {
		return mapGRPCError(err)
	}

	return nil
}

func (s *gatewayService) ListObjects(ctx context.Context, req service.ListObjectsRequest) (*service.ListObjectsResponse, error) {
	if err := s.checkAccess(ctx, req.UserID, authzv1.Action_ACTION_READ, bucketResource(req.Bucket)); err != nil {
		return nil, err
	}

	maxKeys := req.MaxKeys
	if maxKeys <= 0 || maxKeys > defaultListMaxKeys {
		maxKeys = defaultListMaxKeys
	}

	resp, err := s.metadataClient.ListObjects(ctx, &metadatav1.ListObjectsRequest{
		BucketName:        req.Bucket,
		Prefix:            req.Prefix,
		ContinuationToken: req.ContinuationToken,
		MaxKeys:           maxKeys,
	})
	if err != nil {
		return nil, mapGRPCError(err)
	}

	result := &service.ListObjectsResponse{
		Objects:               make([]service.ObjectInfo, 0, len(resp.GetObjects())),
		NextContinuationToken: resp.GetNextContinuationToken(),
		IsTruncated:           resp.GetIsTruncated(),
	}
	for _, object := range resp.GetObjects() {
		result.Objects = append(result.Objects, service.ObjectInfo{
			Key:          object.GetKey(),
			ETag:         object.GetEtag(),
			Size:         object.GetSizeBytes(),
			ContentType:  object.GetContentType(),
			LastModified: millisToTime(object.GetLastModified()),
			VersionID:    object.GetVersionId(),
		})
	}

	return result, nil
}

func (s *gatewayService) CreateMultipartUpload(ctx context.Context, req service.CreateMultipartUploadRequest) (*service.CreateMultipartUploadResponse, error) {
	if err := s.checkAccess(ctx, req.UserID, authzv1.Action_ACTION_CREATE, objectResource(req.Bucket, req.Key)); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.ContentType) == "" {
		req.ContentType = "application/octet-stream"
	}

	resp, err := s.storageClient.InitiateMultipartUpload(ctx, &storagev1.InitiateMultipartUploadRequest{
		ContentType: req.ContentType,
	})
	if err != nil {
		return nil, mapGRPCError(err)
	}

	return &service.CreateMultipartUploadResponse{UploadID: resp.GetUploadId()}, nil
}

func (s *gatewayService) UploadPart(ctx context.Context, req service.UploadPartRequest) (*service.UploadPartResponse, error) {
	if err := s.checkAccess(ctx, req.UserID, authzv1.Action_ACTION_WRITE, objectResource(req.Bucket, req.Key)); err != nil {
		return nil, err
	}

	chunks := make(chan *storagev1.UploadPartRequest)
	errCh := make(chan error, 1)
	done := make(chan struct{})
	var once sync.Once
	closeDone := func() {
		once.Do(func() { close(done) })
	}

	go func() {
		defer close(chunks)
		defer close(errCh)

		buf := make([]byte, chunkSize)
		for {
			n, err := req.Body.Read(buf)
			if n > 0 {
				data := make([]byte, n)
				copy(data, buf[:n])
				chunk := &storagev1.UploadPartRequest{
					Payload: &storagev1.UploadPartRequest_Header{
						Header: &storagev1.UploadPartHeader{
							UploadId:   req.UploadID,
							PartNumber: req.PartNumber,
							Data:       data,
						},
					},
				}
				select {
				case chunks <- chunk:
					req.UploadID = ""
				case <-done:
					return
				case <-ctx.Done():
					select {
					case errCh <- ctx.Err():
					default:
					}
					return
				}
			}
			if err == io.EOF {
				select {
				case errCh <- nil:
				default:
				}
				return
			}
			if err != nil {
				select {
				case errCh <- err:
				default:
				}
				return
			}
		}
	}()

	resp, err := s.storageClient.UploadPart(ctx, chunks)
	closeDone()
	if err != nil {
		return nil, mapGRPCError(err)
	}
	if err := <-errCh; err != nil {
		return nil, err
	}

	return &service.UploadPartResponse{ETag: quoteETag(resp.GetPartChecksumMd5())}, nil
}

func (s *gatewayService) CompleteMultipartUpload(ctx context.Context, req service.CompleteMultipartUploadRequest) (*service.CompleteMultipartUploadResponse, error) {
	if err := s.checkAccess(ctx, req.UserID, authzv1.Action_ACTION_WRITE, objectResource(req.Bucket, req.Key)); err != nil {
		return nil, err
	}

	parts := make([]*storagev1.PartInfo, 0, len(req.Parts))
	for _, part := range req.Parts {
		parts = append(parts, &storagev1.PartInfo{
			PartNumber:  part.PartNumber,
			ChecksumMd5: strings.Trim(part.ETag, `"`),
		})
	}

	completeResp, err := s.storageClient.CompleteMultipartUpload(ctx, &storagev1.CompleteMultipartUploadRequest{
		UploadId: req.UploadID,
		Parts:    parts,
	})
	if err != nil {
		return nil, mapGRPCError(err)
	}

	metaResp, err := s.metadataClient.CreateObjectVersion(ctx, &metadatav1.CreateObjectVersionRequest{
		BucketName: req.Bucket,
		Key:        req.Key,
		BlobId:     completeResp.GetBlobId(),
		Etag:       completeResp.GetChecksumMd5(),
	})
	if err != nil {
		return nil, mapObjectGRPCError(err)
	}

	return &service.CompleteMultipartUploadResponse{
		ETag:      quoteETag(completeResp.GetChecksumMd5()),
		VersionID: metaResp.GetVersionId(),
	}, nil
}

func (s *gatewayService) AbortMultipartUpload(ctx context.Context, req service.AbortMultipartUploadRequest) error {
	if err := s.checkAccess(ctx, req.UserID, authzv1.Action_ACTION_WRITE, objectResource(req.Bucket, req.Key)); err != nil {
		return err
	}

	_, err := s.storageClient.AbortMultipartUpload(ctx, &storagev1.AbortMultipartUploadRequest{UploadId: req.UploadID})
	if err != nil {
		return mapGRPCError(err)
	}

	return nil
}

func (s *gatewayService) Ready(ctx context.Context) error {
	authzResp, err := s.authzClient.HealthCheck(ctx, &authzv1.HealthCheckRequest{})
	if err != nil {
		return mapGRPCError(err)
	}
	if authzResp.GetStatus() != authzv1.HealthCheckResponse_SERVING {
		return domainerrors.ErrServiceUnavailable
	}

	metadataResp, err := s.metadataClient.HealthCheck(ctx, &metadatav1.HealthCheckRequest{})
	if err != nil {
		return mapGRPCError(err)
	}
	if metadataResp.GetStatus() != metadatav1.HealthCheckResponse_SERVING {
		return domainerrors.ErrServiceUnavailable
	}

	storageResp, err := s.storageClient.HealthCheck(ctx, &storagev1.HealthCheckRequest{})
	if err != nil {
		return mapGRPCError(err)
	}
	if storageResp.GetStatus() != storagev1.HealthCheckResponse_SERVING {
		return domainerrors.ErrServiceUnavailable
	}

	return nil
}

func (s *gatewayService) checkAccess(ctx context.Context, userID string, action authzv1.Action, object string) error {
	resp, err := s.authzClient.Check(ctx, &authzv1.CheckRequest{
		Subject: subjectUser(userID),
		Action:  action,
		Object:  object,
	})
	if err != nil {
		return mapGRPCError(err)
	}
	if !resp.GetAllowed() {
		return domainerrors.ErrForbidden
	}

	return nil
}

func (s *gatewayService) storeObject(ctx context.Context, body io.Reader, size int64, contentType string) (*storagev1.StoreObjectResponse, error) {
	chunks := make(chan *storagev1.StoreObjectRequest)
	errCh := make(chan error, 1)
	done := make(chan struct{})
	var once sync.Once
	closeDone := func() {
		once.Do(func() { close(done) })
	}

	go func() {
		defer close(chunks)
		defer close(errCh)

		buf := make([]byte, chunkSize)
		first := true
		for {
			n, err := body.Read(buf)
			if n > 0 {
				data := make([]byte, n)
				copy(data, buf[:n])
				var chunk *storagev1.StoreObjectRequest
				if first {
					chunk = &storagev1.StoreObjectRequest{
						Payload: &storagev1.StoreObjectRequest_Header{
							Header: &storagev1.StoreObjectHeader{
								Size:        &size,
								ContentType: contentType,
								Data:        data,
							},
						},
					}
					first = false
				} else {
					chunk = &storagev1.StoreObjectRequest{
						Payload: &storagev1.StoreObjectRequest_Chunk{
							Chunk: &storagev1.StoreObjectChunk{Data: data},
						},
					}
				}

				select {
				case chunks <- chunk:
				case <-done:
					return
				case <-ctx.Done():
					select {
					case errCh <- ctx.Err():
					default:
					}
					return
				}
			}
			if err == io.EOF {
				if first {
					emptyChunk := &storagev1.StoreObjectRequest{
						Payload: &storagev1.StoreObjectRequest_Header{
							Header: &storagev1.StoreObjectHeader{
								Size:        &size,
								ContentType: contentType,
							},
						},
					}
					select {
					case chunks <- emptyChunk:
					case <-done:
						return
					case <-ctx.Done():
						select {
						case errCh <- ctx.Err():
						default:
						}
						return
					}
				}
				select {
				case errCh <- nil:
				default:
				}
				return
			}
			if err != nil {
				select {
				case errCh <- err:
				default:
				}
				return
			}
		}
	}()

	resp, err := s.storageClient.StoreObject(ctx, chunks)
	closeDone()
	if err != nil {
		return nil, mapGRPCError(err)
	}
	if err := <-errCh; err != nil {
		return nil, err
	}

	return resp, nil
}

func mapBucketGRPCError(err error) error {
	return mapGRPCError(err, domainerrors.ErrBucketNotFound)
}

func mapObjectGRPCError(err error) error {
	return mapGRPCError(err, domainerrors.ErrObjectNotFound)
}

func mapGRPCError(err error, notFoundErr ...error) error {
	if err == nil {
		return nil
	}

	st, ok := status.FromError(err)
	if !ok {
		return err
	}

	var nf error
	if len(notFoundErr) > 0 {
		nf = notFoundErr[0]
	}

	switch st.Code() {
	case codes.NotFound:
		if nf != nil {
			return nf
		}
		return domainerrors.ErrObjectNotFound
	case codes.AlreadyExists:
		return domainerrors.ErrBucketAlreadyExist
	case codes.FailedPrecondition:
		return domainerrors.ErrBucketNotEmpty
	case codes.PermissionDenied:
		return domainerrors.ErrForbidden
	case codes.InvalidArgument:
		return fmt.Errorf("%w: %s", domainerrors.ErrInvalidRequest, st.Message())
	case codes.ResourceExhausted:
		return domainerrors.ErrInsufficientSpace
	case codes.Unavailable, codes.Unimplemented:
		return domainerrors.ErrServiceUnavailable
	default:
		return err
	}
}

func (s *gatewayService) writeTupleWithRetry(ctx context.Context, req *authzv1.WriteTupleRequest) error {
	const maxAttempts = 3
	backoff := 100 * time.Millisecond

	var err error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		_, err = s.authzClient.WriteTuple(ctx, req)
		if err == nil {
			return nil
		}
		if attempt == maxAttempts {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}
		backoff *= 2
	}

	return err
}

func normalizeRange(total int64, requested *service.ByteRange) (int64, int64, error) {
	if requested == nil {
		return 0, total - 1, nil
	}
	if total <= 0 {
		return 0, 0, domainerrors.ErrInvalidRange
	}
	start := requested.Start
	end := requested.End
	if start < 0 || end < start || start >= total {
		return 0, 0, domainerrors.ErrInvalidRange
	}
	if end >= total {
		end = total - 1
	}
	return start, end, nil
}

func quoteETag(etag string) string {
	trimmed := strings.Trim(etag, `"`)
	return fmt.Sprintf("\"%s\"", trimmed)
}

func millisToTime(v int64) time.Time {
	if v == 0 {
		return time.Time{}
	}
	return time.UnixMilli(v).UTC()
}

func subjectUser(userID string) string {
	return "user:" + userID
}

func bucketResource(bucket string) string {
	return "bucket:" + bucket
}

func objectResource(bucket, key string) string {
	return "object:" + bucket + "/" + key
}
