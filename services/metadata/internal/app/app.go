package app

import (
	"context"
	"flag"
	"net"
	"syscall"
	"time"

	"github.com/alesplll/opens3-rebac/services/metadata/internal/config"
	metadatav1 "github.com/alesplll/opens3-rebac/shared/pkg/go/metadata/v1"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/closer"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/logger"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/metric"
	metricsInterceptor "github.com/alesplll/opens3-rebac/shared/pkg/go-kit/middleware/metrics"
	rateLimiterInterceptor "github.com/alesplll/opens3-rebac/shared/pkg/go-kit/middleware/ratelimiter"
	validationInterceptor "github.com/alesplll/opens3-rebac/shared/pkg/go-kit/middleware/validation"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/tracing"
	grpcMiddleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

const shutdownTimeout = 10 * time.Second

var configPath string

func init() {
	flag.StringVar(&configPath, "config-path", ".env", "path to config file")
}

type App struct {
	serviceProvider *serviceProvider
	grpcServer      *grpc.Server
}

func NewApp(ctx context.Context) (*App, error) {
	a := &App{}

	if err := a.initDeps(ctx); err != nil {
		return nil, err
	}

	return a, nil
}

func (a *App) initDeps(ctx context.Context) error {
	inits := []func(context.Context) error{
		a.initConfig,
		a.initLogger,
		a.initCloser,
		a.initServiceProvider,
		a.initMetrics,
		a.initGRPCServer,
		a.initTracing,
	}

	for _, f := range inits {
		if err := f(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (a *App) initConfig(_ context.Context) error {
	flag.Parse()
	return config.Load(configPath)
}

func (a *App) initLogger(_ context.Context) error {
	return logger.Init(config.AppConfig().Logger)
}

func (a *App) initCloser(_ context.Context) error {
	closer.Configure(logger.Logger(), shutdownTimeout, syscall.SIGINT, syscall.SIGTERM)
	return nil
}

func (a *App) initServiceProvider(_ context.Context) error {
	a.serviceProvider = newServiceProvider()
	return nil
}

func (a *App) initMetrics(ctx context.Context) error {
	meterProvider, err := metric.InitOTELMetrics(config.AppConfig().Metrics)
	if err != nil {
		logger.Error(ctx, "failed to create meter provider", zap.Error(err))
	}

	closer.AddNamed("OTEL Metrics", meterProvider.Shutdown)

	if err := metric.Init(ctx, config.AppConfig().Metrics); err != nil {
		logger.Error(ctx, "failed to init metrics", zap.Error(err))
		return err
	}
	return nil
}

func (a *App) initGRPCServer(ctx context.Context) error {
	a.grpcServer = grpc.NewServer(
		grpc.UnaryInterceptor(
			grpcMiddleware.ChainUnaryServer(
				rateLimiterInterceptor.NewRateLimiterInterceptor(ctx, config.AppConfig().RateLimiter).Unary,
				metricsInterceptor.MetricsInterceptor,
				validationInterceptor.ErrorCodesUnaryInterceptor(logger.Logger()),
				tracing.UnaryServerInterceptor(config.AppConfig().Tracing.ServiceName()),
			),
		),
	)

	closer.AddNamed("GRPC server", func(ctx context.Context) error {
		a.grpcServer.GracefulStop()
		return nil
	})

	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(a.grpcServer, healthServer)
	healthServer.SetServingStatus("opens3.metadata.v1.MetadataService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	reflection.Register(a.grpcServer)

	metadatav1.RegisterMetadataServiceServer(a.grpcServer, a.serviceProvider.MetadataHandler(ctx))

	return nil
}

func (a *App) initTracing(ctx context.Context) error {
	if err := tracing.InitTracer(ctx, config.AppConfig().Tracing); err != nil {
		return err
	}

	closer.AddNamed("tracer", tracing.ShutdownTracer)
	return nil
}

func (a *App) runGRPCServer() error {
	lis, err := net.Listen("tcp", config.AppConfig().GRPC.Address())
	if err != nil {
		return err
	}

	logger.Info(context.Background(), "GRPC server listening", zap.String("address", config.AppConfig().GRPC.Address()))

	if err := a.grpcServer.Serve(lis); err != nil {
		return err
	}

	logger.Info(context.Background(), "GRPC server stopped gracefully")
	return nil
}

func (a *App) Run() error {
	if err := a.runGRPCServer(); err != nil {
		logger.Error(context.Background(), "fault grpc server", zap.Error(err))
		return err
	}

	return nil
}
