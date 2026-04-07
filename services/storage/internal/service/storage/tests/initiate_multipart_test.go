package tests

import (
	"context"
	"errors"
	"testing"

	domainerrors "github.com/alesplll/opens3-rebac/services/storage/internal/errors/domain_errors"
	"github.com/alesplll/opens3-rebac/services/storage/internal/repository"
	storageService "github.com/alesplll/opens3-rebac/services/storage/internal/service/storage"
	"github.com/alesplll/opens3-rebac/services/storage/pkg/mocks"
	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/require"
)

func TestInitiateMultipartUpload(t *testing.T) {
	type repoMockFunc func(mc *minimock.Controller) repository.StorageRepository

	ctx := context.Background()
	repoErr := errors.New("repo error")

	tests := []struct {
		name          string
		expectedParts int32
		contentType   string
		wantContent   string
		wantErr       error
		repoMock      repoMockFunc
	}{
		{
			name:          "success case",
			expectedParts: 3,
			contentType:   "video/mp4",
			wantContent:   "video/mp4",
			repoMock: func(mc *minimock.Controller) repository.StorageRepository {
				mock := mocks.NewStorageRepositoryMock(mc)
				mock.CreateMultipartSessionMock.Set(func(gotCtx context.Context, uploadID string, expectedParts int32, contentType string) error {
					require.Equal(t, ctx, gotCtx)
					require.NotEmpty(t, uploadID)
					require.Equal(t, int32(3), expectedParts)
					require.Equal(t, "video/mp4", contentType)
					return nil
				})
				return mock
			},
		},
		{
			name:          "defaults content type when empty",
			expectedParts: 1,
			contentType:   "",
			wantContent:   "application/octet-stream",
			repoMock: func(mc *minimock.Controller) repository.StorageRepository {
				mock := mocks.NewStorageRepositoryMock(mc)
				mock.CreateMultipartSessionMock.Set(func(gotCtx context.Context, uploadID string, expectedParts int32, contentType string) error {
					require.Equal(t, ctx, gotCtx)
					require.NotEmpty(t, uploadID)
					require.Equal(t, int32(1), expectedParts)
					require.Equal(t, "application/octet-stream", contentType)
					return nil
				})
				return mock
			},
		},
		{
			name:          "rejects negative expected parts",
			expectedParts: -1,
			contentType:   "video/mp4",
			wantErr:       domainerrors.ErrInvalidParts,
			repoMock: func(mc *minimock.Controller) repository.StorageRepository {
				return mocks.NewStorageRepositoryMock(mc)
			},
		},
		{
			name:          "repo error case",
			expectedParts: 2,
			contentType:   "video/mp4",
			wantErr:       repoErr,
			repoMock: func(mc *minimock.Controller) repository.StorageRepository {
				mock := mocks.NewStorageRepositoryMock(mc)
				mock.CreateMultipartSessionMock.Set(func(gotCtx context.Context, uploadID string, expectedParts int32, contentType string) error {
					require.Equal(t, ctx, gotCtx)
					require.NotEmpty(t, uploadID)
					require.Equal(t, int32(2), expectedParts)
					require.Equal(t, "video/mp4", contentType)
					return repoErr
				})
				return mock
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := minimock.NewController(t)
			svc := storageService.NewService(tt.repoMock(mc))

			uploadID, err := svc.InitiateMultipartUpload(ctx, tt.expectedParts, tt.contentType)
			require.Equal(t, tt.wantErr, err)

			if tt.wantErr != nil {
				require.Empty(t, uploadID)
				return
			}

			require.NotEmpty(t, uploadID)
		})
	}
}
