package tests

import (
	"context"
	"errors"
	"testing"

	metadatav1 "github.com/alesplll/opens3-rebac/shared/pkg/go/metadata/v1"
	"github.com/stretchr/testify/require"
)

func TestHeadBucket(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx := context.Background()

		handler := newHandler(&bucketServiceStub{
			headBucketFunc: func(gotCtx context.Context, name string) (bool, string, string, error) {
				require.Equal(t, ctx, gotCtx)
				require.Equal(t, "photos", name)
				return true, "bucket-1", "owner-1", nil
			},
		}, nil)

		res, err := handler.HeadBucket(ctx, &metadatav1.HeadBucketRequest{
			BucketName: "photos",
		})

		require.NoError(t, err)
		require.Equal(t, &metadatav1.HeadBucketResponse{
			Exists:   true,
			BucketId: "bucket-1",
			OwnerId:  "owner-1",
		}, res)
	})

	t.Run("service error", func(t *testing.T) {
		ctx := context.Background()
		serviceErr := errors.New("service error")

		handler := newHandler(&bucketServiceStub{
			headBucketFunc: func(context.Context, string) (bool, string, string, error) {
				return false, "", "", serviceErr
			},
		}, nil)

		res, err := handler.HeadBucket(ctx, &metadatav1.HeadBucketRequest{
			BucketName: "photos",
		})

		require.Nil(t, res)
		require.ErrorIs(t, err, serviceErr)
	})
}
