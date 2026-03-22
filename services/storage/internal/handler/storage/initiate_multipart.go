package storage

import (
	"context"

	desc "github.com/alesplll/opens3-rebac/shared/pkg/storage/v1"
)

func (h *handler) InitiateMultipartUpload(ctx context.Context, req *desc.InitiateMultipartUploadRequest) (*desc.InitiateMultipartUploadResponse, error) {
	uploadID, err := h.service.InitiateMultipartUpload(ctx, req.GetExpectedParts(), req.GetContentType())
	if err != nil {
		return nil, err
	}

	return &desc.InitiateMultipartUploadResponse{
		UploadId: uploadID,
	}, nil
}
