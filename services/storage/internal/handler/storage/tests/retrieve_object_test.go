package tests

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	handlerStorage "github.com/alesplll/opens3-rebac/services/storage/internal/handler/storage"
	"github.com/alesplll/opens3-rebac/services/storage/internal/model"
	"github.com/alesplll/opens3-rebac/services/storage/internal/service"
	desc "github.com/alesplll/opens3-rebac/shared/pkg/go/storage/v1"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestRetrieveObject_ReturnsReadError(t *testing.T) {
	t.Parallel()

	readErr := errors.New("read failed")

	svc := testRetrieveStorageService{
		retrieveObjectFn: func(ctx context.Context, blobID string, offset, length int64) (io.ReadCloser, int64, error) {
			return &failingReadCloser{
				chunks: [][]byte{[]byte("hello")},
				err:    readErr,
			}, 5, nil
		},
	}

	stream := &retrieveObjectServerMock{
		ctx: context.Background(),
	}

	h := handlerStorage.NewHandler(svc, 1<<20)
	err := h.RetrieveObject(&desc.RetrieveObjectRequest{BlobId: "blob-1"}, stream)
	require.ErrorIs(t, err, readErr)
	require.Len(t, stream.sent, 1)
	require.Equal(t, []byte("hello"), stream.sent[0].GetData())
	require.Equal(t, int64(5), stream.sent[0].GetTotalSize())
}

func TestRetrieveObject_UsesConfiguredChunkSize(t *testing.T) {
	t.Parallel()

	const configuredChunkSize = 4
	svc := testRetrieveStorageService{
		retrieveObjectFn: func(context.Context, string, int64, int64) (io.ReadCloser, int64, error) {
			return io.NopCloser(strings.NewReader("0123456789")), 10, nil
		},
	}
	stream := &retrieveObjectServerMock{ctx: context.Background()}

	h := handlerStorage.NewHandler(svc, configuredChunkSize)
	err := h.RetrieveObject(&desc.RetrieveObjectRequest{BlobId: "blob-1"}, stream)
	require.NoError(t, err)
	require.Len(t, stream.sent, 3)
	require.Equal(t, []byte("0123"), stream.sent[0].GetData())
	require.Equal(t, int64(10), stream.sent[0].GetTotalSize())
	require.Equal(t, []byte("4567"), stream.sent[1].GetData())
	require.Zero(t, stream.sent[1].GetTotalSize())
	require.Equal(t, []byte("89"), stream.sent[2].GetData())
}

type testRetrieveStorageService struct {
	retrieveObjectFn func(ctx context.Context, blobID string, offset, length int64) (io.ReadCloser, int64, error)
}

var _ service.StorageService = testRetrieveStorageService{}

func (s testRetrieveStorageService) StoreObject(ctx context.Context, reader io.Reader, size *int64, contentType string) (*model.BlobMeta, error) {
	panic("unexpected call")
}

func (s testRetrieveStorageService) RetrieveObject(ctx context.Context, blobID string, offset, length int64) (io.ReadCloser, int64, error) {
	if s.retrieveObjectFn == nil {
		panic("unexpected call")
	}
	return s.retrieveObjectFn(ctx, blobID, offset, length)
}

func (s testRetrieveStorageService) DeleteObject(ctx context.Context, blobID string) error {
	panic("unexpected call")
}

func (s testRetrieveStorageService) InitiateMultipartUpload(ctx context.Context, expectedParts int32, contentType string) (string, error) {
	panic("unexpected call")
}

func (s testRetrieveStorageService) UploadPart(ctx context.Context, uploadID string, partNumber int32, reader io.Reader) (string, error) {
	panic("unexpected call")
}

func (s testRetrieveStorageService) CompleteMultipartUpload(ctx context.Context, uploadID string, parts []model.PartInfo) (*model.BlobMeta, error) {
	panic("unexpected call")
}

func (s testRetrieveStorageService) AbortMultipartUpload(ctx context.Context, uploadID string) error {
	panic("unexpected call")
}

func (s testRetrieveStorageService) HealthCheck(ctx context.Context, serviceName string) (bool, error) {
	panic("unexpected call")
}

type retrieveObjectServerMock struct {
	grpc.ServerStream
	ctx  context.Context
	sent []*desc.RetrieveObjectResponse
}

func (s *retrieveObjectServerMock) Context() context.Context {
	return s.ctx
}

func (s *retrieveObjectServerMock) Send(resp *desc.RetrieveObjectResponse) error {
	s.sent = append(s.sent, &desc.RetrieveObjectResponse{
		Data:      bytes.Clone(resp.GetData()),
		TotalSize: resp.GetTotalSize(),
	})
	return nil
}

func (s *retrieveObjectServerMock) SetHeader(metadata.MD) error {
	return nil
}

func (s *retrieveObjectServerMock) SendHeader(metadata.MD) error {
	return nil
}

func (s *retrieveObjectServerMock) SetTrailer(metadata.MD) {}

func (s *retrieveObjectServerMock) SendMsg(any) error {
	return nil
}

func (s *retrieveObjectServerMock) RecvMsg(any) error {
	return nil
}

type failingReadCloser struct {
	chunks [][]byte
	err    error
	index  int
}

func (r *failingReadCloser) Read(p []byte) (int, error) {
	if r.index >= len(r.chunks) {
		return 0, r.err
	}

	n := copy(p, r.chunks[r.index])
	r.index++
	return n, nil
}

func (r *failingReadCloser) Close() error {
	return nil
}
