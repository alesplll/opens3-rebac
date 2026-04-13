package app

import (
	"context"
	"flag"
	"net/http"
	"syscall"
	"time"

	"github.com/alesplll/opens3-rebac/services/gateway/internal/config"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/closer"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/logger"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/metric"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/tracing"
	"go.uber.org/zap"
)

const shutdownTimeout = 10 * time.Second

var configPath string

func init() {
	flag.StringVar(&configPath, "config-path", ".env", "path to config file")
}

type App struct {
	serviceProvider *serviceProvider
	httpServer      *http.Server
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
		a.initTracing,
		a.initHTTPServer,
	}

	for _, f := range inits {
		if err := f(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (a *App) initConfig(_ context.Context) error {
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
	} else if meterProvider != nil {
		closer.AddNamed("OTEL Metrics", meterProvider.Shutdown)
	}

	if err := metric.Init(ctx, config.AppConfig().Metrics); err != nil {
		logger.Error(ctx, "failed to init metrics", zap.Error(err))
		return err
	}

	return nil
}

func (a *App) initTracing(ctx context.Context) error {
	if err := tracing.InitTracer(ctx, config.AppConfig().Tracing); err != nil {
		return err
	}

	closer.AddNamed("tracer", tracing.ShutdownTracer)
	return nil
}

func (a *App) initHTTPServer(ctx context.Context) error {
	a.httpServer = &http.Server{
		Addr:         config.AppConfig().HTTP.Address(),
		Handler:      a.serviceProvider.HTTPHandler(ctx).Router(),
		ReadTimeout:  config.AppConfig().HTTP.ReadTimeout(),
		WriteTimeout: config.AppConfig().HTTP.WriteTimeout(),
		IdleTimeout:  config.AppConfig().HTTP.IdleTimeout(),
	}

	closer.AddNamed("http server", func(ctx context.Context) error {
		shutdownCtx, cancel := context.WithTimeout(ctx, config.AppConfig().HTTP.ShutdownTimeout())
		defer cancel()
		return a.httpServer.Shutdown(shutdownCtx)
	})

	return nil
}

func (a *App) Run() error {
	logger.Info(context.Background(), "http server listening", zap.String("address", config.AppConfig().HTTP.Address()))

	if err := a.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error(context.Background(), "http server failed", zap.Error(err))
		return err
	}

	logger.Info(context.Background(), "http server stopped gracefully")
	return nil
}
