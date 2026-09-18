package app

import (
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
	metadataConn  *grpc.ClientConn
	storageConn   *grpc.ClientConn
	objectService service.ObjectService
	objectHandler http.Handler
}

func newServiceProvider(cfg *config.Config) (*serviceProvider, error) {
	metadataConn, err := grpc.NewClient(cfg.Metadata.Address(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("metadata client: %w", err)
	}
	storageConn, err := grpc.NewClient(cfg.Storage.Address(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		_ = metadataConn.Close()
		return nil, fmt.Errorf("storage client: %w", err)
	}
	return &serviceProvider{metadataConn: metadataConn, storageConn: storageConn}, nil
}

func (s *serviceProvider) ObjectService() service.ObjectService {
	if s.objectService == nil {
		s.objectService = objectservice.NewService(
			metadatav1.NewMetadataServiceClient(s.metadataConn),
			storagev1.NewDataStorageServiceClient(s.storageConn),
		)
	}
	return s.objectService
}

func (s *serviceProvider) ObjectHandler() http.Handler {
	if s.objectHandler == nil {
		s.objectHandler = httpapi.NewHandler(s.ObjectService())
	}
	return s.objectHandler
}

func (s *serviceProvider) Close() {
	_ = s.storageConn.Close()
	_ = s.metadataConn.Close()
}
