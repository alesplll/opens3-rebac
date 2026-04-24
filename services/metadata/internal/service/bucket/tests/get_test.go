package tests

import (
	"context"
	"testing"
	"time"

	domainerrors "github.com/alesplll/opens3-rebac/services/metadata/internal/errors/domain_errors"
	"github.com/alesplll/opens3-rebac/services/metadata/internal/model"
	bucketService "github.com/alesplll/opens3-rebac/services/metadata/internal/service/bucket"
	"github.com/alesplll/opens3-rebac/services/metadata/pkg/mocks"
	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/require"
)

func TestGetBucket(t *testing.T) {
	type args struct {
		ctx  context.Context
		name string
	}

	ctx := context.Background()

	wantBucket := &model.Bucket{
		ID:        "bucket-uuid-1",
		Name:      "my-bucket",
		OwnerID:   "owner-uuid-1",
		CreatedAt: time.Now(),
	}

	tests := []struct {
		name          string
		args          args
		wantBucket    *model.Bucket
		wantErr       error
		buildRepoMock func(mc *minimock.Controller) *mocks.BucketRepositoryMock
	}{
		{
			name:       "success",
			args:       args{ctx: ctx, name: "my-bucket"},
			wantBucket: wantBucket,
			buildRepoMock: func(mc *minimock.Controller) *mocks.BucketRepositoryMock {
				m := mocks.NewBucketRepositoryMock(mc)
				m.GetMock.Inspect(func(ctx context.Context, name string) {
					require.Equal(t, "my-bucket", name)
				}).Return(wantBucket, nil)
				return m
			},
		},
		{
			name:    "error: bucket not found",
			args:    args{ctx: ctx, name: "missing-bucket"},
			wantErr: domainerrors.ErrBucketNotFound,
			buildRepoMock: func(mc *minimock.Controller) *mocks.BucketRepositoryMock {
				m := mocks.NewBucketRepositoryMock(mc)
				m.GetMock.Return(nil, domainerrors.ErrBucketNotFound)
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

			got, err := svc.GetBucket(tt.args.ctx, tt.args.name)

			require.Equal(t, tt.wantErr, err)
			if tt.wantErr == nil {
				require.Equal(t, tt.wantBucket, got)
			}
		})
	}
}
