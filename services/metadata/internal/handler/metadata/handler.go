package metadata

import (
	"github.com/alesplll/opens3-rebac/services/metadata/internal/service"
	metadatav1 "github.com/alesplll/opens3-rebac/shared/pkg/go/metadata/v1"
)

type handler struct {
	metadatav1.UnimplementedMetadataServiceServer
	bucketService service.BucketService
	objectService service.ObjectService
}

func NewHandler(bucketService service.BucketService, objectService service.ObjectService) metadatav1.MetadataServiceServer {
	return &handler{
		bucketService: bucketService,
		objectService: objectService,
	}
}
