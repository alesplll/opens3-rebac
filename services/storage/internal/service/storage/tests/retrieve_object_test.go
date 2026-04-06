package tests

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/alesplll/opens3-rebac/services/storage/internal/repository"
	storageService "github.com/alesplll/opens3-rebac/services/storage/internal/service/storage"
	"github.com/alesplll/opens3-rebac/services/storage/pkg/mocks"
	"github.com/brianvoe/gofakeit"
	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/require"
)

const testMaxInt64 = int64(^uint64(0) >> 1)

func TestRetrieveObject(t *testing.T) {
	type repoMockFunc func(mc *minimock.Controller) repository.StorageRepository

	type args struct {
		ctx    context.Context
		blobID string
		offset int64
		length int64
	}

	var (
		ctx    = context.Background()
		blobID = gofakeit.UUID()

		blobContent = "test blob content"
		blobSize    = int64(len(blobContent))

		repoErr = errors.New("repo error")
	)

	tests := []struct {
		name     string
		args     args
		wantSize int64
		wantBody string
		err      error
		repoMock repoMockFunc
	}{
		{
			name: "success case",
			args: args{
				ctx:    ctx,
				blobID: blobID,
				offset: 0,
				length: 0,
			},
			wantSize: blobSize,
			wantBody: blobContent,
			err:      nil,
			repoMock: func(mc *minimock.Controller) repository.StorageRepository {
				mock := mocks.NewStorageRepositoryMock(mc)
				mock.RetrieveBlobMock.Expect(ctx, blobID).Return(io.NopCloser(strings.NewReader(blobContent)), blobSize, nil)
				return mock
			},
		},
		{
			name: "normalizes negative offset to zero and zero length to full read",
			args: args{
				ctx:    ctx,
				blobID: blobID,
				offset: -10,
				length: 0,
			},
			wantSize: blobSize,
			wantBody: blobContent,
			err:      nil,
			repoMock: func(mc *minimock.Controller) repository.StorageRepository {
				mock := mocks.NewStorageRepositoryMock(mc)
				mock.RetrieveBlobMock.Expect(ctx, blobID).Return(io.NopCloser(strings.NewReader(blobContent)), blobSize, nil)
				return mock
			},
		},
		{
			name: "partial range uses range repo method",
			args: args{
				ctx:    ctx,
				blobID: blobID,
				offset: 4,
				length: 5,
			},
			wantSize: blobSize,
			wantBody: " blob",
			err:      nil,
			repoMock: func(mc *minimock.Controller) repository.StorageRepository {
				mock := mocks.NewStorageRepositoryMock(mc)
				mock.RetrieveBlobRangeMock.Expect(ctx, blobID, int64(4), int64(5)).Return(io.NopCloser(strings.NewReader(" blob")), blobSize, nil)
				return mock
			},
		},
		{
			name: "zero length after non-zero offset reads to end",
			args: args{
				ctx:    ctx,
				blobID: blobID,
				offset: 4,
				length: 0,
			},
			wantSize: blobSize,
			wantBody: "blob content",
			err:      nil,
			repoMock: func(mc *minimock.Controller) repository.StorageRepository {
				mock := mocks.NewStorageRepositoryMock(mc)
				mock.RetrieveBlobRangeMock.Expect(ctx, blobID, int64(4), testMaxInt64).Return(io.NopCloser(strings.NewReader("blob content")), blobSize, nil)
				return mock
			},
		},
		{
			name: "repo error case",
			args: args{
				ctx:    ctx,
				blobID: blobID,
				offset: 0,
				length: 0,
			},
			wantSize: 0,
			err:      repoErr,
			repoMock: func(mc *minimock.Controller) repository.StorageRepository {
				mock := mocks.NewStorageRepositoryMock(mc)
				mock.RetrieveBlobMock.Expect(ctx, blobID).Return(nil, 0, repoErr)
				return mock
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := minimock.NewController(t)
			repoMock := tt.repoMock(mc)
			svc := storageService.NewService(repoMock)

			reader, size, err := svc.RetrieveObject(tt.args.ctx, tt.args.blobID, tt.args.offset, tt.args.length)
			require.Equal(t, tt.err, err)
			require.Equal(t, tt.wantSize, size)

			if reader != nil {
				body, readErr := io.ReadAll(reader)
				require.NoError(t, readErr)
				require.Equal(t, tt.wantBody, string(body))
				reader.Close()
			}
		})
	}
}
