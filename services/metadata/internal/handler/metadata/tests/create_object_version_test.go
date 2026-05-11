package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	metadatav1 "github.com/alesplll/opens3-rebac/shared/pkg/go/metadata/v1"
	"github.com/stretchr/testify/require"
)

func TestCreateObjectVersion(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		createdAt := time.UnixMilli(1712345678901)

		handler := newHandler(nil, &objectServiceStub{
			createObjectVersionFunc: func(gotCtx context.Context, bucketName, key, blobID string, sizeBytes int64, etag, contentType string) (string, string, time.Time, error) {
				require.Equal(t, ctx, gotCtx)
				require.Equal(t, "photos", bucketName)
				require.Equal(t, "cats/1.jpg", key)
				require.Equal(t, "blob-1", blobID)
				require.EqualValues(t, 123, sizeBytes)
				require.Equal(t, "\"etag-1\"", etag)
				require.Equal(t, "image/jpeg", contentType)

				return "object-1", "version-1", createdAt, nil
			},
		})

		res, err := handler.CreateObjectVersion(ctx, &metadatav1.CreateObjectVersionRequest{
			BucketName:  "photos",
			Key:         "cats/1.jpg",
			BlobId:      "blob-1",
			SizeBytes:   123,
			Etag:        "\"etag-1\"",
			ContentType: "image/jpeg",
		})

		require.NoError(t, err)
		require.Equal(t, &metadatav1.CreateObjectVersionResponse{
			ObjectId:  "object-1",
			VersionId: "version-1",
			CreatedAt: createdAt.UnixMilli(),
		}, res)
	})

	t.Run("service error", func(t *testing.T) {
		ctx := context.Background()
		serviceErr := errors.New("service error")

		handler := newHandler(nil, &objectServiceStub{
			createObjectVersionFunc: func(context.Context, string, string, string, int64, string, string) (string, string, time.Time, error) {
				return "", "", time.Time{}, serviceErr
			},
		})

		res, err := handler.CreateObjectVersion(ctx, &metadatav1.CreateObjectVersionRequest{
			BucketName: "photos",
			Key:        "cats/1.jpg",
		})

		require.Nil(t, res)
		require.ErrorIs(t, err, serviceErr)
	})
}
