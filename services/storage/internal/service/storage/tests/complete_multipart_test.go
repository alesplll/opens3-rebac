package tests

import (
	"context"
	"errors"
	"testing"

	domainerrors "github.com/alesplll/opens3-rebac/services/storage/internal/errors/domain_errors"
	"github.com/alesplll/opens3-rebac/services/storage/internal/model"
	"github.com/alesplll/opens3-rebac/services/storage/internal/repository"
	storageService "github.com/alesplll/opens3-rebac/services/storage/internal/service/storage"
	"github.com/alesplll/opens3-rebac/services/storage/pkg/mocks"
	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/require"
)

func TestCompleteMultipartUpload(t *testing.T) {
	type repoMockFunc func(mc *minimock.Controller) repository.StorageRepository

	ctx := context.Background()
	parts := []model.PartInfo{
		{PartNumber: 1, ChecksumMD5: "checksum-1"},
		{PartNumber: 2, ChecksumMD5: "checksum-2"},
	}
	repoResult := &model.BlobMeta{
		BlobID:      "blob-1",
		ChecksumMD5: "blob-checksum",
		SizeBytes:   128,
		ContentType: "video/mp4",
	}
	repoErr := errors.New("repo error")

	tests := []struct {
		name     string
		uploadID string
		parts    []model.PartInfo
		want     *model.BlobMeta
		wantErr  error
		repoMock repoMockFunc
	}{
		{
			name:     "success case",
			uploadID: "upload-1",
			parts:    parts,
			want:     repoResult,
			repoMock: func(mc *minimock.Controller) repository.StorageRepository {
				mock := mocks.NewStorageRepositoryMock(mc)
				mock.AssemblePartsMock.Set(func(gotCtx context.Context, uploadID string, gotParts []model.PartInfo, destBlobID string) (*model.BlobMeta, error) {
					require.Equal(t, ctx, gotCtx)
					require.Equal(t, "upload-1", uploadID)
					require.Equal(t, parts, gotParts)
					require.Equal(t, "upload-1", destBlobID)
					return repoResult, nil
				})
				return mock
			},
		},
		{
			name:     "rejects empty parts list",
			uploadID: "upload-1",
			parts:    nil,
			wantErr:  domainerrors.ErrInvalidParts,
			repoMock: func(mc *minimock.Controller) repository.StorageRepository {
				return mocks.NewStorageRepositoryMock(mc)
			},
		},
		{
			name:     "rejects empty checksum",
			uploadID: "upload-1",
			parts: []model.PartInfo{
				{PartNumber: 1, ChecksumMD5: ""},
			},
			wantErr: domainerrors.ErrChecksumMismatch,
			repoMock: func(mc *minimock.Controller) repository.StorageRepository {
				return mocks.NewStorageRepositoryMock(mc)
			},
		},
		{
			name:     "rejects unordered parts",
			uploadID: "upload-1",
			parts: []model.PartInfo{
				{PartNumber: 2, ChecksumMD5: "checksum-2"},
				{PartNumber: 1, ChecksumMD5: "checksum-1"},
			},
			wantErr: domainerrors.ErrInvalidParts,
			repoMock: func(mc *minimock.Controller) repository.StorageRepository {
				return mocks.NewStorageRepositoryMock(mc)
			},
		},
		{
			name:     "repo error case",
			uploadID: "upload-1",
			parts:    parts,
			wantErr:  repoErr,
			repoMock: func(mc *minimock.Controller) repository.StorageRepository {
				mock := mocks.NewStorageRepositoryMock(mc)
				mock.AssemblePartsMock.Set(func(gotCtx context.Context, uploadID string, gotParts []model.PartInfo, destBlobID string) (*model.BlobMeta, error) {
					require.Equal(t, ctx, gotCtx)
					require.Equal(t, "upload-1", uploadID)
					require.Equal(t, parts, gotParts)
					require.Equal(t, "upload-1", destBlobID)
					return nil, repoErr
				})
				return mock
			},
		},
		{
			name:     "missing part propagates not found",
			uploadID: "upload-1",
			parts:    parts,
			wantErr:  domainerrors.ErrUploadNotFound,
			repoMock: func(mc *minimock.Controller) repository.StorageRepository {
				mock := mocks.NewStorageRepositoryMock(mc)
				mock.AssemblePartsMock.Set(func(gotCtx context.Context, uploadID string, gotParts []model.PartInfo, destBlobID string) (*model.BlobMeta, error) {
					require.Equal(t, ctx, gotCtx)
					require.Equal(t, "upload-1", uploadID)
					require.Equal(t, parts, gotParts)
					require.Equal(t, "upload-1", destBlobID)
					return nil, domainerrors.ErrUploadNotFound
				})
				return mock
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := minimock.NewController(t)
			svc := storageService.NewService(tt.repoMock(mc))

			meta, err := svc.CompleteMultipartUpload(ctx, tt.uploadID, tt.parts)
			require.Equal(t, tt.wantErr, err)
			require.Equal(t, tt.want, meta)
		})
	}
}
