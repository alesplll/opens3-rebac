package metadata

import (
	"context"

	metadatav1 "github.com/alesplll/opens3-rebac/shared/pkg/go/metadata/v1"
)

func (h *handler) HeadBucket(ctx context.Context, req *metadatav1.HeadBucketRequest) (*metadatav1.HeadBucketResponse, error) {
	exists, bucketID, ownerID, err := h.bucketService.HeadBucket(ctx, req.GetBucketName())
	if err != nil {
		return nil, err
	}

	return &metadatav1.HeadBucketResponse{
		Exists:   exists,
		BucketId: bucketID,
		OwnerId:  ownerID,
	}, nil
}
