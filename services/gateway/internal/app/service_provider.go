package app

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/alesplll/opens3-rebac/services/gateway/internal/config"
	"github.com/alesplll/opens3-rebac/services/gateway/internal/handler/httpapi"
	"github.com/alesplll/opens3-rebac/services/gateway/internal/service"
	objectservice "github.com/alesplll/opens3-rebac/services/gateway/internal/service/object"
	metadatav1 "github.com/alesplll/opens3-rebac/shared/pkg/go/metadata/v1"
	storagev1 "github.com/alesplll/opens3-rebac/shared/pkg/go/storage/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type serviceProvider struct {
	config *config.Config

	metadataConn   *grpc.ClientConn
	storageConn    *grpc.ClientConn
	metadataClient metadatav1.MetadataServiceClient
	storageClient  storagev1.DataStorageServiceClient

	objectService service.ObjectService
	objectHandler http.Handler
}

func newServiceProvider(cfg *config.Config) *serviceProvider {
	return &serviceProvider{config: cfg}
}

func (s *serviceProvider) MetadataClient() (metadatav1.MetadataServiceClient, error) {
	if s.metadataClient == nil {
		conn, err := grpc.NewClient(s.config.Metadata.Address(), grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return nil, fmt.Errorf("create Metadata client: %w", err)
		}
		s.metadataConn = conn
		s.metadataClient = metadatav1.NewMetadataServiceClient(conn)
	}
	return s.metadataClient, nil
}

func (s *serviceProvider) StorageClient() (storagev1.DataStorageServiceClient, error) {
	if s.storageClient == nil {
		conn, err := grpc.NewClient(s.config.Storage.Address(), grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return nil, fmt.Errorf("create Storage client: %w", err)
		}
		s.storageConn = conn
		s.storageClient = storagev1.NewDataStorageServiceClient(conn)
	}
	return s.storageClient, nil
}

func (s *serviceProvider) ObjectService() (service.ObjectService, error) {
	if s.objectService == nil {
		metadata, err := s.MetadataClient()
		if err != nil {
			return nil, err
		}
		storage, err := s.StorageClient()
		if err != nil {
			return nil, err
		}
		s.objectService = objectservice.NewService(metadata, storage)
	}
	return s.objectService, nil
}

func (s *serviceProvider) ObjectHandler() (http.Handler, error) {
	if s.objectHandler == nil {
		objects, err := s.ObjectService()
		if err != nil {
			return nil, err
		}
		s.objectHandler = httpapi.NewHandler(objects)
	}
	return s.objectHandler, nil
}

func (s *serviceProvider) Close() error {
	var errs []error
	if s.storageConn != nil {
		if err := s.storageConn.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close Storage client: %w", err))
		}
	}
	if s.metadataConn != nil {
		if err := s.metadataConn.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close Metadata client: %w", err))
		}
	}
	return errors.Join(errs...)
}
