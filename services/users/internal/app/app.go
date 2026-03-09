package app

import (
	"context"
	"errors"
	"flag"
	"net"
	"net/http"
	"sync"
	"syscall"
	"time"

	"github.com/alesplll/opens3-rebac/services/users/internal/config"
	desc "github.com/alesplll/opens3-rebac/shared/pkg/user/v1"
	"github.com/alesplll/opens3-rebac/shared/pkg/kit/closer"
	"github.com/alesplll/opens3-rebac/shared/pkg/kit/logger"
	"github.com/alesplll/opens3-rebac/shared/pkg/kit/metric"
	metricsInterceptor "github.com/alesplll/opens3-rebac/shared/pkg/kit/middleware/metrics"
	rateLimiterInterceptor "github.com/alesplll/opens3-rebac/shared/pkg/kit/middleware/ratelimiter"
	validationInterceptor "github.com/alesplll/opens3-rebac/shared/pkg/kit/middleware/validation"
	"github.com/alesplll/opens3-rebac/shared/pkg/kit/tracing"
	grpcMiddleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

const (
	shutdownTimeout = 10 * time.Second
)

var configPath string

func init() {
	flag.StringVar(&configPath, "config-path", ".env", "path to config file")
}

type App struct {
	serviceProvider *serviceProvider
	grpcServer      *grpc.Server
	httpServer      *http.Server
}

func NewApp(ctx context.Context) (*App, error) {
	a := &App{}

	err := a.initDeps(ctx)
	if err != nil {
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
		a.initHTTPServer,
		a.initTracing,
	}

	for _, f := range inits {
		err := f(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func (a *App) initConfig(_ context.Context) error {
	err := config.Load(configPath)
	if err != nil {
		return err
	}

	return nil
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
		grpc.Creds(insecure.NewCredentials()),
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
	healthServer.SetServingStatus("user.v1.UserServer", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	reflection.Register(a.grpcServer)

	desc.RegisterUserV1Server(a.grpcServer, a.serviceProvider.UserHandler(ctx))

	return nil
}

func (a *App) initHTTPServer(ctx context.Context) error {
	mux := runtime.NewServeMux()

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	err := desc.RegisterUserV1HandlerFromEndpoint(ctx, mux, config.AppConfig().GRPC.Address(), opts)
	if err != nil {
		return err
	}

	a.httpServer = &http.Server{
		Addr:    config.AppConfig().HTTP.Address(),
		Handler: mux,
	}

	closer.AddNamed("HTTP server", func(ctx context.Context) error {
		return a.httpServer.Shutdown(ctx)
	})

	return nil
}

func (a *App) initTracing(ctx context.Context) error {
	err := tracing.InitTracer(ctx, config.AppConfig().Tracing)
	if err != nil {
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

	err = a.grpcServer.Serve(lis)
	if err != nil {
		return err
	}

	logger.Info(context.Background(), "GRPC server stopped gracefully")
	return nil
}

func (a *App) runHTTPServer() error {
	logger.Info(context.Background(), "HTTP server is starting listening and serving", zap.String("address", config.AppConfig().HTTP.Address()))
	err := a.httpServer.ListenAndServe()
	if err != nil {
		return err
	}
	logger.Info(context.Background(), "HTTP server stopped gracefully")
	return nil
}

func (a *App) Run() error {
	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()

		err := a.runGRPCServer()
		if err != nil {
			logger.Error(context.Background(), "fault grpc server", zap.Error(err))
		}
	}()

	go func() {
		defer wg.Done()

		err := a.runHTTPServer()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error(context.Background(), "fault http server", zap.Error(err))
		}
	}()

	wg.Wait()

	return nil
}
