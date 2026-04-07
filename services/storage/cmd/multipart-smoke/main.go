package main

import (
	"bytes"
	"context"
	"crypto/md5"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	desc "github.com/alesplll/opens3-rebac/shared/pkg/go/storage/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type config struct {
	addr        string
	contentType string
	partOne     string
	partTwo     string
	chunkSize   int
	timeout     time.Duration
}

func main() {
	cfg := parseFlags()

	if cfg.chunkSize < 1 {
		fail("chunk-size must be >= 1")
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.timeout)
	defer cancel()

	conn, err := grpc.NewClient(cfg.addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		failf("dial storage: %v", err)
	}
	defer func() {
		_ = conn.Close()
	}()

	client := desc.NewDataStorageServiceClient(conn)

	partOne := []byte(cfg.partOne)
	partTwo := []byte(cfg.partTwo)

	uploadID, err := initiateMultipart(ctx, client, cfg.contentType)
	if err != nil {
		failf("initiate multipart: %v", err)
	}

	checksumOne, err := uploadPart(ctx, client, uploadID, 1, partOne, cfg.chunkSize)
	if err != nil {
		failf("upload part 1: %v", err)
	}

	checksumTwo, err := uploadPart(ctx, client, uploadID, 2, partTwo, cfg.chunkSize)
	if err != nil {
		failf("upload part 2: %v", err)
	}

	blobID, finalChecksum, err := completeMultipart(ctx, client, uploadID, checksumOne, checksumTwo)
	if err != nil {
		failf("complete multipart: %v", err)
	}

	body, err := retrieveBlob(ctx, client, blobID)
	if err != nil {
		failf("retrieve blob: %v", err)
	}

	expectedBody := append(append([]byte(nil), partOne...), partTwo...)
	expectedChecksum := md5Hex(expectedBody)

	if !bytes.Equal(body, expectedBody) {
		failf("retrieved body mismatch: got %q, want %q", string(body), string(expectedBody))
	}
	if finalChecksum != expectedChecksum {
		failf("final checksum mismatch: got %s, want %s", finalChecksum, expectedChecksum)
	}

	fmt.Printf("multipart smoke OK\n")
	fmt.Printf("upload_id: %s\n", uploadID)
	fmt.Printf("part_1_md5: %s\n", checksumOne)
	fmt.Printf("part_2_md5: %s\n", checksumTwo)
	fmt.Printf("blob_id: %s\n", blobID)
	fmt.Printf("blob_md5: %s\n", finalChecksum)
	fmt.Printf("size_bytes: %d\n", len(body))
}

func parseFlags() config {
	cfg := config{}
	flag.StringVar(&cfg.addr, "addr", "localhost:50053", "storage gRPC address")
	flag.StringVar(&cfg.contentType, "content-type", "text/plain", "multipart content type")
	flag.StringVar(&cfg.partOne, "part1", "hello ", "multipart part 1 payload")
	flag.StringVar(&cfg.partTwo, "part2", "world", "multipart part 2 payload")
	flag.IntVar(&cfg.chunkSize, "chunk-size", 3, "bytes per streamed chunk")
	flag.DurationVar(&cfg.timeout, "timeout", 15*time.Second, "overall smoke timeout")
	flag.Parse()
	return cfg
}

func initiateMultipart(ctx context.Context, client desc.DataStorageServiceClient, contentType string) (string, error) {
	resp, err := client.InitiateMultipartUpload(ctx, &desc.InitiateMultipartUploadRequest{
		ExpectedParts: 2,
		ContentType:   contentType,
	})
	if err != nil {
		return "", err
	}

	return resp.GetUploadId(), nil
}

func uploadPart(ctx context.Context, client desc.DataStorageServiceClient, uploadID string, partNumber int32, data []byte, chunkSize int) (string, error) {
	stream, err := client.UploadPart(ctx)
	if err != nil {
		return "", err
	}

	reader := bytes.NewReader(data)
	chunk := make([]byte, chunkSize)
	first := true

	for {
		n, readErr := reader.Read(chunk)
		if n > 0 {
			req := &desc.UploadPartRequest{
				Data: chunk[:n],
			}
			if first {
				req.UploadId = uploadID
				req.PartNumber = partNumber
				first = false
			}
			if err := stream.Send(req); err != nil {
				return "", err
			}
		}

		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return "", readErr
		}
	}

	if first {
		if err := stream.Send(&desc.UploadPartRequest{
			UploadId:   uploadID,
			PartNumber: partNumber,
		}); err != nil {
			return "", err
		}
	}

	resp, err := stream.CloseAndRecv()
	if err != nil {
		return "", err
	}

	return resp.GetPartChecksumMd5(), nil
}

func completeMultipart(ctx context.Context, client desc.DataStorageServiceClient, uploadID string, checksumOne string, checksumTwo string) (string, string, error) {
	resp, err := client.CompleteMultipartUpload(ctx, &desc.CompleteMultipartUploadRequest{
		UploadId: uploadID,
		Parts: []*desc.PartInfo{
			{PartNumber: 1, ChecksumMd5: checksumOne},
			{PartNumber: 2, ChecksumMd5: checksumTwo},
		},
	})
	if err != nil {
		return "", "", err
	}

	return resp.GetBlobId(), resp.GetChecksumMd5(), nil
}

func retrieveBlob(ctx context.Context, client desc.DataStorageServiceClient, blobID string) ([]byte, error) {
	stream, err := client.RetrieveObject(ctx, &desc.RetrieveObjectRequest{
		BlobId: blobID,
	})
	if err != nil {
		return nil, err
	}

	var body bytes.Buffer
	for {
		resp, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		if _, err := body.Write(resp.GetData()); err != nil {
			return nil, err
		}
	}

	return body.Bytes(), nil
}

func md5Hex(data []byte) string {
	sum := md5.Sum(data)
	return fmt.Sprintf("%x", sum)
}

func fail(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}

func failf(format string, args ...any) {
	fail(fmt.Sprintf(format, args...))
}
