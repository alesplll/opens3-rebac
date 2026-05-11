package object

import (
	"context"
	"encoding/json"

	domainerrors "github.com/alesplll/opens3-rebac/services/metadata/internal/errors/domain_errors"
)

type objectDeletedEvent struct {
	ObjectID string `json:"object_id"`
	BlobID   string `json:"blob_id"`
}

func (s *objectService) DeleteObjectMeta(ctx context.Context, bucketName, key string) (string, string, error) {
	objectID, blobID, err := s.repo.Delete(ctx, bucketName, key)
	if err != nil {
		return "", "", err
	}

	payload, _ := json.Marshal(objectDeletedEvent{ObjectID: objectID, BlobID: blobID})
	if err := s.objectDeleted.Send(ctx, []byte(objectID), payload, nil); err != nil {
		return "", "", domainerrors.ErrInternal
	}

	return objectID, blobID, nil
}
