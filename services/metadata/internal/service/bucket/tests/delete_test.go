package tests

import (
	"context"
	"errors"
	"testing"

	domainerrors "github.com/alesplll/opens3-rebac/services/metadata/internal/errors/domain_errors"
	"github.com/alesplll/opens3-rebac/services/metadata/internal/model"
	bucketService "github.com/alesplll/opens3-rebac/services/metadata/internal/service/bucket"
	"github.com/alesplll/opens3-rebac/services/metadata/pkg/mocks"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/client/db"
	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/require"
)

func TestDeleteBucket(t *testing.T) {
	type args struct {
		ctx  context.Context
		name string
	}

	type repoMockBuilder func(mc *minimock.Controller) *mocks.BucketRepositoryMock
	type txMockBuilder func(mc *minimock.Controller) *mocks.TxManagerMock
	type producerMockBuilder func(mc *minimock.Controller) *mocks.ProducerMock

	ctx := context.Background()

	existingBucket := &model.Bucket{
		ID:      "bucket-uuid-1",
		Name:    "my-bucket",
		OwnerID: "owner-uuid-1",
	}

	// txMock that actually executes the handler
	executingTx := func(mc *minimock.Controller) *mocks.TxManagerMock {
		m := mocks.NewTxManagerMock(mc)
		m.ReadCommittedMock.Set(func(ctx context.Context, f db.Handler) error {
			return f(ctx)
		})
		return m
	}

	tests := []struct {
		name              string
		args              args
		wantErr           error
		buildRepoMock     repoMockBuilder
		buildTxMock       txMockBuilder
		buildProducerMock producerMockBuilder
	}{
		{
			name: "success: empty bucket deleted and event sent",
			args: args{ctx: ctx, name: "my-bucket"},
			buildRepoMock: func(mc *minimock.Controller) *mocks.BucketRepositoryMock {
				m := mocks.NewBucketRepositoryMock(mc)
				m.GetMock.Return(existingBucket, nil)
				m.CountObjectsMock.Return(0, nil)
				m.DeleteMock.Inspect(func(ctx context.Context, bucketID string) {
					require.Equal(t, existingBucket.ID, bucketID)
				}).Return(nil)
				return m
			},
			buildTxMock: executingTx,
			buildProducerMock: func(mc *minimock.Controller) *mocks.ProducerMock {
				m := mocks.NewProducerMock(mc)
				m.SendMock.Return(nil)
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
			buildTxMock: executingTx,
			buildProducerMock: func(mc *minimock.Controller) *mocks.ProducerMock {
				return mocks.NewProducerMock(mc)
			},
		},
		{
			name:    "error: bucket not empty",
			args:    args{ctx: ctx, name: "my-bucket"},
			wantErr: domainerrors.ErrBucketNotEmpty,
			buildRepoMock: func(mc *minimock.Controller) *mocks.BucketRepositoryMock {
				m := mocks.NewBucketRepositoryMock(mc)
				m.GetMock.Return(existingBucket, nil)
				m.CountObjectsMock.Return(3, nil)
				return m
			},
			buildTxMock: executingTx,
			buildProducerMock: func(mc *minimock.Controller) *mocks.ProducerMock {
				return mocks.NewProducerMock(mc)
			},
		},
		{
			name:    "error: CountObjects fails",
			args:    args{ctx: ctx, name: "my-bucket"},
			wantErr: errors.New("db error"),
			buildRepoMock: func(mc *minimock.Controller) *mocks.BucketRepositoryMock {
				m := mocks.NewBucketRepositoryMock(mc)
				m.GetMock.Return(existingBucket, nil)
				m.CountObjectsMock.Return(0, errors.New("db error"))
				return m
			},
			buildTxMock: executingTx,
			buildProducerMock: func(mc *minimock.Controller) *mocks.ProducerMock {
				return mocks.NewProducerMock(mc)
			},
		},
		{
			name:    "error: Delete fails",
			args:    args{ctx: ctx, name: "my-bucket"},
			wantErr: errors.New("delete error"),
			buildRepoMock: func(mc *minimock.Controller) *mocks.BucketRepositoryMock {
				m := mocks.NewBucketRepositoryMock(mc)
				m.GetMock.Return(existingBucket, nil)
				m.CountObjectsMock.Return(0, nil)
				m.DeleteMock.Return(errors.New("delete error"))
				return m
			},
			buildTxMock: executingTx,
			buildProducerMock: func(mc *minimock.Controller) *mocks.ProducerMock {
				return mocks.NewProducerMock(mc)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := minimock.NewController(t)

			repoMock := tt.buildRepoMock(mc)
			txMock := tt.buildTxMock(mc)
			producerMock := tt.buildProducerMock(mc)

			svc := bucketService.NewService(repoMock, txMock, producerMock)

			err := svc.DeleteBucket(tt.args.ctx, tt.args.name)

			require.Equal(t, tt.wantErr, err)
		})
	}
}
