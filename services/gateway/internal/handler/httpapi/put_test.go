package httpapi_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestPutStreamsBeforeEOFAndPublishesOnlyAfterStorageCommit(t *testing.T) {
	server, state := newGateway(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	chunkReceived, atCommit, allowCommit := make(chan struct{}), make(chan struct{}), make(chan struct{})
	var once sync.Once
	state.onChunk = func() { once.Do(func() { close(chunkReceived) }) }
	state.beforeCommit = func(ctx context.Context) error {
		close(atCommit)
		select {
		case <-allowCommit:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	reader, writer := io.Pipe()
	defer reader.Close()
	defer writer.Close()
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, server.URL+"/fixture/stream", reader)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		response, err := server.Client().Do(req)
		if err == nil {
			response.Body.Close()
			if response.StatusCode != http.StatusOK {
				err = status.Errorf(codes.Internal, "HTTP %d", response.StatusCode)
			}
		}
		done <- err
	}()
	if _, err := io.WriteString(writer, strings.Repeat("x", 128*1024)); err != nil {
		t.Fatal(err)
	}
	select {
	case <-chunkReceived:
	case <-ctx.Done():
		t.Fatal("Gateway buffered PUT until EOF")
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-atCommit:
	case <-ctx.Done():
		t.Fatal("Storage did not reach commit")
	}
	state.Lock()
	published := len(state.meta)
	state.Unlock()
	if published != 0 {
		t.Fatal("Metadata published before Storage acknowledged commit")
	}
	close(allowCommit)
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	state.Lock()
	defer state.Unlock()
	if len(state.meta) != 1 {
		t.Fatal("successful PUT did not publish Metadata")
	}
}

func TestStorageRejectsUploadWithoutPublishingMetadata(t *testing.T) {
	server, state := newGateway(t)
	state.failStore = status.Error(codes.ResourceExhausted, "disk full")
	request, err := http.NewRequest(http.MethodPut, server.URL+"/fixture/key", strings.NewReader(strings.Repeat("x", 8*1024*1024)))
	if err != nil {
		t.Fatal(err)
	}
	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusInsufficientStorage {
		t.Fatalf("status = %d, want 507", response.StatusCode)
	}
	state.Lock()
	defer state.Unlock()
	if len(state.meta) != 0 || state.storeCount != 0 {
		t.Fatal("failed upload published data")
	}
}
