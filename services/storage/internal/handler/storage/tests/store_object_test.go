package tests

import (
	"context"
	"io"
	"testing"

	handlerStorage "github.com/alesplll/opens3-rebac/services/storage/internal/handler/storage"
	"github.com/alesplll/opens3-rebac/services/storage/internal/model"
	"github.com/alesplll/opens3-rebac/services/storage/internal/service"
	desc "github.com/alesplll/opens3-rebac/shared/pkg/go/storage/v1"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestStoreObject_StreamsAllChunksToService(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	reqs := []*desc.StoreObjectRequest{
		{
			Payload: &desc.StoreObjectRequest_Header{
				Header: &desc.StoreObjectHeader{
					Data:        []byte("hello "),
					Size:        int64Ptr(11),
					ContentType: "text/plain",
				},
			},
		},
		{
			Payload: &desc.StoreObjectRequest_Chunk{
				Chunk: &desc.StoreObjectChunk{Data: []byte("world")},
			},
		},
	}

	var (
		gotSize        *int64
		gotContentType string
		gotBody        []byte
	)

	svc := testStorageService{
		storeObjectFn: func(ctx context.Context, reader io.Reader, size *int64, contentType string) (*model.BlobMeta, error) {
			body, err := io.ReadAll(reader)
			require.NoError(t, err)

			gotSize = size
			gotContentType = contentType
			gotBody = body

			return &model.BlobMeta{
				BlobID:      "blob-1",
				ChecksumMD5: "md5-1",
			}, nil
		},
	}

	stream := &storeObjectServerMock{
		ctx:      ctx,
		requests: reqs,
	}

	h := handlerStorage.NewHandler(svc)
	err := h.StoreObject(stream)
	require.NoError(t, err)

	require.NotNil(t, gotSize)
	require.Equal(t, int64(11), *gotSize)
	require.Equal(t, "text/plain", gotContentType)
	require.Equal(t, []byte("hello world"), gotBody)
	require.Equal(t, &desc.StoreObjectResponse{
		BlobId:      "blob-1",
		ChecksumMd5: "md5-1",
	}, stream.closedWith)
}

func TestStoreObject_EmptyStream(t *testing.T) {
	t.Parallel()

	h := handlerStorage.NewHandler(testStorageService{})
	err := h.StoreObject(&storeObjectServerMock{
		ctx: context.Background(),
	})
	require.ErrorIs(t, err, io.ErrUnexpectedEOF)
}

func TestStoreObject_RejectsSizeOutsideFirstMessage(t *testing.T) {
	t.Parallel()

	serviceCalled := false
	h := handlerStorage.NewHandler(testStorageService{
		storeObjectFn: func(ctx context.Context, reader io.Reader, size *int64, contentType string) (*model.BlobMeta, error) {
			serviceCalled = true
			_, err := io.ReadAll(reader)
			require.Error(t, err)
			return nil, err
			return nil, err
		},
	})

	err := h.StoreObject(&storeObjectServerMock{
		ctx: context.Background(),
		requests: []*desc.StoreObjectRequest{
			{
				Payload: &desc.StoreObjectRequest_Header{
					Header: &desc.StoreObjectHeader{
						Data:        []byte("hello "),
						Size:        int64Ptr(11),
						ContentType: "text/plain",
					},
				},
			},
			{
				Payload: &desc.StoreObjectRequest_Header{
					Header: &desc.StoreObjectHeader{
						Data:        []byte("world"),
						ContentType: "application/json",
					},
				},
			},
		},
	})

	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	require.Contains(t, err.Error(), "messages after the first must be store_object chunks")
	require.True(t, serviceCalled)
}

func TestStoreObject_RejectsChunkAsFirstMessage(t *testing.T) {
	t.Parallel()

	h := handlerStorage.NewHandler(testStorageService{})
	err := h.StoreObject(&storeObjectServerMock{
		ctx: context.Background(),
		requests: []*desc.StoreObjectRequest{
			{
				Payload: &desc.StoreObjectRequest_Chunk{
					Chunk: &desc.StoreObjectChunk{Data: []byte("hello")},
				},
			},
		},
	})

	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	require.Contains(t, err.Error(), "first message must be store_object header")
}

func TestStoreObject_AllowsEmptyObjectInHeaderOnly(t *testing.T) {
	t.Parallel()

	var (
		gotSize        *int64
		gotContentType string
		gotBody        []byte
	)

	h := handlerStorage.NewHandler(testStorageService{
		storeObjectFn: func(ctx context.Context, reader io.Reader, size *int64, contentType string) (*model.BlobMeta, error) {
			body, err := io.ReadAll(reader)
			require.NoError(t, err)
			gotSize = size
			gotContentType = contentType
			gotBody = body
			return &model.BlobMeta{BlobID: "blob-empty", ChecksumMD5: "md5-empty"}, nil
		},
	})

	stream := &storeObjectServerMock{
		ctx: context.Background(),
		requests: []*desc.StoreObjectRequest{
			{
				Payload: &desc.StoreObjectRequest_Header{
					Header: &desc.StoreObjectHeader{
						Size:        int64Ptr(0),
						ContentType: "application/octet-stream",
					},
				},
			},
		},
	}

	err := h.StoreObject(stream)
	require.NoError(t, err)
	require.NotNil(t, gotSize)
	require.Equal(t, int64(0), *gotSize)
	require.Equal(t, "application/octet-stream", gotContentType)
	require.Empty(t, gotBody)
	require.Equal(t, "blob-empty", stream.closedWith.GetBlobId())
}

func int64Ptr(v int64) *int64 {
	return &v
}

type testStorageService struct {
	storeObjectFn func(ctx context.Context, reader io.Reader, size *int64, contentType string) (*model.BlobMeta, error)
	uploadPartFn  func(ctx context.Context, uploadID string, partNumber int32, reader io.Reader) (string, error)
}

var _ service.StorageService = testStorageService{}

func (s testStorageService) StoreObject(ctx context.Context, reader io.Reader, size *int64, contentType string) (*model.BlobMeta, error) {
	if s.storeObjectFn == nil {
		return nil, nil
	}
	return s.storeObjectFn(ctx, reader, size, contentType)
}

func (s testStorageService) RetrieveObject(ctx context.Context, blobID string, offset, length int64) (io.ReadCloser, int64, error) {
	panic("unexpected call")
}

func (s testStorageService) DeleteObject(ctx context.Context, blobID string) error {
	panic("unexpected call")
}

func (s testStorageService) InitiateMultipartUpload(ctx context.Context, expectedParts int32, contentType string) (string, error) {
	panic("unexpected call")
}

func (s testStorageService) UploadPart(ctx context.Context, uploadID string, partNumber int32, reader io.Reader) (string, error) {
	if s.uploadPartFn == nil {
		panic("unexpected call")
	}
	return s.uploadPartFn(ctx, uploadID, partNumber, reader)
}

func (s testStorageService) CompleteMultipartUpload(ctx context.Context, uploadID string, parts []model.PartInfo) (*model.BlobMeta, error) {
	panic("unexpected call")
}

func (s testStorageService) AbortMultipartUpload(ctx context.Context, uploadID string) error {
	panic("unexpected call")
}

func (s testStorageService) HealthCheck(ctx context.Context, serviceName string) (bool, error) {
	panic("unexpected call")
}

type storeObjectServerMock struct {
	grpc.ServerStream
	ctx        context.Context
	requests   []*desc.StoreObjectRequest
	recvIndex  int
	closedWith *desc.StoreObjectResponse
}

func (s *storeObjectServerMock) Context() context.Context {
	return s.ctx
}

func (s *storeObjectServerMock) SendAndClose(resp *desc.StoreObjectResponse) error {
	s.closedWith = resp
	return nil
}

func (s *storeObjectServerMock) Recv() (*desc.StoreObjectRequest, error) {
	if s.recvIndex >= len(s.requests) {
		return nil, io.EOF
	}

	req := s.requests[s.recvIndex]
	s.recvIndex++
	return req, nil
}

func (s *storeObjectServerMock) SetHeader(metadata.MD) error {
	return nil
}

func (s *storeObjectServerMock) SendHeader(metadata.MD) error {
	return nil
}

func (s *storeObjectServerMock) SetTrailer(metadata.MD) {}

func (s *storeObjectServerMock) SendMsg(any) error {
	return nil
}

func (s *storeObjectServerMock) RecvMsg(any) error {
	return nil
}
