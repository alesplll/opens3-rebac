package tests

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	domainerrors "github.com/alesplll/opens3-rebac/services/storage/internal/errors/domain_errors"
	"github.com/alesplll/opens3-rebac/services/storage/internal/repository"
	storageService "github.com/alesplll/opens3-rebac/services/storage/internal/service/storage"
	"github.com/alesplll/opens3-rebac/services/storage/pkg/mocks"
	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/require"
)

func TestUploadPart(t *testing.T) {
	type repoMockFunc func(mc *minimock.Controller) repository.StorageRepository

	ctx := context.Background()
	body := []byte("part body")
	repoErr := errors.New("repo error")

	tests := []struct {
		name       string
		uploadID   string
		partNumber int32
		want       string
		wantErr    error
		repoMock   repoMockFunc
	}{
		{
			name:       "success case",
			uploadID:   "upload-1",
			partNumber: 1,
			want:       "checksum-1",
			repoMock: func(mc *minimock.Controller) repository.StorageRepository {
				mock := mocks.NewStorageRepositoryMock(mc)
				mock.StorePartMock.Set(func(gotCtx context.Context, uploadID string, partNumber int32, reader io.Reader) (string, error) {
					require.Equal(t, ctx, gotCtx)
					require.Equal(t, "upload-1", uploadID)
					require.Equal(t, int32(1), partNumber)
					gotBody, err := io.ReadAll(reader)
					require.NoError(t, err)
					require.Equal(t, body, gotBody)
					return "checksum-1", nil
				})
				return mock
			},
		},
		{
			name:       "rejects invalid part number",
			uploadID:   "upload-1",
			partNumber: 0,
			wantErr:    domainerrors.ErrInvalidPartNumber,
			repoMock: func(mc *minimock.Controller) repository.StorageRepository {
				return mocks.NewStorageRepositoryMock(mc)
			},
		},
		{
			name:       "repo error case",
			uploadID:   "upload-1",
			partNumber: 2,
			wantErr:    repoErr,
			repoMock: func(mc *minimock.Controller) repository.StorageRepository {
				mock := mocks.NewStorageRepositoryMock(mc)
				mock.StorePartMock.Set(func(gotCtx context.Context, uploadID string, partNumber int32, reader io.Reader) (string, error) {
					require.Equal(t, ctx, gotCtx)
					require.Equal(t, "upload-1", uploadID)
					require.Equal(t, int32(2), partNumber)
					gotBody, err := io.ReadAll(reader)
					require.NoError(t, err)
					require.Equal(t, body, gotBody)
					return "", repoErr
				})
				return mock
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := minimock.NewController(t)
			svc := storageService.NewService(tt.repoMock(mc))

			checksum, err := svc.UploadPart(ctx, tt.uploadID, tt.partNumber, bytes.NewReader(body))
			require.Equal(t, tt.wantErr, err)
			require.Equal(t, tt.want, checksum)
		})
	}
}
