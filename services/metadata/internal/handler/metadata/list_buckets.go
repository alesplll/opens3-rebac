package metadata

import (
	"context"

	metadatav1 "github.com/alesplll/opens3-rebac/shared/pkg/go/metadata/v1"
)

func (h *handler) ListBuckets(ctx context.Context, req *metadatav1.ListBucketsRequest) (*metadatav1.ListBucketsResponse, error) {
	buckets, err := h.bucketService.ListBuckets(ctx, req.GetOwnerId())
	if err != nil {
		return nil, err
	}

	infos := make([]*metadatav1.BucketInfo, 0, len(buckets))
	for _, b := range buckets {
		infos = append(infos, &metadatav1.BucketInfo{
			BucketId:  b.ID,
			Name:      b.Name,
			OwnerId:   b.OwnerID,
			CreatedAt: b.CreatedAt.UnixMilli(),
		})
	}

	return &metadatav1.ListBucketsResponse{Buckets: infos}, nil
}
