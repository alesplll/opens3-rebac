package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/alesplll/opens3-rebac/services/storage/internal/repository"
	storageService "github.com/alesplll/opens3-rebac/services/storage/internal/service/storage"
	"github.com/alesplll/opens3-rebac/services/storage/pkg/mocks"
	"github.com/brianvoe/gofakeit"
	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/require"
)

func TestDeleteObject(t *testing.T) {
	type repoMockFunc func(mc *minimock.Controller) repository.StorageRepository

	type args struct {
		ctx    context.Context
		blobID string
	}

	var (
		ctx    = context.Background()
		blobID = gofakeit.UUID()

		repoErr = errors.New("repo error")
	)

	tests := []struct {
		name     string
		args     args
		err      error
		repoMock repoMockFunc
	}{
		{
			name: "success case",
			args: args{
				ctx:    ctx,
				blobID: blobID,
			},
			err: nil,
			repoMock: func(mc *minimock.Controller) repository.StorageRepository {
				mock := mocks.NewStorageRepositoryMock(mc)
				mock.DeleteBlobMock.Expect(ctx, blobID).Return(nil)
				return mock
			},
		},
		{
			name: "repo error case",
			args: args{
				ctx:    ctx,
				blobID: blobID,
			},
			err: repoErr,
			repoMock: func(mc *minimock.Controller) repository.StorageRepository {
				mock := mocks.NewStorageRepositoryMock(mc)
				mock.DeleteBlobMock.Expect(ctx, blobID).Return(repoErr)
				return mock
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := minimock.NewController(t)
			repoMock := tt.repoMock(mc)
			svc := storageService.NewService(repoMock)

			err := svc.DeleteObject(tt.args.ctx, tt.args.blobID)
			require.Equal(t, tt.err, err)
		})
	}
}
