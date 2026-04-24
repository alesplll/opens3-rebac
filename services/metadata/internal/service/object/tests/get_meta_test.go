package tests

import (
	"context"
	"testing"
	"time"

	domainerrors "github.com/alesplll/opens3-rebac/services/metadata/internal/errors/domain_errors"
	"github.com/alesplll/opens3-rebac/services/metadata/internal/model"
	objectService "github.com/alesplll/opens3-rebac/services/metadata/internal/service/object"
	"github.com/alesplll/opens3-rebac/services/metadata/pkg/mocks"
	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/require"
)

func TestGetObjectMeta(t *testing.T) {
	type args struct {
		ctx        context.Context
		bucketName string
		key        string
		versionID  string
	}

	ctx := context.Background()

	wantMeta := &model.ObjectMeta{
		ObjectID:     "object-uuid-1",
		VersionID:    "version-uuid-1",
		BlobID:       "blob-uuid-1",
		SizeBytes:    2048,
		Etag:         `"etag123"`,
		ContentType:  "image/png",
		LastModified: time.Now(),
	}

	tests := []struct {
		name          string
		args          args
		wantMeta      *model.ObjectMeta
		wantErr       error
		buildRepoMock func(mc *minimock.Controller) *mocks.ObjectRepositoryMock
	}{
		{
			name:     "success: get latest version",
			args:     args{ctx: ctx, bucketName: "my-bucket", key: "photos/cat.jpg", versionID: ""},
			wantMeta: wantMeta,
			buildRepoMock: func(mc *minimock.Controller) *mocks.ObjectRepositoryMock {
				m := mocks.NewObjectRepositoryMock(mc)
				m.GetMetaMock.Inspect(func(ctx context.Context, bucketName, key, versionID string) {
					require.Equal(t, "my-bucket", bucketName)
					require.Equal(t, "photos/cat.jpg", key)
					require.Equal(t, "", versionID)
				}).Return(wantMeta, nil)
				return m
			},
		},
		{
			name:     "success: get specific version",
			args:     args{ctx: ctx, bucketName: "my-bucket", key: "photos/cat.jpg", versionID: "version-uuid-1"},
			wantMeta: wantMeta,
			buildRepoMock: func(mc *minimock.Controller) *mocks.ObjectRepositoryMock {
				m := mocks.NewObjectRepositoryMock(mc)
				m.GetMetaMock.Return(wantMeta, nil)
				return m
			},
		},
		{
			name:    "error: object not found",
			args:    args{ctx: ctx, bucketName: "my-bucket", key: "nonexistent.jpg", versionID: ""},
			wantErr: domainerrors.ErrObjectNotFound,
			buildRepoMock: func(mc *minimock.Controller) *mocks.ObjectRepositoryMock {
				m := mocks.NewObjectRepositoryMock(mc)
				m.GetMetaMock.Return(nil, domainerrors.ErrObjectNotFound)
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

			got, err := svc.GetObjectMeta(tt.args.ctx, tt.args.bucketName, tt.args.key, tt.args.versionID)

			require.Equal(t, tt.wantErr, err)
			if tt.wantErr == nil {
				require.Equal(t, tt.wantMeta, got)
			}
		})
	}
}
