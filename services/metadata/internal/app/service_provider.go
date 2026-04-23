package app

import (
	"context"

	metadataHandler "github.com/alesplll/opens3-rebac/services/metadata/internal/handler/metadata"
	metadatav1 "github.com/alesplll/opens3-rebac/shared/pkg/go/metadata/v1"
)

type serviceProvider struct {
	metadataHandler metadatav1.MetadataServiceServer
}

func newServiceProvider() *serviceProvider {
	return &serviceProvider{}
}

func (s *serviceProvider) MetadataHandler(_ context.Context) metadatav1.MetadataServiceServer {
	if s.metadataHandler == nil {
		s.metadataHandler = metadataHandler.NewHandler()
	}

	return s.metadataHandler
}
