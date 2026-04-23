package tests

import (
	"context"
	"errors"
	"testing"

	bucketService "github.com/alesplll/opens3-rebac/services/metadata/internal/service/bucket"
	"github.com/alesplll/opens3-rebac/services/metadata/pkg/mocks"
	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/require"
)

func TestHeadBucket(t *testing.T) {
	type args struct {
		ctx  context.Context
		name string
	}

	ctx := context.Background()
	repoErr := errors.New("db error")

	tests := []struct {
		name          string
		args          args
		wantExists    bool
		wantBucketID  string
		wantOwnerID   string
		wantErr       error
		buildRepoMock func(mc *minimock.Controller) *mocks.BucketRepositoryMock
	}{
		{
			name:         "bucket exists",
			args:         args{ctx: ctx, name: "my-bucket"},
			wantExists:   true,
			wantBucketID: "bucket-uuid-1",
			wantOwnerID:  "owner-uuid-1",
			buildRepoMock: func(mc *minimock.Controller) *mocks.BucketRepositoryMock {
				m := mocks.NewBucketRepositoryMock(mc)
				m.HeadMock.Inspect(func(ctx context.Context, name string) {
					require.Equal(t, "my-bucket", name)
				}).Return(true, "bucket-uuid-1", "owner-uuid-1", nil)
				return m
			},
		},
		{
			name:         "bucket does not exist",
			args:         args{ctx: ctx, name: "missing-bucket"},
			wantExists:   false,
			wantBucketID: "",
			wantOwnerID:  "",
			buildRepoMock: func(mc *minimock.Controller) *mocks.BucketRepositoryMock {
				m := mocks.NewBucketRepositoryMock(mc)
				m.HeadMock.Return(false, "", "", nil)
				return m
			},
		},
		{
			name:    "error: repository failure",
			args:    args{ctx: ctx, name: "my-bucket"},
			wantErr: repoErr,
			buildRepoMock: func(mc *minimock.Controller) *mocks.BucketRepositoryMock {
				m := mocks.NewBucketRepositoryMock(mc)
				m.HeadMock.Return(false, "", "", repoErr)
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

			svc := bucketService.NewService(repoMock, txMock, producerMock)

			exists, bucketID, ownerID, err := svc.HeadBucket(tt.args.ctx, tt.args.name)

			require.Equal(t, tt.wantErr, err)
			if tt.wantErr == nil {
				require.Equal(t, tt.wantExists, exists)
				require.Equal(t, tt.wantBucketID, bucketID)
				require.Equal(t, tt.wantOwnerID, ownerID)
			}
		})
	}
}
