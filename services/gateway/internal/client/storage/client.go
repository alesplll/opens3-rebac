package storage

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"

	storagev1 "github.com/alesplll/opens3-rebac/shared/pkg/go/storage/v1"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const (
	uploadChunkSize            = 64 * 1024
	storageMessageOverheadSize = 64 * 1024
)

var ErrEmptyStream = errors.New("storage returned an empty stream")

type BodyReadError struct{ Err error }

func (e *BodyReadError) Error() string { return e.Err.Error() }
func (e *BodyReadError) Unwrap() error { return e.Err }

type StoredObject struct {
	BlobID string
	ETag   string
	Size   int64
}

type Client interface {
	Store(ctx context.Context, body io.Reader, size *int64, contentType string) (StoredObject, error)
	Retrieve(ctx context.Context, blobID string, expectedSize int64) (io.ReadCloser, error)
}

type client struct {
	grpc                   storagev1.DataStorageServiceClient
	retrieveChunkSizeBytes int
}

func NewClient(grpcClient storagev1.DataStorageServiceClient, retrieveChunkSizeBytes int) Client {
	return &client{grpc: grpcClient, retrieveChunkSizeBytes: retrieveChunkSizeBytes}
}

func (c *client) Store(ctx context.Context, body io.Reader, size *int64, contentType string) (result StoredObject, resultErr error) {
	ctx, span := otel.Tracer("gateway.storage").Start(ctx, "storage.StoreObject", trace.WithSpanKind(trace.SpanKindClient))
	defer func() { finishSpan(span, resultErr) }()

	streamCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	stream, err := c.grpc.StoreObject(withTraceContext(streamCtx))
	if err != nil {
		return StoredObject{}, err
	}
	header := &storagev1.StoreObjectHeader{ContentType: contentType, Size: size}
	if err := stream.Send(&storagev1.StoreObjectRequest{Payload: &storagev1.StoreObjectRequest_Header{Header: header}}); err != nil {
		return StoredObject{}, storeSendError(stream, err)
	}

	buffer := make([]byte, uploadChunkSize)
	var total int64
	emptyReads := 0
	for {
		n, readErr := body.Read(buffer)
		if n > 0 {
			emptyReads = 0
			if err := stream.Send(&storagev1.StoreObjectRequest{Payload: &storagev1.StoreObjectRequest_Chunk{
				Chunk: &storagev1.StoreObjectChunk{Data: buffer[:n]},
			}}); err != nil {
				return StoredObject{}, storeSendError(stream, err)
			}
			total += int64(n)
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return StoredObject{}, &BodyReadError{Err: readErr}
		}
		if n == 0 {
			emptyReads++
			if emptyReads >= 100 {
				return StoredObject{}, &BodyReadError{Err: io.ErrNoProgress}
			}
		}
	}

	response, err := stream.CloseAndRecv()
	if err != nil {
		return StoredObject{}, err
	}
	return StoredObject{BlobID: response.GetBlobId(), ETag: response.GetChecksumMd5(), Size: total}, nil
}

func storeSendError(stream storagev1.DataStorageService_StoreObjectClient, sendErr error) error {
	if errors.Is(sendErr, io.EOF) {
		if _, err := stream.CloseAndRecv(); err != nil {
			return err
		}
	}
	return sendErr
}

func (c *client) Retrieve(ctx context.Context, blobID string, expectedSize int64) (io.ReadCloser, error) {
	ctx, span := otel.Tracer("gateway.storage").Start(ctx, "storage.RetrieveObject", trace.WithSpanKind(trace.SpanKindClient))
	streamCtx, cancel := context.WithCancel(ctx)
	stream, err := c.grpc.RetrieveObject(
		withTraceContext(streamCtx),
		&storagev1.RetrieveObjectRequest{BlobId: blobID},
		grpc.MaxCallRecvMsgSize(c.retrieveChunkSizeBytes+storageMessageOverheadSize),
	)
	if err != nil {
		cancel()
		finishSpan(span, err)
		return nil, err
	}
	first, err := stream.Recv()
	if err != nil && !errors.Is(err, io.EOF) {
		cancel()
		finishSpan(span, err)
		return nil, err
	}
	if errors.Is(err, io.EOF) {
		if expectedSize != 0 {
			cancel()
			finishSpan(span, ErrEmptyStream)
			return nil, ErrEmptyStream
		}
		finishSpan(span, nil)
		return &streamReader{cancel: cancel}, nil
	}
	return &streamReader{
		stream: stream,
		buffer: bytes.NewReader(first.GetData()),
		cancel: cancel,
		span:   span,
	}, nil
}

type streamReader struct {
	stream storagev1.DataStorageService_RetrieveObjectClient
	buffer *bytes.Reader
	cancel context.CancelFunc
	span   trace.Span
	end    sync.Once
}

func (r *streamReader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	for {
		if r.buffer != nil && r.buffer.Len() > 0 {
			return r.buffer.Read(p)
		}
		if r.stream == nil {
			return 0, io.EOF
		}
		chunk, err := r.stream.Recv()
		if err != nil {
			r.finish(err)
			return 0, err
		}
		r.buffer = bytes.NewReader(chunk.GetData())
	}
}

func (r *streamReader) Close() error {
	if r.cancel != nil {
		r.cancel()
	}
	r.finish(nil)
	return nil
}

func (r *streamReader) finish(err error) {
	r.end.Do(func() { finishSpan(r.span, err) })
}

func finishSpan(span trace.Span, err error) {
	if span == nil {
		return
	}
	if err != nil && !errors.Is(err, io.EOF) {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	span.End()
}

func withTraceContext(ctx context.Context) context.Context {
	header := make(http.Header)
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(header))
	outgoing, ok := metadata.FromOutgoingContext(ctx)
	if ok {
		outgoing = outgoing.Copy()
	} else {
		outgoing = metadata.New(nil)
	}
	for key, values := range header {
		outgoing.Append(strings.ToLower(key), values...)
	}
	return metadata.NewOutgoingContext(ctx, outgoing)
}
