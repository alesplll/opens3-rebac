package metadata

import (
	"context"

	metadatav1 "github.com/alesplll/opens3-rebac/shared/pkg/go/metadata/v1"
)

func (h *handler) HealthCheck(ctx context.Context, _ *metadatav1.HealthCheckRequest) (*metadatav1.HealthCheckResponse, error) {
	pgOK, kafkaOK := h.objectService.HealthCheck(ctx)

	status := metadatav1.HealthCheckResponse_SERVING
	if !pgOK || !kafkaOK {
		status = metadatav1.HealthCheckResponse_NOT_SERVING
	}

	return &metadatav1.HealthCheckResponse{Status: status}, nil
}
