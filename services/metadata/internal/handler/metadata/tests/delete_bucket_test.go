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

func TestDeleteBucket(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		mc := minimock.NewController(t)

		bucketServiceMock := mocks.NewBucketServiceMock(mc)
		bucketServiceMock.DeleteBucketMock.Expect(ctx, "photos").Return(nil)

		handler := newHandler(bucketServiceMock, nil)

		res, err := handler.DeleteBucket(ctx, &metadatav1.DeleteBucketRequest{
			BucketName: "photos",
		})

		require.NoError(t, err)
		require.Equal(t, &metadatav1.DeleteBucketResponse{Success: true}, res)
	})

	t.Run("service error", func(t *testing.T) {
		ctx := context.Background()
		serviceErr := errors.New("service error")
		mc := minimock.NewController(t)

		bucketServiceMock := mocks.NewBucketServiceMock(mc)
		bucketServiceMock.DeleteBucketMock.Expect(ctx, "photos").Return(serviceErr)

		handler := newHandler(bucketServiceMock, nil)

		res, err := handler.DeleteBucket(ctx, &metadatav1.DeleteBucketRequest{
			BucketName: "photos",
		})

		require.Nil(t, res)
		require.ErrorIs(t, err, serviceErr)
	})
}
