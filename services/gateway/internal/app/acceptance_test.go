package app

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc/connectivity"
)

func acceptanceApp(t *testing.T) *App {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := listener.Addr().String()
	listener.Close()
	t.Setenv("GATEWAY_HTTP_ADDR", addr)
	t.Setenv("METADATA_GRPC_ADDR", "127.0.0.1:1")
	t.Setenv("STORAGE_GRPC_ADDR", "127.0.0.1:1")
	t.Setenv("LOGGER_ENABLE_OLTP", "false")
	app, err := NewApp("")
	if err != nil {
		t.Fatal(err)
	}
	return app
}

func TestConfiguredClientsAreReused(t *testing.T) {
	a := acceptanceApp(t)
	defer a.serviceProvider.Close()
	m1, err := a.serviceProvider.MetadataClient()
	if err != nil {
		t.Fatal(err)
	}
	m2, err := a.serviceProvider.MetadataClient()
	if err != nil {
		t.Fatal(err)
	}
	s1, err := a.serviceProvider.StorageClient()
	if err != nil {
		t.Fatal(err)
	}
	s2, err := a.serviceProvider.StorageClient()
	if err != nil {
		t.Fatal(err)
	}
	if m1 != m2 || s1 != s2 {
		t.Fatal("downstream clients are not reused")
	}
	if a.serviceProvider.metadataConn.Target() != a.config.Metadata.Address() || a.serviceProvider.storageConn.Target() != a.config.Storage.Address() {
		t.Fatal("downstream target differs from configuration")
	}
}

func TestUnavailableMetadataDoesNotReturnHTTPSuccess(t *testing.T) {
	a := acceptanceApp(t)
	defer a.serviceProvider.Close()
	for _, method := range []string{http.MethodPut, http.MethodGet} {
		t.Run(method, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			req := httptest.NewRequest(method, "/fixture/key", strings.NewReader("data")).WithContext(ctx)
			response := httptest.NewRecorder()
			a.httpServer.Handler.ServeHTTP(response, req)
			if response.Code != http.StatusServiceUnavailable {
				t.Fatalf("HTTP %d, want 503", response.Code)
			}
		})
	}
}

func TestShutdownDrainsActiveRequestBeforeClosingClients(t *testing.T) {
	a := acceptanceApp(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	entered, release := make(chan struct{}), make(chan struct{})
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
	}()
	a.httpServer.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			w.WriteHeader(http.StatusOK)
			return
		}
		close(entered)
		select {
		case <-release:
		case <-r.Context().Done():
			return
		}
		io.WriteString(w, "completed")
	})
	done := make(chan error, 1)
	go func() { done <- a.Run(ctx) }()
	client := &http.Client{Timeout: 3 * time.Second}
	defer client.CloseIdleConnections()
	url := "http://" + a.config.HTTP.Address()
	deadline := time.Now().Add(3 * time.Second)
	for {
		response, err := client.Get(url + "/healthz")
		if err == nil {
			response.Body.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatal(err)
		}
		time.Sleep(10 * time.Millisecond)
	}
	requestDone := make(chan error, 1)
	go func() {
		response, err := client.Get(url + "/object")
		if err == nil {
			defer response.Body.Close()
			var body []byte
			body, err = io.ReadAll(response.Body)
			if err == nil && string(body) != "completed" {
				err = io.ErrUnexpectedEOF
			}
		}
		requestDone <- err
	}()
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		t.Fatal("request did not start")
	}
	cancel()
	select {
	case err := <-done:
		t.Fatalf("shutdown returned before request completion: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
	if a.serviceProvider.storageConn.GetState() == connectivity.Shutdown || a.serviceProvider.metadataConn.GetState() == connectivity.Shutdown {
		t.Fatal("downstream clients closed before drain")
	}
	close(release)
	select {
	case err := <-requestDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("active request did not finish")
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("shutdown did not finish")
	}
	if a.serviceProvider.storageConn.GetState() != connectivity.Shutdown || a.serviceProvider.metadataConn.GetState() != connectivity.Shutdown {
		t.Fatal("downstream clients not closed")
	}
}
