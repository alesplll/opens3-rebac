package tests

import (
	"context"
	"errors"
	"testing"

	domainerrors "github.com/alesplll/opens3-rebac/services/metadata/internal/errors/domain_errors"
	objectService "github.com/alesplll/opens3-rebac/services/metadata/internal/service/object"
	"github.com/alesplll/opens3-rebac/services/metadata/pkg/mocks"
	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/require"
)

func TestDeleteObjectMeta(t *testing.T) {
	type args struct {
		ctx        context.Context
		bucketName string
		key        string
	}

	ctx := context.Background()

	wantObjectID := "object-uuid-1"
	wantBlobID   := "blob-uuid-1"

	tests := []struct {
		name              string
		args              args
		wantObjectID      string
		wantBlobID        string
		wantErr           error
		buildRepoMock     func(mc *minimock.Controller) *mocks.ObjectRepositoryMock
		buildProducerMock func(mc *minimock.Controller) *mocks.ProducerMock
	}{
		{
			name:         "success: object deleted and event sent",
			args:         args{ctx: ctx, bucketName: "my-bucket", key: "photos/cat.jpg"},
			wantObjectID: wantObjectID,
			wantBlobID:   wantBlobID,
			buildRepoMock: func(mc *minimock.Controller) *mocks.ObjectRepositoryMock {
				m := mocks.NewObjectRepositoryMock(mc)
				m.DeleteMock.Inspect(func(ctx context.Context, bucketName, key string) {
					require.Equal(t, "my-bucket", bucketName)
					require.Equal(t, "photos/cat.jpg", key)
				}).Return(wantObjectID, wantBlobID, nil)
				return m
			},
			buildProducerMock: func(mc *minimock.Controller) *mocks.ProducerMock {
				m := mocks.NewProducerMock(mc)
				m.SendMock.Return(nil)
				return m
			},
		},
		{
			name:    "error: object not found",
			args:    args{ctx: ctx, bucketName: "my-bucket", key: "nonexistent.jpg"},
			wantErr: domainerrors.ErrObjectNotFound,
			buildRepoMock: func(mc *minimock.Controller) *mocks.ObjectRepositoryMock {
				m := mocks.NewObjectRepositoryMock(mc)
				m.DeleteMock.Return("", "", domainerrors.ErrObjectNotFound)
				return m
			},
			buildProducerMock: func(mc *minimock.Controller) *mocks.ProducerMock {
				return mocks.NewProducerMock(mc)
			},
		},
		{
			name:    "error: repository failure",
			args:    args{ctx: ctx, bucketName: "my-bucket", key: "photos/cat.jpg"},
			wantErr: errors.New("db error"),
			buildRepoMock: func(mc *minimock.Controller) *mocks.ObjectRepositoryMock {
				m := mocks.NewObjectRepositoryMock(mc)
				m.DeleteMock.Return("", "", errors.New("db error"))
				return m
			},
			buildProducerMock: func(mc *minimock.Controller) *mocks.ProducerMock {
				return mocks.NewProducerMock(mc)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := minimock.NewController(t)

			repoMock := tt.buildRepoMock(mc)
			txMock := mocks.NewTxManagerMock(mc)
			producerMock := tt.buildProducerMock(mc)

			svc := objectService.NewService(repoMock, txMock, producerMock, nil, nil)

			objectID, blobID, err := svc.DeleteObjectMeta(tt.args.ctx, tt.args.bucketName, tt.args.key)

			require.Equal(t, tt.wantErr, err)
			if tt.wantErr == nil {
				require.Equal(t, tt.wantObjectID, objectID)
				require.Equal(t, tt.wantBlobID, blobID)
			}
		})
	}
}
