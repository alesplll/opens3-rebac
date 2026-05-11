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

func TestListObjects(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		firstModified := time.UnixMilli(1712345678901)
		secondModified := time.UnixMilli(1712345679901)

		handler := newHandler(nil, &objectServiceStub{
			listObjectsFunc: func(gotCtx context.Context, bucketName, prefix, continuationToken string, maxKeys int32) ([]*model.ObjectListItem, string, bool, error) {
				require.Equal(t, ctx, gotCtx)
				require.Equal(t, "photos", bucketName)
				require.Equal(t, "cats/", prefix)
				require.Equal(t, "token-1", continuationToken)
				require.EqualValues(t, 100, maxKeys)

				return []*model.ObjectListItem{
					{
						ObjectID:     "object-1",
						VersionID:    "version-1",
						Key:          "cats/1.jpg",
						Etag:         "\"etag-1\"",
						SizeBytes:    123,
						ContentType:  "image/jpeg",
						LastModified: firstModified,
					},
					{
						ObjectID:     "object-2",
						VersionID:    "version-2",
						Key:          "cats/2.jpg",
						Etag:         "\"etag-2\"",
						SizeBytes:    456,
						ContentType:  "image/jpeg",
						LastModified: secondModified,
					},
				}, "token-2", true, nil
			},
		})

		res, err := handler.ListObjects(ctx, &metadatav1.ListObjectsRequest{
			BucketName:        "photos",
			Prefix:            "cats/",
			ContinuationToken: "token-1",
			MaxKeys:           100,
		})

		require.NoError(t, err)
		require.Equal(t, &metadatav1.ListObjectsResponse{
			Objects: []*metadatav1.ObjectInfo{
				{
					ObjectId:     "object-1",
					VersionId:    "version-1",
					Key:          "cats/1.jpg",
					Etag:         "\"etag-1\"",
					SizeBytes:    123,
					ContentType:  "image/jpeg",
					LastModified: firstModified.UnixMilli(),
				},
				{
					ObjectId:     "object-2",
					VersionId:    "version-2",
					Key:          "cats/2.jpg",
					Etag:         "\"etag-2\"",
					SizeBytes:    456,
					ContentType:  "image/jpeg",
					LastModified: secondModified.UnixMilli(),
				},
			},
			NextContinuationToken: "token-2",
			IsTruncated:           true,
		}, res)
	})

	t.Run("service error", func(t *testing.T) {
		ctx := context.Background()
		serviceErr := errors.New("service error")

		handler := newHandler(nil, &objectServiceStub{
			listObjectsFunc: func(context.Context, string, string, string, int32) ([]*model.ObjectListItem, string, bool, error) {
				return nil, "", false, serviceErr
			},
		})

		res, err := handler.ListObjects(ctx, &metadatav1.ListObjectsRequest{
			BucketName: "photos",
		})

		require.Nil(t, res)
		require.ErrorIs(t, err, serviceErr)
	})
}
