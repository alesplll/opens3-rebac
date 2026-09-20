package storage

import (
	"github.com/alesplll/opens3-rebac/services/storage/internal/service"
	desc "github.com/alesplll/opens3-rebac/shared/pkg/go/storage/v1"
)

type handler struct {
	desc.UnimplementedDataStorageServiceServer
	service                service.StorageService
	retrieveChunkSizeBytes int
}

func NewHandler(service service.StorageService, retrieveChunkSizeBytes int) desc.DataStorageServiceServer {
	return &handler{
		service:                service,
		retrieveChunkSizeBytes: retrieveChunkSizeBytes,
	}
}
