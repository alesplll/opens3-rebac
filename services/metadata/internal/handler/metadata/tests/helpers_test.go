package tests

import (
	metadatahandler "github.com/alesplll/opens3-rebac/services/metadata/internal/handler/metadata"
	"github.com/alesplll/opens3-rebac/services/metadata/internal/service"
	metadatav1 "github.com/alesplll/opens3-rebac/shared/pkg/go/metadata/v1"
)

func newHandler(bucketService service.BucketService, objectService service.ObjectService) metadatav1.MetadataServiceServer {
	return metadatahandler.NewHandler(bucketService, objectService)
}
