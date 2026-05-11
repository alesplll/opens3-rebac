package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/alesplll/opens3-rebac/services/metadata/pkg/mocks"
	metadatav1 "github.com/alesplll/opens3-rebac/shared/pkg/go/metadata/v1"
	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/require"
)

func TestDeleteObjectMeta(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		mc := minimock.NewController(t)

		objectServiceMock := mocks.NewObjectServiceMock(mc)
		objectServiceMock.DeleteObjectMetaMock.Expect(ctx, "photos", "cats/1.jpg").Return("object-1", "blob-1", nil)

		handler := newHandler(nil, objectServiceMock)

		res, err := handler.DeleteObjectMeta(ctx, &metadatav1.DeleteObjectMetaRequest{
			BucketName: "photos",
			Key:        "cats/1.jpg",
		})

		require.NoError(t, err)
		require.Equal(t, &metadatav1.DeleteObjectMetaResponse{
			ObjectId: "object-1",
			BlobId:   "blob-1",
			Success:  true,
		}, res)
	})

	t.Run("service error", func(t *testing.T) {
		ctx := context.Background()
		serviceErr := errors.New("service error")
		mc := minimock.NewController(t)

		objectServiceMock := mocks.NewObjectServiceMock(mc)
		objectServiceMock.DeleteObjectMetaMock.Expect(ctx, "photos", "cats/1.jpg").Return("", "", serviceErr)

		handler := newHandler(nil, objectServiceMock)

		res, err := handler.DeleteObjectMeta(ctx, &metadatav1.DeleteObjectMetaRequest{
			BucketName: "photos",
			Key:        "cats/1.jpg",
		})

		require.Nil(t, res)
		require.ErrorIs(t, err, serviceErr)
	})
}
