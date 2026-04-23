package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alesplll/opens3-rebac/services/metadata/internal/model"
	bucketService "github.com/alesplll/opens3-rebac/services/metadata/internal/service/bucket"
	"github.com/alesplll/opens3-rebac/services/metadata/pkg/mocks"
	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/require"
)

func TestListBuckets(t *testing.T) {
	type args struct {
		ctx     context.Context
		ownerID string
	}

	ctx := context.Background()

	now := time.Now()
	buckets := []*model.Bucket{
		{ID: "bucket-uuid-1", Name: "bucket-1", OwnerID: "owner-uuid-1", CreatedAt: now},
		{ID: "bucket-uuid-2", Name: "bucket-2", OwnerID: "owner-uuid-1", CreatedAt: now},
	}
	repoErr := errors.New("db error")

	tests := []struct {
		name          string
		args          args
		wantBuckets   []*model.Bucket
		wantErr       error
		buildRepoMock func(mc *minimock.Controller) *mocks.BucketRepositoryMock
	}{
		{
			name:        "success: multiple buckets",
			args:        args{ctx: ctx, ownerID: "owner-uuid-1"},
			wantBuckets: buckets,
			buildRepoMock: func(mc *minimock.Controller) *mocks.BucketRepositoryMock {
				m := mocks.NewBucketRepositoryMock(mc)
				m.ListMock.Inspect(func(ctx context.Context, ownerID string) {
					require.Equal(t, "owner-uuid-1", ownerID)
				}).Return(buckets, nil)
				return m
			},
		},
		{
			name:        "success: empty list",
			args:        args{ctx: ctx, ownerID: "owner-uuid-2"},
			wantBuckets: []*model.Bucket{},
			buildRepoMock: func(mc *minimock.Controller) *mocks.BucketRepositoryMock {
				m := mocks.NewBucketRepositoryMock(mc)
				m.ListMock.Return([]*model.Bucket{}, nil)
				return m
			},
		},
		{
			name:    "error: repository failure",
			args:    args{ctx: ctx, ownerID: "owner-uuid-1"},
			wantErr: repoErr,
			buildRepoMock: func(mc *minimock.Controller) *mocks.BucketRepositoryMock {
				m := mocks.NewBucketRepositoryMock(mc)
				m.ListMock.Return(nil, repoErr)
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

			got, err := svc.ListBuckets(tt.args.ctx, tt.args.ownerID)

			require.Equal(t, tt.wantErr, err)
			if tt.wantErr == nil {
				require.Equal(t, tt.wantBuckets, got)
			}
		})
	}
}
