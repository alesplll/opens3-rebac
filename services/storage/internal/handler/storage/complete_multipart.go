package storage

import (
	"context"

	"github.com/alesplll/opens3-rebac/services/storage/internal/model"
	desc "github.com/alesplll/opens3-rebac/shared/pkg/storage/v1"
)

func (h *handler) CompleteMultipartUpload(ctx context.Context, req *desc.CompleteMultipartUploadRequest) (*desc.CompleteMultipartUploadResponse, error) {
	parts := make([]model.PartInfo, 0, len(req.GetParts()))
	for _, p := range req.GetParts() {
		parts = append(parts, model.PartInfo{
			PartNumber:  p.GetPartNumber(),
			ChecksumMD5: p.GetChecksumMd5(),
		})
	}

	meta, err := h.service.CompleteMultipartUpload(ctx, req.GetUploadId(), parts)
	if err != nil {
		return nil, err
	}

	return &desc.CompleteMultipartUploadResponse{
		BlobId:      meta.BlobID,
		ChecksumMd5: meta.ChecksumMD5,
	}, nil
}
