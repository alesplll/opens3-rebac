package tests

import (
	"bytes"
	"context"
	"io"
	"testing"

	domainerrors "github.com/alesplll/opens3-rebac/services/storage/internal/errors/domain_errors"
	"github.com/alesplll/opens3-rebac/services/storage/internal/model"
	"github.com/alesplll/opens3-rebac/services/storage/internal/repository"
	storageService "github.com/alesplll/opens3-rebac/services/storage/internal/service/storage"
	"github.com/alesplll/opens3-rebac/services/storage/pkg/mocks"
	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/require"
)

func TestStoreObject(t *testing.T) {
	type repoMockFunc func(mc *minimock.Controller) repository.StorageRepository

	type args struct {
		ctx         context.Context
		reader      io.Reader
		size        *int64
		contentType string
	}

	var (
		ctx         = context.Background()
		content     = []byte("storage content")
		contentType = "application/json"
	)

	tests := []struct {
		name     string
		args     args
		want     *model.BlobMeta
		err      error
		repoMock repoMockFunc
	}{
		{
			name: "success case",
			args: args{
				ctx:         ctx,
				reader:      bytes.NewReader(content),
				size:        int64Ptr(int64(len(content))),
				contentType: contentType,
			},
			want: &model.BlobMeta{
				BlobID:      "",
				ChecksumMD5: "checksum-1",
				SizeBytes:   int64(len(content)),
				ContentType: contentType,
			},
			err: nil,
			repoMock: func(mc *minimock.Controller) repository.StorageRepository {
				mock := mocks.NewStorageRepositoryMock(mc)
				mock.StoreBlobMock.Set(func(gotCtx context.Context, blobID string, reader io.Reader) (*model.BlobMeta, error) {
					require.Equal(t, ctx, gotCtx)
					require.NotEmpty(t, blobID)
					body, err := io.ReadAll(reader)
					require.NoError(t, err)
					require.Equal(t, content, body)

					return &model.BlobMeta{
						BlobID:      blobID,
						ChecksumMD5: "checksum-1",
						SizeBytes:   int64(len(content)),
					}, nil
				})
				return mock
			},
		},
		{
			name: "uses default content type when empty",
			args: args{
				ctx:         ctx,
				reader:      bytes.NewReader(content),
				size:        int64Ptr(int64(len(content))),
				contentType: "",
			},
			want: &model.BlobMeta{
				BlobID:      "",
				ChecksumMD5: "checksum-2",
				SizeBytes:   int64(len(content)),
				ContentType: "application/octet-stream",
			},
			err: nil,
			repoMock: func(mc *minimock.Controller) repository.StorageRepository {
				mock := mocks.NewStorageRepositoryMock(mc)
				mock.StoreBlobMock.Set(func(gotCtx context.Context, blobID string, reader io.Reader) (*model.BlobMeta, error) {
					require.Equal(t, ctx, gotCtx)
					require.NotEmpty(t, blobID)
					body, err := io.ReadAll(reader)
					require.NoError(t, err)
					require.Equal(t, content, body)

					return &model.BlobMeta{
						BlobID:      blobID,
						ChecksumMD5: "checksum-2",
						SizeBytes:   int64(len(content)),
					}, nil
				})
				return mock
			},
		},
		{
			name: "returns invalid blob size for negative size",
			args: args{
				ctx:         ctx,
				reader:      bytes.NewReader(content),
				size:        int64Ptr(-1),
				contentType: contentType,
			},
			want: nil,
			err:  domainerrors.ErrInvalidBlobSize,
			repoMock: func(mc *minimock.Controller) repository.StorageRepository {
				return mocks.NewStorageRepositoryMock(mc)
			},
		},
		{
			name: "skips size validation when size is not provided",
			args: args{
				ctx:         ctx,
				reader:      bytes.NewReader(content),
				size:        nil,
				contentType: contentType,
			},
			want: &model.BlobMeta{
				BlobID:      "",
				ChecksumMD5: "checksum-3",
				SizeBytes:   int64(len(content) + 7),
				ContentType: contentType,
			},
			err: nil,
			repoMock: func(mc *minimock.Controller) repository.StorageRepository {
				mock := mocks.NewStorageRepositoryMock(mc)
				mock.StoreBlobMock.Set(func(gotCtx context.Context, blobID string, reader io.Reader) (*model.BlobMeta, error) {
					require.Equal(t, ctx, gotCtx)
					require.NotEmpty(t, blobID)
					body, err := io.ReadAll(reader)
					require.NoError(t, err)
					require.Equal(t, content, body)

					return &model.BlobMeta{
						BlobID:      blobID,
						ChecksumMD5: "checksum-3",
						SizeBytes:   int64(len(content) + 7),
					}, nil
				})
				return mock
			},
		},
		{
			name: "cleans up stored blob when actual size mismatches expected size",
			args: args{
				ctx:         ctx,
				reader:      bytes.NewReader(content),
				size:        int64Ptr(int64(len(content) + 5)),
				contentType: contentType,
			},
			want: nil,
			err:  domainerrors.ErrInvalidBlobSize,
			repoMock: func(mc *minimock.Controller) repository.StorageRepository {
				mock := mocks.NewStorageRepositoryMock(mc)
				mock.StoreBlobMock.Set(func(gotCtx context.Context, blobID string, reader io.Reader) (*model.BlobMeta, error) {
					require.Equal(t, ctx, gotCtx)
					require.NotEmpty(t, blobID)
					body, err := io.ReadAll(reader)
					require.NoError(t, err)
					require.Equal(t, content, body)

					return &model.BlobMeta{
						BlobID:      blobID,
						ChecksumMD5: "checksum-mismatch",
						SizeBytes:   int64(len(content)),
					}, nil
				})
				mock.DeleteBlobMock.Set(func(gotCtx context.Context, blobID string) error {
					require.Equal(t, ctx, gotCtx)
					require.NotEmpty(t, blobID)
					return nil
				})
				return mock
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := minimock.NewController(t)
			repoMock := tt.repoMock(mc)
			svc := storageService.NewService(repoMock)

			res, err := svc.StoreObject(tt.args.ctx, tt.args.reader, tt.args.size, tt.args.contentType)
			require.Equal(t, tt.err, err)
			if tt.want == nil {
				require.Nil(t, res)
				return
			}

			require.NotNil(t, res)
			require.NotEmpty(t, res.BlobID)
			require.Equal(t, tt.want.ChecksumMD5, res.ChecksumMD5)
			require.Equal(t, tt.want.SizeBytes, res.SizeBytes)
			require.Equal(t, tt.want.ContentType, res.ContentType)
		})
	}
}

func int64Ptr(v int64) *int64 {
	return &v
}
