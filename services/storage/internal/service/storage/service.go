package storage

import (
	"github.com/alesplll/opens3-rebac/services/storage/internal/repository"
	"github.com/alesplll/opens3-rebac/services/storage/internal/service"
)

type storageService struct {
	repo repository.StorageRepository
}

func NewService(repo repository.StorageRepository) service.StorageService {
	return &storageService{
		repo: repo,
	}
}
