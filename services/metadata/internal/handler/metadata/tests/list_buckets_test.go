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

func TestListBuckets(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		firstCreatedAt := time.UnixMilli(1712345678901)
		secondCreatedAt := time.UnixMilli(1712345679901)

		handler := newHandler(&bucketServiceStub{
			listBucketsFunc: func(gotCtx context.Context, ownerID string) ([]*model.Bucket, error) {
				require.Equal(t, ctx, gotCtx)
				require.Equal(t, "owner-1", ownerID)

				return []*model.Bucket{
					{
						ID:        "bucket-1",
						Name:      "photos",
						OwnerID:   "owner-1",
						CreatedAt: firstCreatedAt,
					},
					{
						ID:        "bucket-2",
						Name:      "videos",
						OwnerID:   "owner-1",
						CreatedAt: secondCreatedAt,
					},
				}, nil
			},
		}, nil)

		res, err := handler.ListBuckets(ctx, &metadatav1.ListBucketsRequest{
			OwnerId: "owner-1",
		})

		require.NoError(t, err)
		require.Equal(t, &metadatav1.ListBucketsResponse{
			Buckets: []*metadatav1.BucketInfo{
				{
					BucketId:  "bucket-1",
					Name:      "photos",
					OwnerId:   "owner-1",
					CreatedAt: firstCreatedAt.UnixMilli(),
				},
				{
					BucketId:  "bucket-2",
					Name:      "videos",
					OwnerId:   "owner-1",
					CreatedAt: secondCreatedAt.UnixMilli(),
				},
			},
		}, res)
	})

	t.Run("service error", func(t *testing.T) {
		ctx := context.Background()
		serviceErr := errors.New("service error")

		handler := newHandler(&bucketServiceStub{
			listBucketsFunc: func(context.Context, string) ([]*model.Bucket, error) {
				return nil, serviceErr
			},
		}, nil)

		res, err := handler.ListBuckets(ctx, &metadatav1.ListBucketsRequest{
			OwnerId: "owner-1",
		})

		require.Nil(t, res)
		require.ErrorIs(t, err, serviceErr)
	})
}
