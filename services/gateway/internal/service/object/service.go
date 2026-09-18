package object

import (
	"github.com/alesplll/opens3-rebac/services/gateway/internal/service"
	metadatav1 "github.com/alesplll/opens3-rebac/shared/pkg/go/metadata/v1"
	storagev1 "github.com/alesplll/opens3-rebac/shared/pkg/go/storage/v1"
)

type objectService struct {
	metadata metadatav1.MetadataServiceClient
	storage  storagev1.DataStorageServiceClient
}

func NewService(metadata metadatav1.MetadataServiceClient, storage storagev1.DataStorageServiceClient) service.ObjectService {
	return &objectService{metadata: metadata, storage: storage}
}
