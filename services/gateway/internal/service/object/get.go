package object

import (
	"bytes"
	"context"
	"errors"
	"io"

	"github.com/alesplll/opens3-rebac/services/gateway/internal/service"
	metadatav1 "github.com/alesplll/opens3-rebac/shared/pkg/go/metadata/v1"
	storagev1 "github.com/alesplll/opens3-rebac/shared/pkg/go/storage/v1"
)

func (s *objectService) Get(ctx context.Context, bucket, key string) (service.GetResult, error) {
	meta, err := s.metadata.GetObjectMeta(ctx, &metadatav1.GetObjectMetaRequest{BucketName: bucket, Key: key})
	if err != nil {
		return service.GetResult{}, err
	}
	stream, err := s.storage.RetrieveObject(ctx, &storagev1.RetrieveObjectRequest{BlobId: meta.GetBlobId()})
	if err != nil {
		return service.GetResult{}, err
	}
	first, err := stream.Recv()
	if err != nil && !errors.Is(err, io.EOF) {
		return service.GetResult{}, err
	}
	if errors.Is(err, io.EOF) && meta.GetSizeBytes() != 0 {
		return service.GetResult{}, service.ErrEmptyStorageStream
	}
	var body io.Reader = bytes.NewReader(nil)
	if first != nil {
		body = io.MultiReader(bytes.NewReader(first.GetData()), &streamReader{stream: stream})
	}
	return service.GetResult{
		Body: body, Size: meta.GetSizeBytes(), ETag: meta.GetEtag(),
		VersionID: meta.GetVersionId(), ContentType: meta.GetContentType(),
	}, nil
}

type streamReader struct {
	stream storagev1.DataStorageService_RetrieveObjectClient
	buffer *bytes.Reader
}

func (r *streamReader) Read(p []byte) (int, error) {
	for r.buffer == nil || r.buffer.Len() == 0 {
		chunk, err := r.stream.Recv()
		if err != nil {
			return 0, err
		}
		r.buffer = bytes.NewReader(chunk.GetData())
	}
	return r.buffer.Read(p)
}
