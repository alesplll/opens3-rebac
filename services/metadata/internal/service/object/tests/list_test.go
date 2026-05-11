package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alesplll/opens3-rebac/services/metadata/internal/model"
	objectService "github.com/alesplll/opens3-rebac/services/metadata/internal/service/object"
	"github.com/alesplll/opens3-rebac/services/metadata/pkg/mocks"
	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/require"
)

func TestListObjects(t *testing.T) {
	type args struct {
		ctx               context.Context
		bucketName        string
		prefix            string
		continuationToken string
		maxKeys           int32
	}

	ctx := context.Background()
	now := time.Now()

	items := []*model.ObjectListItem{
		{
			ObjectID:     "object-uuid-1",
			VersionID:    "version-uuid-1",
			Key:          "photos/cat.jpg",
			Etag:         `"abc"`,
			SizeBytes:    1024,
			ContentType:  "image/jpeg",
			LastModified: now,
		},
		{
			ObjectID:     "object-uuid-2",
			VersionID:    "version-uuid-2",
			Key:          "photos/dog.jpg",
			Etag:         `"def"`,
			SizeBytes:    2048,
			ContentType:  "image/jpeg",
			LastModified: now,
		},
	}

	repoErr := errors.New("db error")

	tests := []struct {
		name              string
		args              args
		wantItems         []*model.ObjectListItem
		wantNextToken     string
		wantIsTruncated   bool
		wantErr           error
		buildRepoMock     func(mc *minimock.Controller) *mocks.ObjectRepositoryMock
	}{
		{
			name: "success: returns objects",
			args: args{
				ctx:               ctx,
				bucketName:        "my-bucket",
				prefix:            "",
				continuationToken: "",
				maxKeys:           100,
			},
			wantItems:       items,
			wantNextToken:   "",
			wantIsTruncated: false,
			buildRepoMock: func(mc *minimock.Controller) *mocks.ObjectRepositoryMock {
				m := mocks.NewObjectRepositoryMock(mc)
				m.ListMock.Inspect(func(ctx context.Context, bucketName, prefix, continuationToken string, maxKeys int32) {
					require.Equal(t, "my-bucket", bucketName)
					require.Equal(t, "", prefix)
					require.Equal(t, int32(100), maxKeys)
				}).Return(items, "", false, nil)
				return m
			},
		},
		{
			name: "success: filtered by prefix",
			args: args{
				ctx:        ctx,
				bucketName: "my-bucket",
				prefix:     "photos/",
				maxKeys:    10,
			},
			wantItems:       items,
			wantNextToken:   "",
			wantIsTruncated: false,
			buildRepoMock: func(mc *minimock.Controller) *mocks.ObjectRepositoryMock {
				m := mocks.NewObjectRepositoryMock(mc)
				m.ListMock.Inspect(func(ctx context.Context, bucketName, prefix, continuationToken string, maxKeys int32) {
					require.Equal(t, "photos/", prefix)
				}).Return(items, "", false, nil)
				return m
			},
		},
		{
			name: "success: truncated list with next token",
			args: args{
				ctx:        ctx,
				bucketName: "my-bucket",
				prefix:     "",
				maxKeys:    1,
			},
			wantItems:       items[:1],
			wantNextToken:   "next-token-abc",
			wantIsTruncated: true,
			buildRepoMock: func(mc *minimock.Controller) *mocks.ObjectRepositoryMock {
				m := mocks.NewObjectRepositoryMock(mc)
				m.ListMock.Return(items[:1], "next-token-abc", true, nil)
				return m
			},
		},
		{
			name: "success: empty bucket",
			args: args{
				ctx:        ctx,
				bucketName: "empty-bucket",
				maxKeys:    100,
			},
			wantItems:       []*model.ObjectListItem{},
			wantNextToken:   "",
			wantIsTruncated: false,
			buildRepoMock: func(mc *minimock.Controller) *mocks.ObjectRepositoryMock {
				m := mocks.NewObjectRepositoryMock(mc)
				m.ListMock.Return([]*model.ObjectListItem{}, "", false, nil)
				return m
			},
		},
		{
			name:    "error: repository failure",
			args:    args{ctx: ctx, bucketName: "my-bucket", maxKeys: 100},
			wantErr: repoErr,
			buildRepoMock: func(mc *minimock.Controller) *mocks.ObjectRepositoryMock {
				m := mocks.NewObjectRepositoryMock(mc)
				m.ListMock.Return(nil, "", false, repoErr)
				return m
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := minimock.NewController(t)

			repoMock := tt.buildRepoMock(mc)
			txMock := mocks.NewTxManagerMock(mc)
			producerMock := mocks.NewProducerMock(mc)

			svc := objectService.NewService(repoMock, txMock, producerMock, nil, nil)

			gotItems, gotNextToken, gotIsTruncated, err := svc.ListObjects(
				tt.args.ctx,
				tt.args.bucketName,
				tt.args.prefix,
				tt.args.continuationToken,
				tt.args.maxKeys,
			)

			require.Equal(t, tt.wantErr, err)
			if tt.wantErr == nil {
				require.Equal(t, tt.wantItems, gotItems)
				require.Equal(t, tt.wantNextToken, gotNextToken)
				require.Equal(t, tt.wantIsTruncated, gotIsTruncated)
			}
		})
	}
}
