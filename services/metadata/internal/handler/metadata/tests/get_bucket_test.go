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

func TestGetBucket(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		createdAt := time.UnixMilli(1712345678901)

		handler := newHandler(&bucketServiceStub{
			getBucketFunc: func(gotCtx context.Context, name string) (*model.Bucket, error) {
				require.Equal(t, ctx, gotCtx)
				require.Equal(t, "photos", name)

				return &model.Bucket{
					ID:        "bucket-1",
					Name:      "photos",
					OwnerID:   "owner-1",
					CreatedAt: createdAt,
				}, nil
			},
		}, nil)

		res, err := handler.GetBucket(ctx, &metadatav1.GetBucketRequest{
			BucketName: "photos",
		})

		require.NoError(t, err)
		require.Equal(t, &metadatav1.GetBucketResponse{
			Bucket: &metadatav1.BucketInfo{
				BucketId:  "bucket-1",
				Name:      "photos",
				OwnerId:   "owner-1",
				CreatedAt: createdAt.UnixMilli(),
			},
		}, res)
	})

	t.Run("service error", func(t *testing.T) {
		ctx := context.Background()
		serviceErr := errors.New("service error")

		handler := newHandler(&bucketServiceStub{
			getBucketFunc: func(context.Context, string) (*model.Bucket, error) {
				return nil, serviceErr
			},
		}, nil)

		res, err := handler.GetBucket(ctx, &metadatav1.GetBucketRequest{
			BucketName: "photos",
		})

		require.Nil(t, res)
		require.ErrorIs(t, err, serviceErr)
	})
}
