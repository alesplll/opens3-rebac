package app_test

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/alesplll/opens3-rebac/services/gateway/internal/app"
)

func TestRunServesHealthAndShutsDown(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	t.Setenv("GATEWAY_HTTP_ADDR", address)
	t.Setenv("METADATA_GRPC_ADDR", "127.0.0.1:1")
	t.Setenv("STORAGE_GRPC_ADDR", "127.0.0.1:1")
	a, err := app.NewApp(".env")
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		done <- a.Run(ctx)
	}()

	client := &http.Client{Timeout: time.Second}
	deadline := time.Now().Add(3 * time.Second)
	for {
		response, err := client.Get("http://" + address + "/healthz")
		if err == nil {
			response.Body.Close()
			if response.StatusCode != http.StatusOK {
				t.Fatalf("health status = %d, want 200", response.StatusCode)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("Gateway did not start: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Gateway shutdown: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Gateway did not stop after cancellation")
	}
}

func TestRunReturnsListenError(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	t.Setenv("GATEWAY_HTTP_ADDR", listener.Addr().String())
	t.Setenv("METADATA_GRPC_ADDR", "127.0.0.1:1")
	t.Setenv("STORAGE_GRPC_ADDR", "127.0.0.1:1")
	a, err := app.NewApp("")
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Run(context.Background()); err == nil {
		t.Fatal("Run() should report occupied address")
	}
}

func TestNewAppRejectsInvalidStorageChunkSize(t *testing.T) {
	t.Setenv("STORAGE_RETRIEVE_CHUNK_SIZE_BYTES", "0")
	if _, err := app.NewApp(""); err == nil {
		t.Fatal("NewApp() should reject invalid storage chunk size")
	}
}
