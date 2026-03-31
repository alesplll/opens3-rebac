package app

import (
	"context"

	"github.com/alesplll/opens3-rebac/services/auth/internal/client/cache"
	redis_client "github.com/alesplll/opens3-rebac/services/auth/internal/client/cache/redis"
	grpc_clients "github.com/alesplll/opens3-rebac/services/auth/internal/client/grpc"
	userClient "github.com/alesplll/opens3-rebac/services/auth/internal/client/grpc/user"
	"github.com/alesplll/opens3-rebac/services/auth/internal/config"
	handler_auth "github.com/alesplll/opens3-rebac/services/auth/internal/handler/auth"
	"github.com/alesplll/opens3-rebac/services/auth/internal/repository"
	"github.com/alesplll/opens3-rebac/services/auth/internal/repository/auth"
	"github.com/alesplll/opens3-rebac/services/auth/internal/service"
	service_auth "github.com/alesplll/opens3-rebac/services/auth/internal/service/auth"
	auth_v1 "github.com/alesplll/opens3-rebac/shared/pkg/go/auth/v1"
	desc_user "github.com/alesplll/opens3-rebac/shared/pkg/go/user/v1"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/closer"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/logger"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/tokens"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/tokens/jwt"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/tracing"
	"github.com/gomodule/redigo/redis"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	servicePemPath = "service.pem"
)

type serviceProvider struct {
	authHandler auth_v1.AuthV1Server

	authService    service.AuthService
	authRepository repository.AuthRepository
	cacheClient    cache.CacheClient

	tokenService tokens.TokenService

	redisPool  *redis.Pool
	userClient grpc_clients.UserClient
}

func newServiceProvider() *serviceProvider {
	return &serviceProvider{}
}

func (s *serviceProvider) RedisPool() *redis.Pool {
	if s.redisPool == nil {
		redisPool := &redis.Pool{
			MaxIdle:     int(config.AppConfig().Redis.MaxIdle()),
			IdleTimeout: config.AppConfig().Redis.IdleTimeout(),
			DialContext: func(ctx context.Context) (redis.Conn, error) {
				return redis.DialContext(ctx, "tcp", config.AppConfig().Redis.InternalAddress())
			},
		}

		closer.AddNamed("RedisPool", func(ctx context.Context) error {
			return redisPool.Close()
		})

		s.redisPool = redisPool
	}

	return s.redisPool
}

func (s *serviceProvider) CacheClient() cache.CacheClient {
	if s.cacheClient == nil {
		s.cacheClient = redis_client.NewClient(s.RedisPool())
	}

	return s.cacheClient
}

func (s *serviceProvider) AuthRepository() repository.AuthRepository {
	if s.authRepository == nil {
		s.authRepository = auth.NewRedisRepository(s.CacheClient())
	}

	return s.authRepository
}

func (s *serviceProvider) AuthService(ctx context.Context) service.AuthService {
	if s.authService == nil {
		s.authService = service_auth.NewService(s.UserClient(ctx), s.TokenService(ctx), s.AuthRepository())
	}
	return s.authService
}

func (s *serviceProvider) AuthHandler(ctx context.Context) auth_v1.AuthV1Server {
	if s.authHandler == nil {
		s.authHandler = handler_auth.NewHandler(s.AuthService(ctx))
	}
	return s.authHandler
}

func (s *serviceProvider) TokenService(ctx context.Context) tokens.TokenService {
	if s.tokenService == nil {
		s.tokenService = jwt.NewJWTService(config.AppConfig().JWT)
	}
	return s.tokenService
}

func (s *serviceProvider) UserClient(ctx context.Context) grpc_clients.UserClient {
	if s.userClient == nil {
		conn, err := grpc.NewClient(
			config.AppConfig().UserClient.Address(),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithUnaryInterceptor(
				tracing.UnaryClientInterceptor("user-server-client"),
			),
		)
		if err != nil {
			logger.Fatal(ctx, "failed to create userClient connection", zap.Error(err))
		}

		closer.AddNamed("UserClientGRPC", func(ctx context.Context) error {
			return conn.Close()
		})

		logger.Debug(ctx, "Succesfully create UserServer client", zap.Any("connection", conn))
		protoClient := desc_user.NewUserV1Client(conn)
		s.userClient = userClient.NewClient(protoClient)
	}

	return s.userClient
}
