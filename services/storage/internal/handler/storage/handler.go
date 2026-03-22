package storage

import (
	"github.com/alesplll/opens3-rebac/services/storage/internal/service"
	desc "github.com/alesplll/opens3-rebac/shared/pkg/storage/v1"
)

type handler struct {
	desc.UnimplementedDataStorageServiceServer
	service service.StorageService
}

func NewHandler(service service.StorageService) desc.DataStorageServiceServer {
	return &handler{
		service: service,
	}
}
