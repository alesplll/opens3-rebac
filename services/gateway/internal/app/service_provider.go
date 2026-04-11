package app

import (
	"context"

	authzclient "github.com/alesplll/opens3-rebac/services/gateway/internal/client/grpc/authz"
	grpcclient "github.com/alesplll/opens3-rebac/services/gateway/internal/client/grpc"
	metadataclient "github.com/alesplll/opens3-rebac/services/gateway/internal/client/grpc/metadata"
	storageclient "github.com/alesplll/opens3-rebac/services/gateway/internal/client/grpc/storage"
	"github.com/alesplll/opens3-rebac/services/gateway/internal/config"
	httpgateway "github.com/alesplll/opens3-rebac/services/gateway/internal/handler/http/gateway"
	"github.com/alesplll/opens3-rebac/services/gateway/internal/service"
	gatewayservice "github.com/alesplll/opens3-rebac/services/gateway/internal/service/gateway"
	authzv1 "github.com/alesplll/opens3-rebac/shared/pkg/go/authz/v1"
	metadatav1 "github.com/alesplll/opens3-rebac/shared/pkg/go/metadata/v1"
	storagev1 "github.com/alesplll/opens3-rebac/shared/pkg/go/storage/v1"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/closer"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/logger"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/tokens"
	jwtkit "github.com/alesplll/opens3-rebac/shared/pkg/go-kit/tokens/jwt"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/tracing"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type serviceProvider struct {
	authzConn    *grpc.ClientConn
	metadataConn *grpc.ClientConn
	storageConn  *grpc.ClientConn

	authzClient    grpcclient.AuthZClient
	metadataClient grpcclient.MetadataClient
	storageClient  grpcclient.StorageClient
	tokenVerifier  tokens.TokenVerifier

	gatewayService service.GatewayService
	httpHandler    *httpgateway.Handler
}

func newServiceProvider() *serviceProvider {
	return &serviceProvider{}
}

func (s *serviceProvider) AuthZClient(ctx context.Context) grpcclient.AuthZClient {
	if s.authzClient == nil {
		conn := s.authzConnOrPanic(ctx)
		s.authzClient = authzclient.NewClient(
			authzv1.NewPermissionServiceClient(conn),
			config.AppConfig().AuthZClient.Timeout(),
		)
	}

	return s.authzClient
}

func (s *serviceProvider) MetadataClient(ctx context.Context) grpcclient.MetadataClient {
	if s.metadataClient == nil {
		conn := s.metadataConnOrPanic(ctx)
		s.metadataClient = metadataclient.NewClient(
			metadatav1.NewMetadataServiceClient(conn),
			config.AppConfig().Metadata.Timeout(),
		)
	}

	return s.metadataClient
}

func (s *serviceProvider) StorageClient(ctx context.Context) grpcclient.StorageClient {
	if s.storageClient == nil {
		conn := s.storageConnOrPanic(ctx)
		s.storageClient = storageclient.NewClient(
			storagev1.NewDataStorageServiceClient(conn),
			config.AppConfig().Storage.Timeout(),
			config.AppConfig().Storage.StreamTimeout(),
		)
	}

	return s.storageClient
}

func (s *serviceProvider) GatewayService(ctx context.Context) service.GatewayService {
	if s.gatewayService == nil {
		s.gatewayService = gatewayservice.NewService(
			s.AuthZClient(ctx),
			s.MetadataClient(ctx),
			s.StorageClient(ctx),
		)
	}

	return s.gatewayService
}

func (s *serviceProvider) HTTPHandler(ctx context.Context) *httpgateway.Handler {
	if s.httpHandler == nil {
		s.httpHandler = httpgateway.NewHandler(
			s.GatewayService(ctx),
			config.AppConfig().HTTP.MaxUploadSizeBytes(),
			s.TokenVerifier(),
		)
	}

	return s.httpHandler
}

func (s *serviceProvider) TokenVerifier() tokens.TokenVerifier {
	if s.tokenVerifier == nil {
		s.tokenVerifier = jwtkit.NewJWTVerifier(config.NewJWTVerifierConfig(config.AppConfig().JWT))
	}

	return s.tokenVerifier
}

func (s *serviceProvider) authzConnOrPanic(ctx context.Context) *grpc.ClientConn {
	if s.authzConn == nil {
		s.authzConn = s.mustDial(ctx, "authz", config.AppConfig().AuthZClient.Address())
	}

	return s.authzConn
}

func (s *serviceProvider) metadataConnOrPanic(ctx context.Context) *grpc.ClientConn {
	if s.metadataConn == nil {
		s.metadataConn = s.mustDial(ctx, "metadata", config.AppConfig().Metadata.Address())
	}

	return s.metadataConn
}

func (s *serviceProvider) storageConnOrPanic(ctx context.Context) *grpc.ClientConn {
	if s.storageConn == nil {
		s.storageConn = s.mustDial(ctx, "storage", config.AppConfig().Storage.Address())
	}

	return s.storageConn
}

func (s *serviceProvider) mustDial(ctx context.Context, name, address string) *grpc.ClientConn {
	conn, err := grpc.NewClient(
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(tracing.UnaryClientInterceptor(name+"-client")),
	)
	if err != nil {
		logger.Fatal(ctx, "failed to create grpc client connection", zap.String("client", name), zap.String("address", address), zap.Error(err))
	}

	closer.AddNamed(name+" grpc client", func(context.Context) error {
		return conn.Close()
	})

	logger.Info(ctx, "grpc client connected", zap.String("client", name), zap.String("address", address))

	return conn
}
