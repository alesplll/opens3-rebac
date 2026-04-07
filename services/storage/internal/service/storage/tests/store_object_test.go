package tests

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/alesplll/opens3-rebac/services/storage/internal/model"
	"github.com/alesplll/opens3-rebac/services/storage/internal/repository"
	storageService "github.com/alesplll/opens3-rebac/services/storage/internal/service/storage"
	"github.com/alesplll/opens3-rebac/services/storage/pkg/mocks"
	"github.com/brianvoe/gofakeit"
	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/require"
)

func TestStoreObject(t *testing.T) {
	type repoMockFunc func(mc *minimock.Controller) repository.StorageRepository

	type args struct {
		ctx         context.Context
		reader      io.Reader
		size        int64
		contentType string
	}

	var (
		ctx         = context.Background()
		content     = []byte(gofakeit.Sentence(10))
		contentType = "application/octet-stream"

		repoBlobMeta = &model.BlobMeta{
			BlobID:      gofakeit.UUID(),
			ChecksumMD5: gofakeit.UUID(),
			SizeBytes:   int64(len(content)),
			ContentType: "",
		}

		wantBlobMeta = &model.BlobMeta{
			BlobID:      repoBlobMeta.BlobID,
			ChecksumMD5: repoBlobMeta.ChecksumMD5,
			SizeBytes:   repoBlobMeta.SizeBytes,
			ContentType: contentType,
		}

		repoErr = errors.New("repo error")
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
				size:        int64(len(content)),
				contentType: contentType,
			},
			want: wantBlobMeta,
			err:  nil,
			repoMock: func(mc *minimock.Controller) repository.StorageRepository {
				mock := mocks.NewStorageRepositoryMock(mc)
				mock.StoreBlobMock.Expect(ctx, bytes.NewReader(content)).Return(&model.BlobMeta{
					BlobID:      repoBlobMeta.BlobID,
					ChecksumMD5: repoBlobMeta.ChecksumMD5,
					SizeBytes:   repoBlobMeta.SizeBytes,
					ContentType: repoBlobMeta.ContentType,
				}, nil)
				return mock
			},
		},
		{
			name: "repo error case",
			args: args{
				ctx:         ctx,
				reader:      bytes.NewReader(content),
				size:        int64(len(content)),
				contentType: contentType,
			},
			want: nil,
			err:  repoErr,
			repoMock: func(mc *minimock.Controller) repository.StorageRepository {
				mock := mocks.NewStorageRepositoryMock(mc)
				mock.StoreBlobMock.Expect(ctx, bytes.NewReader(content)).Return(nil, repoErr)
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
			require.Equal(t, tt.want, res)
		})
	}
}
