package quota

import (
	"context"
	"time"

	grpcclient "github.com/alesplll/opens3-rebac/services/gateway/internal/client/grpc"
	quotav1 "github.com/alesplll/opens3-rebac/shared/pkg/go/quota/v1"
)

type client struct {
	client  quotav1.QuotaServiceClient
	timeout time.Duration
}

func NewClient(grpcSvc quotav1.QuotaServiceClient, timeout time.Duration) grpcclient.QuotaClient {
	return &client{client: grpcSvc, timeout: timeout}
}

func (c *client) CheckQuota(ctx context.Context, req *quotav1.CheckQuotaRequest) (*quotav1.CheckQuotaResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	return c.client.CheckQuota(ctx, req)
}

func (c *client) UpdateUsage(ctx context.Context, req *quotav1.UpdateUsageRequest) (*quotav1.UpdateUsageResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	return c.client.UpdateUsage(ctx, req)
}

func (c *client) HealthCheck(ctx context.Context, req *quotav1.HealthCheckRequest) (*quotav1.HealthCheckResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	return c.client.HealthCheck(ctx, req)
}
