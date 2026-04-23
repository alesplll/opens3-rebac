package object

import (
	"context"
	"encoding/json"
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
	_ = s.objectDeleted.Send(ctx, []byte(objectID), payload, nil)

	return objectID, blobID, nil
}
