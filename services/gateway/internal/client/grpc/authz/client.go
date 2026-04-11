package authz

import (
	"context"
	"time"

	grpcclient "github.com/alesplll/opens3-rebac/services/gateway/internal/client/grpc"
	authzv1 "github.com/alesplll/opens3-rebac/shared/pkg/go/authz/v1"
)

type client struct {
	client  authzv1.PermissionServiceClient
	timeout time.Duration
}

func NewClient(grpcSvc authzv1.PermissionServiceClient, timeout time.Duration) grpcclient.AuthZClient {
	return &client{client: grpcSvc, timeout: timeout}
}

func (c *client) Check(ctx context.Context, req *authzv1.CheckRequest) (*authzv1.CheckResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	return c.client.Check(ctx, req)
}

func (c *client) WriteTuple(ctx context.Context, req *authzv1.WriteTupleRequest) (*authzv1.WriteTupleResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	return c.client.WriteTuple(ctx, req)
}

func (c *client) HealthCheck(ctx context.Context, req *authzv1.HealthCheckRequest) (*authzv1.HealthCheckResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	return c.client.HealthCheck(ctx, req)
}
