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

func TestListBuckets(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		firstCreatedAt := time.UnixMilli(1712345678901)
		secondCreatedAt := time.UnixMilli(1712345679901)
		mc := minimock.NewController(t)

		bucketServiceMock := mocks.NewBucketServiceMock(mc)
		bucketServiceMock.ListBucketsMock.
			Expect(ctx, "owner-1").
			Return([]*model.Bucket{
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
			}, nil)

		handler := newHandler(bucketServiceMock, nil)

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
		mc := minimock.NewController(t)

		bucketServiceMock := mocks.NewBucketServiceMock(mc)
		bucketServiceMock.ListBucketsMock.Expect(ctx, "owner-1").Return(nil, serviceErr)

		handler := newHandler(bucketServiceMock, nil)

		res, err := handler.ListBuckets(ctx, &metadatav1.ListBucketsRequest{
			OwnerId: "owner-1",
		})

		require.Nil(t, res)
		require.ErrorIs(t, err, serviceErr)
	})
}
