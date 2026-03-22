package app

import (
	"context"

	"github.com/alesplll/opens3-rebac/services/storage/internal/config"
	storageHandler "github.com/alesplll/opens3-rebac/services/storage/internal/handler/storage"
	"github.com/alesplll/opens3-rebac/services/storage/internal/repository"
	storageRepository "github.com/alesplll/opens3-rebac/services/storage/internal/repository/storage"
	"github.com/alesplll/opens3-rebac/services/storage/internal/service"
	storageService "github.com/alesplll/opens3-rebac/services/storage/internal/service/storage"
	desc "github.com/alesplll/opens3-rebac/shared/pkg/storage/v1"
)

type serviceProvider struct {
	storageRepository repository.StorageRepository
	storageService    service.StorageService
	storageHandler    desc.DataStorageServiceServer
}

func newServiceProvider() *serviceProvider {
	return &serviceProvider{}
}

func (s *serviceProvider) StorageRepository(_ context.Context) repository.StorageRepository {
	if s.storageRepository == nil {
		s.storageRepository = storageRepository.NewRepository(config.AppConfig().Storage)
	}

	return s.storageRepository
}

func (s *serviceProvider) StorageService(ctx context.Context) service.StorageService {
	if s.storageService == nil {
		s.storageService = storageService.NewService(s.StorageRepository(ctx))
	}

	return s.storageService
}

func (s *serviceProvider) StorageHandler(ctx context.Context) desc.DataStorageServiceServer {
	if s.storageHandler == nil {
		s.storageHandler = storageHandler.NewHandler(s.StorageService(ctx))
	}

	return s.storageHandler
}
