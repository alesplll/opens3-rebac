package tests

import (
	"context"
	"errors"
	"testing"

	metadatav1 "github.com/alesplll/opens3-rebac/shared/pkg/go/metadata/v1"
	"github.com/stretchr/testify/require"
)

func TestDeleteObjectMeta(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx := context.Background()

		handler := newHandler(nil, &objectServiceStub{
			deleteObjectMetaFunc: func(gotCtx context.Context, bucketName, key string) (string, string, error) {
				require.Equal(t, ctx, gotCtx)
				require.Equal(t, "photos", bucketName)
				require.Equal(t, "cats/1.jpg", key)
				return "object-1", "blob-1", nil
			},
		})

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

		handler := newHandler(nil, &objectServiceStub{
			deleteObjectMetaFunc: func(context.Context, string, string) (string, string, error) {
				return "", "", serviceErr
			},
		})

		res, err := handler.DeleteObjectMeta(ctx, &metadatav1.DeleteObjectMetaRequest{
			BucketName: "photos",
			Key:        "cats/1.jpg",
		})

		require.Nil(t, res)
		require.ErrorIs(t, err, serviceErr)
	})
}
