package storage

import (
	"context"

	desc "github.com/alesplll/opens3-rebac/shared/pkg/storage/v1"
)

func (h *handler) HealthCheck(ctx context.Context, req *desc.HealthCheckRequest) (*desc.HealthCheckResponse, error) {
	serving, err := h.service.HealthCheck(ctx, req.GetService())
	if err != nil {
		return &desc.HealthCheckResponse{
			Status: desc.HealthCheckResponse_NOT_SERVING,
		}, nil
	}

	if !serving {
		return &desc.HealthCheckResponse{
			Status: desc.HealthCheckResponse_NOT_SERVING,
		}, nil
	}

	return &desc.HealthCheckResponse{
		Status: desc.HealthCheckResponse_SERVING,
	}, nil
}
