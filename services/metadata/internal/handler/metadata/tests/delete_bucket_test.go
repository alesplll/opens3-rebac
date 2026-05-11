package tests

import (
	"context"
	"errors"
	"testing"

	metadatav1 "github.com/alesplll/opens3-rebac/shared/pkg/go/metadata/v1"
	"github.com/stretchr/testify/require"
)

func TestDeleteBucket(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx := context.Background()

		handler := newHandler(&bucketServiceStub{
			deleteBucketFunc: func(gotCtx context.Context, name string) error {
				require.Equal(t, ctx, gotCtx)
				require.Equal(t, "photos", name)
				return nil
			},
		}, nil)

		res, err := handler.DeleteBucket(ctx, &metadatav1.DeleteBucketRequest{
			BucketName: "photos",
		})

		require.NoError(t, err)
		require.Equal(t, &metadatav1.DeleteBucketResponse{Success: true}, res)
	})

	t.Run("service error", func(t *testing.T) {
		ctx := context.Background()
		serviceErr := errors.New("service error")

		handler := newHandler(&bucketServiceStub{
			deleteBucketFunc: func(context.Context, string) error {
				return serviceErr
			},
		}, nil)

		res, err := handler.DeleteBucket(ctx, &metadatav1.DeleteBucketRequest{
			BucketName: "photos",
		})

		require.Nil(t, res)
		require.ErrorIs(t, err, serviceErr)
	})
}
