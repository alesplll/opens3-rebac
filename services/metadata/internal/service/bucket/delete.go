package bucket

import (
	"context"
	"encoding/json"

	domainerrors "github.com/alesplll/opens3-rebac/services/metadata/internal/errors/domain_errors"
)

type bucketDeletedEvent struct {
	BucketID   string `json:"bucket_id"`
	BucketName string `json:"bucket_name"`
}

func (s *bucketService) DeleteBucket(ctx context.Context, name string) error {
	var bucketID string

	txErr := s.txManager.ReadCommitted(ctx, func(ctx context.Context) error {
		bucket, err := s.repo.Get(ctx, name)
		if err != nil {
			return err
		}

		count, err := s.repo.CountObjects(ctx, bucket.ID)
		if err != nil {
			return err
		}

		if count > 0 {
			return domainerrors.ErrBucketNotEmpty
		}

		bucketID = bucket.ID
		return s.repo.Delete(ctx, bucket.ID)
	})

	if txErr != nil {
		return txErr
	}

	payload, _ := json.Marshal(bucketDeletedEvent{BucketID: bucketID, BucketName: name})
	_ = s.bucketDeleted.Send(ctx, []byte(bucketID), payload, nil)

	return nil
}
