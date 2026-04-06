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

func TestHealthCheck(t *testing.T) {
	type repoMockFunc func(mc *minimock.Controller) repository.StorageRepository

	var (
		ctx = context.Background()

		repoErr = errors.New("health check failed")
	)

	tests := []struct {
		name     string
		want     bool
		err      error
		repoMock repoMockFunc
	}{
		{
			name: "healthy",
			want: true,
			err:  nil,
			repoMock: func(mc *minimock.Controller) repository.StorageRepository {
				mock := mocks.NewStorageRepositoryMock(mc)
				mock.HealthCheckMock.Expect(ctx).Return(nil)
				return mock
			},
		},
		{
			name: "unhealthy",
			want: false,
			err:  repoErr,
			repoMock: func(mc *minimock.Controller) repository.StorageRepository {
				mock := mocks.NewStorageRepositoryMock(mc)
				mock.HealthCheckMock.Expect(ctx).Return(repoErr)
				return mock
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := minimock.NewController(t)
			repoMock := tt.repoMock(mc)
			svc := storageService.NewService(repoMock)

			healthy, err := svc.HealthCheck(ctx, "storage")
			require.Equal(t, tt.err, err)
			require.Equal(t, tt.want, healthy)
		})
	}
}
