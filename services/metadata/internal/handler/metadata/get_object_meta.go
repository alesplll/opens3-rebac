package metadata

import (
	"context"

	metadatav1 "github.com/alesplll/opens3-rebac/shared/pkg/go/metadata/v1"
)

func (h *handler) GetObjectMeta(ctx context.Context, req *metadatav1.GetObjectMetaRequest) (*metadatav1.GetObjectMetaResponse, error) {
	meta, err := h.objectService.GetObjectMeta(ctx, req.GetBucketName(), req.GetKey(), req.GetVersionId())
	if err != nil {
		return nil, err
	}

	return &metadatav1.GetObjectMetaResponse{
		ObjectId:     meta.ObjectID,
		VersionId:    meta.VersionID,
		BlobId:       meta.BlobID,
		SizeBytes:    meta.SizeBytes,
		Etag:         meta.Etag,
		ContentType:  meta.ContentType,
		LastModified: meta.LastModified.UnixMilli(),
	}, nil
}
