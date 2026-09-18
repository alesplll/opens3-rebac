package object

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/alesplll/opens3-rebac/services/gateway/internal/service"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/logger"
	metadatav1 "github.com/alesplll/opens3-rebac/shared/pkg/go/metadata/v1"
	storagev1 "github.com/alesplll/opens3-rebac/shared/pkg/go/storage/v1"
	"go.uber.org/zap"
)

const uploadChunkSize = 64 * 1024

func (s *objectService) Put(ctx context.Context, bucket, key string, body io.Reader, size *int64, contentType string) (service.PutResult, error) {
	exists, err := s.metadata.HeadBucket(ctx, &metadatav1.HeadBucketRequest{BucketName: bucket})
	if err != nil {
		return service.PutResult{}, err
	}
	if !exists.GetExists() {
		return service.PutResult{}, service.ErrBucketNotFound
	}

	streamCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	stream, err := s.storage.StoreObject(streamCtx)
	if err != nil {
		return service.PutResult{}, err
	}
	header := &storagev1.StoreObjectHeader{ContentType: contentType, Size: size}
	if err := stream.Send(&storagev1.StoreObjectRequest{Payload: &storagev1.StoreObjectRequest_Header{Header: header}}); err != nil {
		return service.PutResult{}, storeSendError(stream, err)
	}

	buf := make([]byte, uploadChunkSize)
	var total int64
	for {
		n, readErr := body.Read(buf)
		if n > 0 {
			if err := stream.Send(&storagev1.StoreObjectRequest{Payload: &storagev1.StoreObjectRequest_Chunk{Chunk: &storagev1.StoreObjectChunk{Data: buf[:n]}}}); err != nil {
				return service.PutResult{}, storeSendError(stream, err)
			}
			total += int64(n)
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return service.PutResult{}, fmt.Errorf("%w: %v", service.ErrBodyRead, readErr)
		}
	}

	stored, err := stream.CloseAndRecv()
	if err != nil {
		return service.PutResult{}, err
	}
	version, err := s.metadata.CreateObjectVersion(ctx, &metadatav1.CreateObjectVersionRequest{
		BucketName: bucket, Key: key, BlobId: stored.GetBlobId(),
		SizeBytes: total, Etag: stored.GetChecksumMd5(), ContentType: contentType,
	})
	if err != nil {
		logger.Error(ctx, "metadata registration failed after blob commit",
			zap.String("blob_id", stored.GetBlobId()), zap.String("bucket", bucket), zap.String("key", key), zap.Error(err))
		return service.PutResult{}, err
	}
	return service.PutResult{ETag: stored.GetChecksumMd5(), VersionID: version.GetVersionId()}, nil
}

// Send returns EOF when the server has rejected the stream. Receive its status
// so an early rejection has the same HTTP mapping as a failure at CloseAndRecv.
func storeSendError(stream storagev1.DataStorageService_StoreObjectClient, sendErr error) error {
	if errors.Is(sendErr, io.EOF) {
		if _, err := stream.CloseAndRecv(); err != nil {
			return err
		}
	}
	return sendErr
}
