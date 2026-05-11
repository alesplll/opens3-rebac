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

func TestHeadBucket(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		mc := minimock.NewController(t)

		bucketServiceMock := mocks.NewBucketServiceMock(mc)
		bucketServiceMock.HeadBucketMock.Expect(ctx, "photos").Return(true, "bucket-1", "owner-1", nil)

		handler := newHandler(bucketServiceMock, nil)

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
		mc := minimock.NewController(t)

		bucketServiceMock := mocks.NewBucketServiceMock(mc)
		bucketServiceMock.HeadBucketMock.Expect(ctx, "photos").Return(false, "", "", serviceErr)

		handler := newHandler(bucketServiceMock, nil)

		res, err := handler.HeadBucket(ctx, &metadatav1.HeadBucketRequest{
			BucketName: "photos",
		})

		require.Nil(t, res)
		require.ErrorIs(t, err, serviceErr)
	})
}
