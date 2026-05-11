package metadata

import (
	"context"

	metadatav1 "github.com/alesplll/opens3-rebac/shared/pkg/go/metadata/v1"
)

func (h *handler) ListObjects(ctx context.Context, req *metadatav1.ListObjectsRequest) (*metadatav1.ListObjectsResponse, error) {
	items, nextToken, isTruncated, err := h.objectService.ListObjects(
		ctx,
		req.GetBucketName(),
		req.GetPrefix(),
		req.GetContinuationToken(),
		req.GetMaxKeys(),
	)
	if err != nil {
		return nil, err
	}

	objects := make([]*metadatav1.ObjectInfo, 0, len(items))
	for _, item := range items {
		objects = append(objects, &metadatav1.ObjectInfo{
			ObjectId:     item.ObjectID,
			VersionId:    item.VersionID,
			Key:          item.Key,
			Etag:         item.Etag,
			SizeBytes:    item.SizeBytes,
			ContentType:  item.ContentType,
			LastModified: item.LastModified.UnixMilli(),
		})
	}

	return &metadatav1.ListObjectsResponse{
		Objects:               objects,
		NextContinuationToken: nextToken,
		IsTruncated:           isTruncated,
	}, nil
}
