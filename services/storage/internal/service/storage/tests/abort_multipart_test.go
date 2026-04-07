package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/alesplll/opens3-rebac/services/storage/internal/repository"
	storageService "github.com/alesplll/opens3-rebac/services/storage/internal/service/storage"
	"github.com/alesplll/opens3-rebac/services/storage/pkg/mocks"
	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/require"
)

func TestAbortMultipartUpload(t *testing.T) {
	type repoMockFunc func(mc *minimock.Controller) repository.StorageRepository

	ctx := context.Background()
	repoErr := errors.New("repo error")

	tests := []struct {
		name     string
		uploadID string
		wantErr  error
		repoMock repoMockFunc
	}{
		{
			name:     "success case",
			uploadID: "upload-1",
			repoMock: func(mc *minimock.Controller) repository.StorageRepository {
				mock := mocks.NewStorageRepositoryMock(mc)
				mock.CleanupMultipartMock.Expect(ctx, "upload-1").Return(nil)
				return mock
			},
		},
		{
			name:     "repo error case",
			uploadID: "upload-1",
			wantErr:  repoErr,
			repoMock: func(mc *minimock.Controller) repository.StorageRepository {
				mock := mocks.NewStorageRepositoryMock(mc)
				mock.CleanupMultipartMock.Expect(ctx, "upload-1").Return(repoErr)
				return mock
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := minimock.NewController(t)
			svc := storageService.NewService(tt.repoMock(mc))

			err := svc.AbortMultipartUpload(ctx, tt.uploadID)
			require.Equal(t, tt.wantErr, err)
		})
	}
}
