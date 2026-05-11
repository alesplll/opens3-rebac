package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alesplll/opens3-rebac/services/metadata/pkg/mocks"
	metadatav1 "github.com/alesplll/opens3-rebac/shared/pkg/go/metadata/v1"
	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/require"
)

func TestCreateObjectVersion(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		createdAt := time.UnixMilli(1712345678901)
		mc := minimock.NewController(t)

		objectServiceMock := mocks.NewObjectServiceMock(mc)
		objectServiceMock.CreateObjectVersionMock.
			Expect(ctx, "photos", "cats/1.jpg", "blob-1", int64(123), "\"etag-1\"", "image/jpeg").
			Return("object-1", "version-1", createdAt, nil)

		handler := newHandler(nil, objectServiceMock)

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
		mc := minimock.NewController(t)

		objectServiceMock := mocks.NewObjectServiceMock(mc)
		objectServiceMock.CreateObjectVersionMock.
			Expect(ctx, "photos", "cats/1.jpg", "", int64(0), "", "").
			Return("", "", time.Time{}, serviceErr)

		handler := newHandler(nil, objectServiceMock)

		res, err := handler.CreateObjectVersion(ctx, &metadatav1.CreateObjectVersionRequest{
			BucketName: "photos",
			Key:        "cats/1.jpg",
		})

		require.Nil(t, res)
		require.ErrorIs(t, err, serviceErr)
	})
}
