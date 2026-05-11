package metadata

import (
	"context"

	metadatav1 "github.com/alesplll/opens3-rebac/shared/pkg/go/metadata/v1"
)

func (h *handler) CreateBucket(ctx context.Context, req *metadatav1.CreateBucketRequest) (*metadatav1.CreateBucketResponse, error) {
	bucket, err := h.bucketService.CreateBucket(ctx, req.GetName(), req.GetOwnerId())
	if err != nil {
		return nil, err
	}

	return &metadatav1.CreateBucketResponse{
		BucketId:  bucket.ID,
		CreatedAt: bucket.CreatedAt.UnixMilli(),
	}, nil
}
