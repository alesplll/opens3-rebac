package app

import (
	"context"

	"github.com/alesplll/opens3-rebac/services/gateway/internal/authentication"
	grpcclient "github.com/alesplll/opens3-rebac/services/gateway/internal/client/grpc"
	authclient "github.com/alesplll/opens3-rebac/services/gateway/internal/client/grpc/auth"
	authzclient "github.com/alesplll/opens3-rebac/services/gateway/internal/client/grpc/authz"
	metadataclient "github.com/alesplll/opens3-rebac/services/gateway/internal/client/grpc/metadata"
	storageclient "github.com/alesplll/opens3-rebac/services/gateway/internal/client/grpc/storage"
	usersclient "github.com/alesplll/opens3-rebac/services/gateway/internal/client/grpc/users"
	"github.com/alesplll/opens3-rebac/services/gateway/internal/config"
	httpgateway "github.com/alesplll/opens3-rebac/services/gateway/internal/handler/http/gateway"
	"github.com/alesplll/opens3-rebac/services/gateway/internal/service"
	authservice "github.com/alesplll/opens3-rebac/services/gateway/internal/service/auth"
	gatewayservice "github.com/alesplll/opens3-rebac/services/gateway/internal/service/gateway"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/closer"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/logger"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/tokens"
	jwtkit "github.com/alesplll/opens3-rebac/shared/pkg/go-kit/tokens/jwt"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/tracing"
	authv1 "github.com/alesplll/opens3-rebac/shared/pkg/go/auth/v1"
	authzv1 "github.com/alesplll/opens3-rebac/shared/pkg/go/authz/v1"
	metadatav1 "github.com/alesplll/opens3-rebac/shared/pkg/go/metadata/v1"
	storagev1 "github.com/alesplll/opens3-rebac/shared/pkg/go/storage/v1"
	userv1 "github.com/alesplll/opens3-rebac/shared/pkg/go/user/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type serviceProvider struct {
	authConn     *grpc.ClientConn
	authzConn    *grpc.ClientConn
	usersConn    *grpc.ClientConn
	metadataConn *grpc.ClientConn
	storageConn  *grpc.ClientConn

	authClient     grpcclient.AuthClient
	authzClient    grpcclient.AuthZClient
	usersClient    grpcclient.UsersClient
	metadataClient grpcclient.MetadataClient
	storageClient  grpcclient.StorageClient
	tokenVerifier  tokens.TokenVerifier
	authenticator  authentication.Service

	authService    service.AuthService
	gatewayService service.GatewayService
	httpHandler    *httpgateway.Handler
}

func newServiceProvider() *serviceProvider {
	return &serviceProvider{}
}

func (s *serviceProvider) AuthClient(ctx context.Context) grpcclient.AuthClient {
	if s.authClient == nil {
		conn := s.authConnOrFatal(ctx)
		s.authClient = authclient.NewClient(
			authv1.NewAuthV1Client(conn),
		)
	}

	return s.authClient
}

func (s *serviceProvider) AuthZClient(ctx context.Context) grpcclient.AuthZClient {
	if s.authzClient == nil {
		cfg := config.AppConfig()
		conn := s.authzConnOrFatal(ctx)
		s.authzClient = authzclient.NewClient(
			authzv1.NewPermissionServiceClient(conn),
			cfg.AuthZClient.Timeout(),
		)
	}

	return s.authzClient
}

func (s *serviceProvider) UsersClient(ctx context.Context) grpcclient.UsersClient {
	if s.usersClient == nil {
		conn := s.usersConnOrFatal(ctx)
		s.usersClient = usersclient.NewClient(
			userv1.NewUserV1Client(conn),
		)
	}

	return s.usersClient
}

func (s *serviceProvider) MetadataClient(ctx context.Context) grpcclient.MetadataClient {
	if s.metadataClient == nil {
		cfg := config.AppConfig()
		conn := s.metadataConnOrFatal(ctx)
		s.metadataClient = metadataclient.NewClient(
			metadatav1.NewMetadataServiceClient(conn),
			cfg.Metadata.Timeout(),
		)
	}

	return s.metadataClient
}

func (s *serviceProvider) StorageClient(ctx context.Context) grpcclient.StorageClient {
	if s.storageClient == nil {
		cfg := config.AppConfig()
		conn := s.storageConnOrFatal(ctx)
		s.storageClient = storageclient.NewClient(
			storagev1.NewDataStorageServiceClient(conn),
			cfg.Storage.Timeout(),
			cfg.Storage.StreamTimeout(),
		)
	}

	return s.storageClient
}

func (s *serviceProvider) AuthService(ctx context.Context) service.AuthService {
	if s.authService == nil {
		s.authService = authservice.NewService(
			s.AuthClient(ctx),
			s.UsersClient(ctx),
		)
	}

	return s.authService
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
		cfg := config.AppConfig()
		s.httpHandler = httpgateway.NewHandler(
			s.GatewayService(ctx),
			s.AuthService(ctx),
			cfg.HTTP.MaxUploadSizeBytes(),
			s.Authenticator(),
		)
	}

	return s.httpHandler
}

func (s *serviceProvider) TokenVerifier() tokens.TokenVerifier {
	if s.tokenVerifier == nil {
		s.tokenVerifier = jwtkit.NewJWTVerifier(config.AppConfig().JWT)
	}

	return s.tokenVerifier
}

func (s *serviceProvider) Authenticator() authentication.Service {
	if s.authenticator == nil {
		s.authenticator = authentication.NewService(s.TokenVerifier())
	}

	return s.authenticator
}

func (s *serviceProvider) authConnOrFatal(ctx context.Context) *grpc.ClientConn {
	if s.authConn == nil {
		cfg := config.AppConfig()
		s.authConn = s.mustDial(ctx, "auth", cfg.Auth.Address())
	}

	return s.authConn
}

func (s *serviceProvider) authzConnOrFatal(ctx context.Context) *grpc.ClientConn {
	if s.authzConn == nil {
		cfg := config.AppConfig()
		s.authzConn = s.mustDial(ctx, "authz", cfg.AuthZClient.Address())
	}

	return s.authzConn
}

func (s *serviceProvider) usersConnOrFatal(ctx context.Context) *grpc.ClientConn {
	if s.usersConn == nil {
		cfg := config.AppConfig()
		s.usersConn = s.mustDial(ctx, "users", cfg.Users.Address())
	}

	return s.usersConn
}

func (s *serviceProvider) metadataConnOrFatal(ctx context.Context) *grpc.ClientConn {
	if s.metadataConn == nil {
		cfg := config.AppConfig()
		s.metadataConn = s.mustDial(ctx, "metadata", cfg.Metadata.Address())
	}

	return s.metadataConn
}

func (s *serviceProvider) storageConnOrFatal(ctx context.Context) *grpc.ClientConn {
	if s.storageConn == nil {
		cfg := config.AppConfig()
		s.storageConn = s.mustDial(ctx, "storage", cfg.Storage.Address())
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
