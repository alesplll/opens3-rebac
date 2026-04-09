package tests

import (
	"context"
	"io"
	"testing"
	"time"

	handlerStorage "github.com/alesplll/opens3-rebac/services/storage/internal/handler/storage"
	desc "github.com/alesplll/opens3-rebac/shared/pkg/go/storage/v1"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestUploadPart_StreamsAllChunksToService(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	reqs := []*desc.UploadPartRequest{
		{
			Payload: &desc.UploadPartRequest_Header{
				Header: &desc.UploadPartHeader{
					UploadId:   "upload-1",
					PartNumber: 3,
					Data:       []byte("hello "),
				},
			},
		},
		{
			Payload: &desc.UploadPartRequest_Chunk{
				Chunk: &desc.UploadPartChunk{Data: []byte("world")},
			},
		},
	}

	var (
		gotUploadID   string
		gotPartNumber int32
		gotBody       []byte
	)

	svc := testStorageService{
		uploadPartFn: func(ctx context.Context, uploadID string, partNumber int32, reader io.Reader) (string, error) {
			body, err := io.ReadAll(reader)
			require.NoError(t, err)

			gotUploadID = uploadID
			gotPartNumber = partNumber
			gotBody = body

			return "md5-1", nil
		},
	}

	stream := &uploadPartServerMock{
		ctx:      ctx,
		requests: reqs,
	}

	h := handlerStorage.NewHandler(svc)
	err := h.UploadPart(stream)
	require.NoError(t, err)

	require.Equal(t, "upload-1", gotUploadID)
	require.Equal(t, int32(3), gotPartNumber)
	require.Equal(t, []byte("hello world"), gotBody)
	require.Equal(t, &desc.UploadPartResponse{
		PartChecksumMd5: "md5-1",
	}, stream.closedWith)
}

func TestUploadPart_EmptyStream(t *testing.T) {
	t.Parallel()

	h := handlerStorage.NewHandler(testStorageService{})
	err := h.UploadPart(&uploadPartServerMock{
		ctx: context.Background(),
	})
	require.ErrorIs(t, err, io.ErrUnexpectedEOF)
}

func TestUploadPart_StartsStreamingBeforeClientStreamEnds(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	secondChunkReleased := make(chan struct{})
	serviceStartedReading := make(chan struct{})
	done := make(chan error, 1)

	svc := testStorageService{
		uploadPartFn: func(ctx context.Context, uploadID string, partNumber int32, reader io.Reader) (string, error) {
			buf := make([]byte, len("hello "))
			n, err := io.ReadFull(reader, buf)
			require.NoError(t, err)
			require.Equal(t, len("hello "), n)
			require.Equal(t, []byte("hello "), buf)
			require.Equal(t, "upload-1", uploadID)
			require.Equal(t, int32(3), partNumber)

			close(serviceStartedReading)

			rest, err := io.ReadAll(reader)
			require.NoError(t, err)
			require.Equal(t, []byte("world"), rest)

			return "md5-1", nil
		},
	}

	stream := &uploadPartServerMock{
		ctx: ctx,
		requests: []*desc.UploadPartRequest{
			{
				Payload: &desc.UploadPartRequest_Header{
					Header: &desc.UploadPartHeader{
						UploadId:   "upload-1",
						PartNumber: 3,
						Data:       []byte("hello "),
					},
				},
			},
			{
				Payload: &desc.UploadPartRequest_Chunk{
					Chunk: &desc.UploadPartChunk{Data: []byte("world")},
				},
			},
		},
		beforeRecv: func(recvIndex int) {
			if recvIndex == 1 {
				<-secondChunkReleased
			}
		},
	}

	h := handlerStorage.NewHandler(svc)
	go func() {
		done <- h.UploadPart(stream)
	}()

	select {
	case <-serviceStartedReading:
	case <-time.After(time.Second):
		t.Fatal("service did not start reading before waiting for the rest of the stream")
	}

	close(secondChunkReleased)

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("UploadPart did not finish after the remaining chunk was released")
	}
}

func TestUploadPart_RejectsHeaderAfterFirstMessage(t *testing.T) {
	t.Parallel()

	serviceCalled := false
	svc := testStorageService{
		uploadPartFn: func(ctx context.Context, uploadID string, partNumber int32, reader io.Reader) (string, error) {
			serviceCalled = true
			_, err := io.ReadAll(reader)
			require.Error(t, err)
			return "", err
		},
	}

	stream := &uploadPartServerMock{
		ctx: context.Background(),
		requests: []*desc.UploadPartRequest{
			{
				Payload: &desc.UploadPartRequest_Header{
					Header: &desc.UploadPartHeader{
						UploadId:   "upload-1",
						PartNumber: 3,
						Data:       []byte("hello "),
					},
				},
			},
			{
				Payload: &desc.UploadPartRequest_Header{
					Header: &desc.UploadPartHeader{
						UploadId: "upload-2",
						Data:     []byte("world"),
					},
				},
			},
		},
	}

	h := handlerStorage.NewHandler(svc)
	err := h.UploadPart(stream)
	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	require.Contains(t, err.Error(), "messages after the first must be upload_part chunks")
	require.True(t, serviceCalled)
	require.Nil(t, stream.closedWith)
}

func TestUploadPart_RejectsChunkAsFirstMessage(t *testing.T) {
	t.Parallel()

	h := handlerStorage.NewHandler(testStorageService{})
	err := h.UploadPart(&uploadPartServerMock{
		ctx: context.Background(),
		requests: []*desc.UploadPartRequest{
			{
				Payload: &desc.UploadPartRequest_Chunk{
					Chunk: &desc.UploadPartChunk{Data: []byte("hello")},
				},
			},
		},
	})

	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	require.Contains(t, err.Error(), "first message must be upload_part header")
}

func TestUploadPart_RejectsHeaderWithPartNumberAfterFirstMessage(t *testing.T) {
	t.Parallel()

	serviceCalled := false
	svc := testStorageService{
		uploadPartFn: func(ctx context.Context, uploadID string, partNumber int32, reader io.Reader) (string, error) {
			serviceCalled = true
			_, err := io.ReadAll(reader)
			require.Error(t, err)
			return "", err
		},
	}

	stream := &uploadPartServerMock{
		ctx: context.Background(),
		requests: []*desc.UploadPartRequest{
			{
				Payload: &desc.UploadPartRequest_Header{
					Header: &desc.UploadPartHeader{
						UploadId:   "upload-1",
						PartNumber: 3,
						Data:       []byte("hello "),
					},
				},
			},
			{
				Payload: &desc.UploadPartRequest_Header{
					Header: &desc.UploadPartHeader{
						PartNumber: 4,
						Data:       []byte("world"),
					},
				},
			},
		},
	}

	h := handlerStorage.NewHandler(svc)
	err := h.UploadPart(stream)
	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	require.Contains(t, err.Error(), "messages after the first must be upload_part chunks")
	require.True(t, serviceCalled)
	require.Nil(t, stream.closedWith)
}

func TestUploadPart_RequiresUploadIDInFirstMessage(t *testing.T) {
	t.Parallel()

	h := handlerStorage.NewHandler(testStorageService{})
	err := h.UploadPart(&uploadPartServerMock{
		ctx: context.Background(),
		requests: []*desc.UploadPartRequest{
			{
				Payload: &desc.UploadPartRequest_Header{
					Header: &desc.UploadPartHeader{
						PartNumber: 1,
						Data:       []byte("hello"),
					},
				},
			},
		},
	})

	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	require.Contains(t, err.Error(), "upload_id is required in the first message")
}

func TestUploadPart_RequiresPartNumberInFirstMessage(t *testing.T) {
	t.Parallel()

	h := handlerStorage.NewHandler(testStorageService{})
	err := h.UploadPart(&uploadPartServerMock{
		ctx: context.Background(),
		requests: []*desc.UploadPartRequest{
			{
				Payload: &desc.UploadPartRequest_Header{
					Header: &desc.UploadPartHeader{
						UploadId: "upload-1",
						Data:     []byte("hello"),
					},
				},
			},
		},
	})

	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	require.Contains(t, err.Error(), "part_number is required in the first message")
}

type uploadPartServerMock struct {
	grpc.ServerStream
	ctx        context.Context
	requests   []*desc.UploadPartRequest
	recvIndex  int
	closedWith *desc.UploadPartResponse
	beforeRecv func(recvIndex int)
}

func (s *uploadPartServerMock) Context() context.Context {
	return s.ctx
}

func (s *uploadPartServerMock) SendAndClose(resp *desc.UploadPartResponse) error {
	s.closedWith = resp
	return nil
}

func (s *uploadPartServerMock) Recv() (*desc.UploadPartRequest, error) {
	if s.beforeRecv != nil {
		s.beforeRecv(s.recvIndex)
	}

	if s.recvIndex >= len(s.requests) {
		return nil, io.EOF
	}

	req := s.requests[s.recvIndex]
	s.recvIndex++
	return req, nil
}

func (s *uploadPartServerMock) SetHeader(metadata.MD) error {
	return nil
}

func (s *uploadPartServerMock) SendHeader(metadata.MD) error {
	return nil
}

func (s *uploadPartServerMock) SetTrailer(metadata.MD) {}

func (s *uploadPartServerMock) SendMsg(any) error {
	return nil
}

func (s *uploadPartServerMock) RecvMsg(any) error {
	return nil
}
