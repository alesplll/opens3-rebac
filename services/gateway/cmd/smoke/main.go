// Command smoke checks the running HTTP -> Gateway -> Metadata/Storage slice.
package main

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	metadatav1 "github.com/alesplll/opens3-rebac/shared/pkg/go/metadata/v1"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

func main() {
	gateway := flag.String("gateway", "http://127.0.0.1:8080", "Gateway HTTP URL")
	metadata := flag.String("metadata", "127.0.0.1:50052", "Metadata gRPC address")
	timeout := flag.Duration("timeout", 2*time.Minute, "whole smoke test timeout, including readiness")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	if err := run(ctx, strings.TrimRight(*gateway, "/"), *metadata); err != nil {
		fmt.Fprintln(os.Stderr, "smoke failed:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, gateway, metadata string) error {
	conn, err := grpc.NewClient(metadata, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer conn.Close()
	client := metadatav1.NewMetadataServiceClient(conn)
	bucket := "gateway-smoke-" + uuid.NewString()
	httpClient := &http.Client{Timeout: 30 * time.Second}
	defer httpClient.CloseIdleConnections()
	// HeadBucket probes readiness without repeating a mutating RPC on an unknown outcome.
	for {
		probeCtx, cancel := context.WithTimeout(ctx, time.Second)
		_, metaErr := client.HeadBucket(probeCtx, &metadatav1.HeadBucketRequest{BucketName: bucket})
		req, err := http.NewRequestWithContext(probeCtx, http.MethodGet, gateway+"/healthz", nil)
		if err != nil {
			cancel()
			return err
		}
		response, httpErr := httpClient.Do(req)
		ready := httpErr == nil && response.StatusCode == http.StatusOK
		if response != nil {
			response.Body.Close()
		}
		cancel()
		if (metaErr == nil || status.Code(metaErr) == codes.NotFound) && ready {
			break
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("readiness: %w (Metadata: %v, HTTP: %v)", ctx.Err(), metaErr, httpErr)
		case <-time.After(250 * time.Millisecond):
		}
	}
	if _, err := client.CreateBucket(ctx, &metadatav1.CreateBucketRequest{Name: bucket, OwnerId: uuid.NewString()}); err != nil {
		return fmt.Errorf("create smoke bucket: %w", err)
	}
	fmt.Println("Smoke bucket:", bucket)
	// Each run has its own bucket. Keep its blobs for inspection; never delete user data.
	for _, tc := range []struct {
		key     string
		size    int64
		seed    byte
		chunked bool
	}{
		{"empty", 0, 0, false},
		{"nested/binary", 20*1024*1024 + 17, 7, false},
		{"chunked", 256*1024 + 13, 19, true},
		{"nested/binary", 128*1024 + 1, 31, false},
	} {
		if err := roundTrip(ctx, httpClient, gateway+"/"+bucket+"/"+tc.key, tc.size, tc.seed, tc.chunked); err != nil {
			return fmt.Errorf("%s (%d bytes): %w", tc.key, tc.size, err)
		}
		fmt.Printf("PASS %s: %d bytes, chunked=%v\n", tc.key, tc.size, tc.chunked)
	}
	return nil
}

func roundTrip(ctx context.Context, client *http.Client, url string, size int64, seed byte, chunked bool) error {
	expected, expectedMD5 := sha256.New(), md5.New()
	if _, err := io.Copy(io.MultiWriter(expected, expectedMD5), &payload{remaining: size, seed: seed}); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, &payload{remaining: size, seed: seed})
	if err != nil {
		return err
	}
	req.ContentLength = size
	if chunked {
		req.ContentLength = -1
	}
	if size == 0 {
		req.Body = http.NoBody
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	response, err := client.Do(req)
	if err != nil {
		return err
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("PUT status %s", response.Status)
	}
	etag := fmt.Sprintf("\"%x\"", expectedMD5.Sum(nil))
	if response.Header.Get("ETag") != etag {
		return fmt.Errorf("PUT ETag mismatch")
	}
	version := response.Header.Get("X-Object-Version-Id")
	if version == "" {
		return fmt.Errorf("PUT missing version id")
	}
	// No retry or delay: a successful PUT must already be immediately readable.
	get, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	response, err = client.Do(get)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("GET status %s", response.Status)
	}
	if response.ContentLength != size || response.Header.Get("ETag") != etag || response.Header.Get("X-Object-Version-Id") != version || response.Header.Get("Content-Type") != "application/octet-stream" {
		return fmt.Errorf("GET metadata mismatch: headers=%v", response.Header)
	}
	actual := sha256.New()
	n, err := io.Copy(actual, response.Body)
	if err != nil {
		return fmt.Errorf("read GET: %w", err)
	}
	if n != size || fmt.Sprintf("%x", actual.Sum(nil)) != fmt.Sprintf("%x", expected.Sum(nil)) {
		return fmt.Errorf("GET bytes mismatch")
	}
	return nil
}

type payload struct {
	remaining, offset int64
	seed              byte
}

func (p *payload) Read(buf []byte) (int, error) {
	if p.remaining == 0 {
		return 0, io.EOF
	}
	n := int(min(int64(len(buf)), p.remaining))
	for i := 0; i < n; i++ {
		buf[i] = byte((p.offset+int64(i))%251) + p.seed
	}
	p.offset += int64(n)
	p.remaining -= int64(n)
	return n, nil
}
