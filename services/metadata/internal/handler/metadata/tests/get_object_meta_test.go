package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alesplll/opens3-rebac/services/metadata/internal/model"
	metadatav1 "github.com/alesplll/opens3-rebac/shared/pkg/go/metadata/v1"
	"github.com/stretchr/testify/require"
)

func TestGetObjectMeta(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		lastModified := time.UnixMilli(1712345678901)

		handler := newHandler(nil, &objectServiceStub{
			getObjectMetaFunc: func(gotCtx context.Context, bucketName, key, versionID string) (*model.ObjectMeta, error) {
				require.Equal(t, ctx, gotCtx)
				require.Equal(t, "photos", bucketName)
				require.Equal(t, "cats/1.jpg", key)
				require.Equal(t, "version-1", versionID)

				return &model.ObjectMeta{
					ObjectID:     "object-1",
					VersionID:    "version-1",
					BlobID:       "blob-1",
					SizeBytes:    123,
					Etag:         "\"etag-1\"",
					ContentType:  "image/jpeg",
					LastModified: lastModified,
				}, nil
			},
		})

		res, err := handler.GetObjectMeta(ctx, &metadatav1.GetObjectMetaRequest{
			BucketName: "photos",
			Key:        "cats/1.jpg",
			VersionId:  "version-1",
		})

		require.NoError(t, err)
		require.Equal(t, &metadatav1.GetObjectMetaResponse{
			ObjectId:     "object-1",
			VersionId:    "version-1",
			BlobId:       "blob-1",
			SizeBytes:    123,
			Etag:         "\"etag-1\"",
			ContentType:  "image/jpeg",
			LastModified: lastModified.UnixMilli(),
		}, res)
	})

	t.Run("service error", func(t *testing.T) {
		ctx := context.Background()
		serviceErr := errors.New("service error")

		handler := newHandler(nil, &objectServiceStub{
			getObjectMetaFunc: func(context.Context, string, string, string) (*model.ObjectMeta, error) {
				return nil, serviceErr
			},
		})

		res, err := handler.GetObjectMeta(ctx, &metadatav1.GetObjectMetaRequest{
			BucketName: "photos",
			Key:        "cats/1.jpg",
		})

		require.Nil(t, res)
		require.ErrorIs(t, err, serviceErr)
	})
}
