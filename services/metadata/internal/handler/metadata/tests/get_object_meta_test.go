package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alesplll/opens3-rebac/services/metadata/internal/model"
	"github.com/alesplll/opens3-rebac/services/metadata/pkg/mocks"
	metadatav1 "github.com/alesplll/opens3-rebac/shared/pkg/go/metadata/v1"
	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/require"
)

func TestGetObjectMeta(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		lastModified := time.UnixMilli(1712345678901)
		mc := minimock.NewController(t)

		objectServiceMock := mocks.NewObjectServiceMock(mc)
		objectServiceMock.GetObjectMetaMock.
			Expect(ctx, "photos", "cats/1.jpg", "version-1").
			Return(&model.ObjectMeta{
				ObjectID:     "object-1",
				VersionID:    "version-1",
				BlobID:       "blob-1",
				SizeBytes:    123,
				Etag:         "\"etag-1\"",
				ContentType:  "image/jpeg",
				LastModified: lastModified,
			}, nil)

		handler := newHandler(nil, objectServiceMock)

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
		mc := minimock.NewController(t)

		objectServiceMock := mocks.NewObjectServiceMock(mc)
		objectServiceMock.GetObjectMetaMock.Expect(ctx, "photos", "cats/1.jpg", "").Return(nil, serviceErr)

		handler := newHandler(nil, objectServiceMock)

		res, err := handler.GetObjectMeta(ctx, &metadatav1.GetObjectMetaRequest{
			BucketName: "photos",
			Key:        "cats/1.jpg",
		})

		require.Nil(t, res)
		require.ErrorIs(t, err, serviceErr)
	})
}
