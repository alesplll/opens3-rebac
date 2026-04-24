package metadata

import (
	"context"

	metadatav1 "github.com/alesplll/opens3-rebac/shared/pkg/go/metadata/v1"
)

func (h *handler) DeleteBucket(ctx context.Context, req *metadatav1.DeleteBucketRequest) (*metadatav1.DeleteBucketResponse, error) {
	if err := h.bucketService.DeleteBucket(ctx, req.GetBucketName()); err != nil {
		return nil, err
	}

	return &metadatav1.DeleteBucketResponse{Success: true}, nil
}
