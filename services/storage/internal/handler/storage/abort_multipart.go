package storage

import (
	"context"

	desc "github.com/alesplll/opens3-rebac/shared/pkg/storage/v1"
)

func (h *handler) AbortMultipartUpload(ctx context.Context, req *desc.AbortMultipartUploadRequest) (*desc.AbortMultipartUploadResponse, error) {
	err := h.service.AbortMultipartUpload(ctx, req.GetUploadId())
	if err != nil {
		return nil, err
	}

	return &desc.AbortMultipartUploadResponse{
		Success: true,
	}, nil
}
