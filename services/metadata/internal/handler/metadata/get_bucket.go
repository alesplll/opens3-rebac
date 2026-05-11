package metadata

import (
	"context"

	metadatav1 "github.com/alesplll/opens3-rebac/shared/pkg/go/metadata/v1"
)

func (h *handler) GetBucket(ctx context.Context, req *metadatav1.GetBucketRequest) (*metadatav1.GetBucketResponse, error) {
	bucket, err := h.bucketService.GetBucket(ctx, req.GetBucketName())
	if err != nil {
		return nil, err
	}

	return &metadatav1.GetBucketResponse{
		Bucket: &metadatav1.BucketInfo{
			BucketId:  bucket.ID,
			Name:      bucket.Name,
			OwnerId:   bucket.OwnerID,
			CreatedAt: bucket.CreatedAt.UnixMilli(),
		},
	}, nil
}
