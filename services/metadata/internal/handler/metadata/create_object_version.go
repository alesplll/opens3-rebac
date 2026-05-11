package metadata

import (
	"context"

	metadatav1 "github.com/alesplll/opens3-rebac/shared/pkg/go/metadata/v1"
)

func (h *handler) CreateObjectVersion(ctx context.Context, req *metadatav1.CreateObjectVersionRequest) (*metadatav1.CreateObjectVersionResponse, error) {
	objectID, versionID, createdAt, err := h.objectService.CreateObjectVersion(
		ctx,
		req.GetBucketName(),
		req.GetKey(),
		req.GetBlobId(),
		req.GetSizeBytes(),
		req.GetEtag(),
		req.GetContentType(),
	)
	if err != nil {
		return nil, err
	}

	return &metadatav1.CreateObjectVersionResponse{
		ObjectId:  objectID,
		VersionId: versionID,
		CreatedAt: createdAt.UnixMilli(),
	}, nil
}
