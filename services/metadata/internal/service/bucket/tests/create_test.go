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

func TestCreateBucket(t *testing.T) {
	type args struct {
		ctx     context.Context
		name    string
		ownerID string
	}

	type repoMockBuilder func(mc *minimock.Controller) *mocks.BucketRepositoryMock
	type txMockBuilder func(mc *minimock.Controller) *mocks.TxManagerMock
	type producerMockBuilder func(mc *minimock.Controller) *mocks.ProducerMock

	ctx := context.Background()

	wantBucket := &model.Bucket{
		ID:        "bucket-uuid-1",
		Name:      "my-bucket",
		OwnerID:   "owner-uuid-1",
		CreatedAt: time.Now(),
	}
	repoErr := errors.New("repository create failed")

	tests := []struct {
		name                string
		args                args
		wantBucket          *model.Bucket
		wantErr             error
		wantValidationError bool
		buildRepoMock       repoMockBuilder
		buildTxMock         txMockBuilder
		buildProducerMock   producerMockBuilder
	}{
		{
			name: "success",
			args: args{ctx: ctx, name: "my-bucket", ownerID: "owner-uuid-1"},
			wantBucket: wantBucket,
			buildRepoMock: func(mc *minimock.Controller) *mocks.BucketRepositoryMock {
				m := mocks.NewBucketRepositoryMock(mc)
				m.CreateMock.Inspect(func(ctx context.Context, name, ownerID string) {
					require.Equal(t, "my-bucket", name)
					require.Equal(t, "owner-uuid-1", ownerID)
				}).Return(wantBucket, nil)
				return m
			},
			buildTxMock:       func(mc *minimock.Controller) *mocks.TxManagerMock { return mocks.NewTxManagerMock(mc) },
			buildProducerMock: func(mc *minimock.Controller) *mocks.ProducerMock { return mocks.NewProducerMock(mc) },
		},
		{
			name:                "validation error: empty name",
			args:                args{ctx: ctx, name: "", ownerID: "owner-uuid-1"},
			wantValidationError: true,
			buildRepoMock: func(mc *minimock.Controller) *mocks.BucketRepositoryMock {
				return mocks.NewBucketRepositoryMock(mc)
			},
			buildTxMock:       func(mc *minimock.Controller) *mocks.TxManagerMock { return mocks.NewTxManagerMock(mc) },
			buildProducerMock: func(mc *minimock.Controller) *mocks.ProducerMock { return mocks.NewProducerMock(mc) },
		},
		{
			name:                "validation error: empty ownerID",
			args:                args{ctx: ctx, name: "my-bucket", ownerID: ""},
			wantValidationError: true,
			buildRepoMock: func(mc *minimock.Controller) *mocks.BucketRepositoryMock {
				return mocks.NewBucketRepositoryMock(mc)
			},
			buildTxMock:       func(mc *minimock.Controller) *mocks.TxManagerMock { return mocks.NewTxManagerMock(mc) },
			buildProducerMock: func(mc *minimock.Controller) *mocks.ProducerMock { return mocks.NewProducerMock(mc) },
		},
		{
			name:    "repository error",
			args:    args{ctx: ctx, name: "my-bucket", ownerID: "owner-uuid-1"},
			wantErr: repoErr,
			buildRepoMock: func(mc *minimock.Controller) *mocks.BucketRepositoryMock {
				m := mocks.NewBucketRepositoryMock(mc)
				m.CreateMock.Return(nil, repoErr)
				return m
			},
			buildTxMock:       func(mc *minimock.Controller) *mocks.TxManagerMock { return mocks.NewTxManagerMock(mc) },
			buildProducerMock: func(mc *minimock.Controller) *mocks.ProducerMock { return mocks.NewProducerMock(mc) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := minimock.NewController(t)

			repoMock := tt.buildRepoMock(mc)
			txMock := tt.buildTxMock(mc)
			producerMock := tt.buildProducerMock(mc)

			svc := bucketService.NewService(repoMock, txMock, producerMock)

			got, err := svc.CreateBucket(tt.args.ctx, tt.args.name, tt.args.ownerID)

			if tt.wantValidationError {
				requireValidationError(t, err)
				require.Equal(t, uint64(0), repoMock.CreateBeforeCounter())
				return
			}

			require.Equal(t, tt.wantErr, err)
			if tt.wantErr == nil {
				require.Equal(t, tt.wantBucket, got)
			}
		})
	}
}
