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

func TestCreateBucket(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		createdAt := time.UnixMilli(1712345678901)

		handler := newHandler(&bucketServiceStub{
			createBucketFunc: func(gotCtx context.Context, name, ownerID string) (*model.Bucket, error) {
				require.Equal(t, ctx, gotCtx)
				require.Equal(t, "photos", name)
				require.Equal(t, "owner-1", ownerID)

				return &model.Bucket{
					ID:        "bucket-1",
					CreatedAt: createdAt,
				}, nil
			},
		}, nil)

		res, err := handler.CreateBucket(ctx, &metadatav1.CreateBucketRequest{
			Name:    "photos",
			OwnerId: "owner-1",
		})

		require.NoError(t, err)
		require.Equal(t, &metadatav1.CreateBucketResponse{
			BucketId:  "bucket-1",
			CreatedAt: createdAt.UnixMilli(),
		}, res)
	})

	t.Run("service error", func(t *testing.T) {
		ctx := context.Background()
		serviceErr := errors.New("service error")

		handler := newHandler(&bucketServiceStub{
			createBucketFunc: func(context.Context, string, string) (*model.Bucket, error) {
				return nil, serviceErr
			},
		}, nil)

		res, err := handler.CreateBucket(ctx, &metadatav1.CreateBucketRequest{
			Name:    "photos",
			OwnerId: "owner-1",
		})

		require.Nil(t, res)
		require.ErrorIs(t, err, serviceErr)
	})
}
