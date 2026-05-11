package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	objectService "github.com/alesplll/opens3-rebac/services/metadata/internal/service/object"
	"github.com/alesplll/opens3-rebac/services/metadata/pkg/mocks"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/client/db"
	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/require"
)

func TestCreateObjectVersion(t *testing.T) {
	type args struct {
		ctx         context.Context
		bucketName  string
		key         string
		blobID      string
		sizeBytes   int64
		etag        string
		contentType string
	}

	ctx := context.Background()

	wantObjectID  := "object-uuid-1"
	wantVersionID := "version-uuid-1"
	wantCreatedAt := time.Now()

	// txMock that actually executes the handler
	executingTx := func(mc *minimock.Controller) *mocks.TxManagerMock {
		m := mocks.NewTxManagerMock(mc)
		m.ReadCommittedMock.Set(func(ctx context.Context, f db.Handler) error {
			return f(ctx)
		})
		return m
	}

	tests := []struct {
		name                string
		args                args
		wantObjectID        string
		wantVersionID       string
		wantErr             error
		wantValidationError bool
		buildRepoMock       func(mc *minimock.Controller) *mocks.ObjectRepositoryMock
		buildTxMock         func(mc *minimock.Controller) *mocks.TxManagerMock
	}{
		{
			name: "success",
			args: args{
				ctx:         ctx,
				bucketName:  "my-bucket",
				key:         "photos/cat.jpg",
				blobID:      "blob-uuid-1",
				sizeBytes:   1024,
				etag:        `"abc123"`,
				contentType: "image/jpeg",
			},
			wantObjectID:  wantObjectID,
			wantVersionID: wantVersionID,
			buildRepoMock: func(mc *minimock.Controller) *mocks.ObjectRepositoryMock {
				m := mocks.NewObjectRepositoryMock(mc)
				m.UpsertObjectMock.Inspect(func(ctx context.Context, bucketName, key string) {
					require.Equal(t, "my-bucket", bucketName)
					require.Equal(t, "photos/cat.jpg", key)
				}).Return(wantObjectID, nil)
				m.InsertVersionMock.Inspect(func(ctx context.Context, objectID, blobID string, sizeBytes int64, etag, contentType string) {
					require.Equal(t, wantObjectID, objectID)
					require.Equal(t, "blob-uuid-1", blobID)
					require.Equal(t, int64(1024), sizeBytes)
				}).Return(wantVersionID, wantCreatedAt, nil)
				m.SetCurrentVersionMock.Inspect(func(ctx context.Context, objectID, versionID string) {
					require.Equal(t, wantObjectID, objectID)
					require.Equal(t, wantVersionID, versionID)
				}).Return(nil)
				return m
			},
			buildTxMock: executingTx,
		},
		{
			name:                "validation error: empty bucketName",
			args:                args{ctx: ctx, bucketName: "", key: "photos/cat.jpg", blobID: "blob-uuid-1"},
			wantValidationError: true,
			buildRepoMock: func(mc *minimock.Controller) *mocks.ObjectRepositoryMock {
				return mocks.NewObjectRepositoryMock(mc)
			},
			buildTxMock: func(mc *minimock.Controller) *mocks.TxManagerMock {
				return mocks.NewTxManagerMock(mc)
			},
		},
		{
			name:                "validation error: empty key",
			args:                args{ctx: ctx, bucketName: "my-bucket", key: "", blobID: "blob-uuid-1"},
			wantValidationError: true,
			buildRepoMock: func(mc *minimock.Controller) *mocks.ObjectRepositoryMock {
				return mocks.NewObjectRepositoryMock(mc)
			},
			buildTxMock: func(mc *minimock.Controller) *mocks.TxManagerMock {
				return mocks.NewTxManagerMock(mc)
			},
		},
		{
			name:                "validation error: empty blobID",
			args:                args{ctx: ctx, bucketName: "my-bucket", key: "photos/cat.jpg", blobID: ""},
			wantValidationError: true,
			buildRepoMock: func(mc *minimock.Controller) *mocks.ObjectRepositoryMock {
				return mocks.NewObjectRepositoryMock(mc)
			},
			buildTxMock: func(mc *minimock.Controller) *mocks.TxManagerMock {
				return mocks.NewTxManagerMock(mc)
			},
		},
		{
			name:    "error: UpsertObject fails",
			args:    args{ctx: ctx, bucketName: "my-bucket", key: "photos/cat.jpg", blobID: "blob-uuid-1"},
			wantErr: errors.New("upsert error"),
			buildRepoMock: func(mc *minimock.Controller) *mocks.ObjectRepositoryMock {
				m := mocks.NewObjectRepositoryMock(mc)
				m.UpsertObjectMock.Return("", errors.New("upsert error"))
				return m
			},
			buildTxMock: executingTx,
		},
		{
			name:    "error: InsertVersion fails",
			args:    args{ctx: ctx, bucketName: "my-bucket", key: "photos/cat.jpg", blobID: "blob-uuid-1"},
			wantErr: errors.New("insert version error"),
			buildRepoMock: func(mc *minimock.Controller) *mocks.ObjectRepositoryMock {
				m := mocks.NewObjectRepositoryMock(mc)
				m.UpsertObjectMock.Return(wantObjectID, nil)
				m.InsertVersionMock.Return("", time.Time{}, errors.New("insert version error"))
				return m
			},
			buildTxMock: executingTx,
		},
		{
			name:    "error: SetCurrentVersion fails",
			args:    args{ctx: ctx, bucketName: "my-bucket", key: "photos/cat.jpg", blobID: "blob-uuid-1"},
			wantErr: errors.New("set current version error"),
			buildRepoMock: func(mc *minimock.Controller) *mocks.ObjectRepositoryMock {
				m := mocks.NewObjectRepositoryMock(mc)
				m.UpsertObjectMock.Return(wantObjectID, nil)
				m.InsertVersionMock.Return(wantVersionID, wantCreatedAt, nil)
				m.SetCurrentVersionMock.Return(errors.New("set current version error"))
				return m
			},
			buildTxMock: executingTx,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := minimock.NewController(t)

			repoMock := tt.buildRepoMock(mc)
			txMock := tt.buildTxMock(mc)
			producerMock := mocks.NewProducerMock(mc)

			svc := objectService.NewService(repoMock, txMock, producerMock, nil, nil)

			objectID, versionID, createdAt, err := svc.CreateObjectVersion(
				tt.args.ctx,
				tt.args.bucketName,
				tt.args.key,
				tt.args.blobID,
				tt.args.sizeBytes,
				tt.args.etag,
				tt.args.contentType,
			)

			if tt.wantValidationError {
				requireValidationError(t, err)
				require.Equal(t, uint64(0), repoMock.UpsertObjectBeforeCounter())
				return
			}

			require.Equal(t, tt.wantErr, err)
			if tt.wantErr == nil {
				require.Equal(t, tt.wantObjectID, objectID)
				require.Equal(t, tt.wantVersionID, versionID)
				require.False(t, createdAt.IsZero())
			}
		})
	}
}
