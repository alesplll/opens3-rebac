package metadata

import (
	"context"

	metadatav1 "github.com/alesplll/opens3-rebac/shared/pkg/go/metadata/v1"
)

func (h *handler) DeleteObjectMeta(ctx context.Context, req *metadatav1.DeleteObjectMetaRequest) (*metadatav1.DeleteObjectMetaResponse, error) {
	objectID, blobID, err := h.objectService.DeleteObjectMeta(ctx, req.GetBucketName(), req.GetKey())
	if err != nil {
		return nil, err
	}

	return &metadatav1.DeleteObjectMetaResponse{
		ObjectId: objectID,
		BlobId:   blobID,
		Success:  true,
	}, nil
}
