package metadata

import (
	metadatav1 "github.com/alesplll/opens3-rebac/shared/pkg/go/metadata/v1"
)

type handler struct {
	metadatav1.UnimplementedMetadataServiceServer
}

func NewHandler() metadatav1.MetadataServiceServer {
	return &handler{}
}
