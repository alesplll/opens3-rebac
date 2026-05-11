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

func TestCreateBucket(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		createdAt := time.UnixMilli(1712345678901)
		mc := minimock.NewController(t)

		bucketServiceMock := mocks.NewBucketServiceMock(mc)
		bucketServiceMock.CreateBucketMock.
			Expect(ctx, "photos", "owner-1").
			Return(&model.Bucket{
				ID:        "bucket-1",
				CreatedAt: createdAt,
			}, nil)

		handler := newHandler(bucketServiceMock, nil)

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
		mc := minimock.NewController(t)

		bucketServiceMock := mocks.NewBucketServiceMock(mc)
		bucketServiceMock.CreateBucketMock.Expect(ctx, "photos", "owner-1").Return(nil, serviceErr)

		handler := newHandler(bucketServiceMock, nil)

		res, err := handler.CreateBucket(ctx, &metadatav1.CreateBucketRequest{
			Name:    "photos",
			OwnerId: "owner-1",
		})

		require.Nil(t, res)
		require.ErrorIs(t, err, serviceErr)
	})
}
