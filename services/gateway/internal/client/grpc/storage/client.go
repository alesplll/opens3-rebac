package storage

import (
	"context"
	"io"
	"time"

	grpcclient "github.com/alesplll/opens3-rebac/services/gateway/internal/client/grpc"
	storagev1 "github.com/alesplll/opens3-rebac/shared/pkg/go/storage/v1"
)

type client struct {
	client        storagev1.DataStorageServiceClient
	timeout       time.Duration
	streamTimeout time.Duration
}

func NewClient(grpcSvc storagev1.DataStorageServiceClient, timeout, streamTimeout time.Duration) grpcclient.StorageClient {
	return &client{client: grpcSvc, timeout: timeout, streamTimeout: streamTimeout}
}

func (c *client) StoreObject(ctx context.Context, chunks <-chan *storagev1.StoreObjectRequest) (*storagev1.StoreObjectResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.streamTimeout)
	defer cancel()

	stream, err := c.client.StoreObject(ctx)
	if err != nil {
		return nil, err
	}

	for chunk := range chunks {
		if err := stream.Send(chunk); err != nil {
			return nil, err
		}
	}

	return stream.CloseAndRecv()
}

func (c *client) RetrieveObject(ctx context.Context, req *storagev1.RetrieveObjectRequest, writer io.Writer) (*storagev1.RetrieveObjectResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.streamTimeout)
	defer cancel()

	stream, err := c.client.RetrieveObject(ctx, req)
	if err != nil {
		return nil, err
	}

	var first *storagev1.RetrieveObjectResponse
	for {
		chunk, recvErr := stream.Recv()
		if recvErr == io.EOF {
			if first == nil {
				return &storagev1.RetrieveObjectResponse{}, nil
			}
			return first, nil
		}
		if recvErr != nil {
			return nil, recvErr
		}

		if first == nil {
			first = &storagev1.RetrieveObjectResponse{TotalSize: chunk.GetTotalSize()}
		}

		if len(chunk.GetData()) > 0 {
			if _, err := writer.Write(chunk.GetData()); err != nil {
				return nil, err
			}
		}
	}
}

func (c *client) DeleteObject(ctx context.Context, req *storagev1.DeleteObjectRequest) (*storagev1.DeleteObjectResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	return c.client.DeleteObject(ctx, req)
}

func (c *client) InitiateMultipartUpload(ctx context.Context, req *storagev1.InitiateMultipartUploadRequest) (*storagev1.InitiateMultipartUploadResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	return c.client.InitiateMultipartUpload(ctx, req)
}

func (c *client) UploadPart(ctx context.Context, chunks <-chan *storagev1.UploadPartRequest) (*storagev1.UploadPartResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.streamTimeout)
	defer cancel()

	stream, err := c.client.UploadPart(ctx)
	if err != nil {
		return nil, err
	}

	for chunk := range chunks {
		if err := stream.Send(chunk); err != nil {
			return nil, err
		}
	}

	return stream.CloseAndRecv()
}

func (c *client) CompleteMultipartUpload(ctx context.Context, req *storagev1.CompleteMultipartUploadRequest) (*storagev1.CompleteMultipartUploadResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	return c.client.CompleteMultipartUpload(ctx, req)
}

func (c *client) AbortMultipartUpload(ctx context.Context, req *storagev1.AbortMultipartUploadRequest) (*storagev1.AbortMultipartUploadResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	return c.client.AbortMultipartUpload(ctx, req)
}

func (c *client) HealthCheck(ctx context.Context, req *storagev1.HealthCheckRequest) (*storagev1.HealthCheckResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	return c.client.HealthCheck(ctx, req)
}
