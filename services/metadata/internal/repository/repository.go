package repository

import (
	"context"
	"time"

	"github.com/alesplll/opens3-rebac/services/metadata/internal/model"
)

type BucketRepository interface {
	Create(ctx context.Context, name, ownerID string) (*model.Bucket, error)
	Get(ctx context.Context, name string) (*model.Bucket, error)
	List(ctx context.Context, ownerID string) ([]*model.Bucket, error)
	Head(ctx context.Context, name string) (bool, string, string, error)
	Delete(ctx context.Context, bucketID string) error
	CountObjects(ctx context.Context, bucketID string) (int64, error)
}

type ObjectRepository interface {
	UpsertObject(ctx context.Context, bucketName, key string) (string, error)
	InsertVersion(ctx context.Context, objectID, blobID string, sizeBytes int64, etag, contentType string) (string, time.Time, error)
	SetCurrentVersion(ctx context.Context, objectID, versionID string) error
	GetMeta(ctx context.Context, bucketName, key, versionID string) (*model.ObjectMeta, error)
	Delete(ctx context.Context, bucketName, key string) (string, string, error)
	List(ctx context.Context, bucketName, prefix, continuationToken string, maxKeys int32) ([]*model.ObjectListItem, string, bool, error)
}
