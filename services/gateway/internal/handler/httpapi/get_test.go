package httpapi_test

import (
	"context"
	"errors"
	metadatav1 "github.com/alesplll/opens3-rebac/shared/pkg/go/metadata/v1"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alesplll/opens3-rebac/services/gateway/internal/handler/httpapi"
	"github.com/alesplll/opens3-rebac/services/gateway/internal/service"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/logger"
)

// The transport test supplies only Get; other operations must not be called.
type getService struct {
	service.ObjectService
	result service.GetResult
}

func (s getService) Get(context.Context, string, string) (service.GetResult, error) {
	return s.result, nil
}

type failedReader struct{}

func (failedReader) Read([]byte) (int, error) { return 0, errors.New("storage stream failed") }

func TestGetAbortsIncompleteResponse(t *testing.T) {
	for _, tc := range []struct {
		name string
		body io.Reader
		size int64
	}{
		{"short clean EOF", strings.NewReader("abc"), 6},
		{"failure after declared bytes", io.MultiReader(strings.NewReader("abc"), failedReader{}), 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			logger.SetNopLogger()
			handler := httpapi.NewHandler(getService{result: service.GetResult{Body: tc.body, Size: tc.size}})
			defer func() {
				if got := recover(); got != http.ErrAbortHandler {
					t.Fatalf("panic = %v, want http.ErrAbortHandler", got)
				}
			}()
			handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/bucket/key", nil))
		})
	}
}

func TestGetStreamsBeforeStorageEOF(t *testing.T) {
	server, state := newGateway(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	next := make(chan struct{})
	state.beforeNextChunk = func(ctx context.Context) error {
		select {
		case <-next:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	const size = 9 * 1024 * 1024
	state.blobs["stream-blob"] = []byte(strings.Repeat("x", size))
	state.meta["stream"] = &metadatav1.GetObjectMetaResponse{BlobId: "stream-blob", SizeBytes: size}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/fixture/stream", nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := server.Client().Do(req)
	if err != nil {
		t.Fatalf("Gateway did not return a prefix before Storage EOF: %v", err)
	}
	defer response.Body.Close()
	first := make([]byte, 1)
	if _, err := io.ReadFull(response.Body, first); err != nil {
		t.Fatal(err)
	}
	if first[0] != 'x' {
		t.Fatal("unexpected first byte")
	}
	close(next)
	n, err := io.Copy(io.Discard, response.Body)
	if err != nil || n != size-1 {
		t.Fatalf("remaining body = %d, error = %v", n, err)
	}
}

func TestClientCancellationStopsStorageRead(t *testing.T) {
	server, state := newGateway(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	stopped := make(chan struct{})
	state.beforeNextChunk = func(ctx context.Context) error {
		<-ctx.Done()
		close(stopped)
		return ctx.Err()
	}
	state.blobs["blob"] = []byte(strings.Repeat("x", 9*1024*1024))
	state.meta["key"] = &metadatav1.GetObjectMetaResponse{BlobId: "blob", SizeBytes: 9 * 1024 * 1024}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/fixture/key", nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := server.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("GET status %d", response.StatusCode)
	}
	cancel()
	select {
	case <-stopped:
	case <-time.After(3 * time.Second):
		t.Fatal("Storage stream survived HTTP client cancellation")
	}
}

func TestMissingObjectReturnsNotFound(t *testing.T) {
	server, _ := newGateway(t)
	response, err := server.Client().Get(server.URL + "/fixture/missing")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("GET status %d, want 404", response.StatusCode)
	}
}
