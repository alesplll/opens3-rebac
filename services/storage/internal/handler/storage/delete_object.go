package storage

import (
	"context"

	desc "github.com/alesplll/opens3-rebac/shared/pkg/go/storage/v1"
)

func (h *handler) DeleteObject(ctx context.Context, req *desc.DeleteObjectRequest) (*desc.DeleteObjectResponse, error) {
	err := h.service.DeleteObject(ctx, req.GetBlobId())
	if err != nil {
		return nil, err
	}

	return &desc.DeleteObjectResponse{
		Success: true,
	}, nil
}
