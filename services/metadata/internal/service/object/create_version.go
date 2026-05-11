package object

import (
	"context"
	"time"

	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/sys/validate"
)

func (s *objectService) CreateObjectVersion(ctx context.Context, bucketName, key, blobID string, sizeBytes int64, etag, contentType string) (string, string, time.Time, error) {
	if err := validate.Validate(ctx,
		func(ctx context.Context) error {
			if bucketName == "" {
				return validate.NewValidationErrors("bucket_name is required")
			}
			return nil
		},
		func(ctx context.Context) error {
			if key == "" {
				return validate.NewValidationErrors("key is required")
			}
			return nil
		},
		func(ctx context.Context) error {
			if blobID == "" {
				return validate.NewValidationErrors("blob_id is required")
			}
			return nil
		},
	); err != nil {
		return "", "", time.Time{}, err
	}

	var objectID, versionID string
	var createdAt time.Time

	txErr := s.txManager.ReadCommitted(ctx, func(ctx context.Context) error {
		var err error
		objectID, err = s.repo.UpsertObject(ctx, bucketName, key)
		if err != nil {
			return err
		}

		versionID, createdAt, err = s.repo.InsertVersion(ctx, objectID, blobID, sizeBytes, etag, contentType)
		if err != nil {
			return err
		}

		return s.repo.SetCurrentVersion(ctx, objectID, versionID)
	})

	if txErr != nil {
		return "", "", time.Time{}, txErr
	}

	return objectID, versionID, createdAt, nil
}
