package app

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/alesplll/opens3-rebac/services/gateway/internal/config"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/logger"
	"go.uber.org/zap"
)

const shutdownTimeout = 10 * time.Second

type App struct {
	config          *config.Config
	serviceProvider *serviceProvider
	httpServer      *http.Server
}

func NewApp(configPath string) (*App, error) {
	a := &App{}
	if err := a.initDeps(configPath); err != nil {
		if a.serviceProvider != nil {
			err = errors.Join(err, a.serviceProvider.Close())
		}
		return nil, err
	}
	return a, nil
}

func (a *App) initDeps(configPath string) error {
	if err := a.initConfig(configPath); err != nil {
		return err
	}
	if err := a.initLogger(); err != nil {
		return err
	}
	a.initServiceProvider()
	return a.initHTTPServer()
}

func (a *App) initConfig(path string) error {
	cfg, err := config.Load(path)
	if err != nil {
		return err
	}
	a.config = cfg
	return nil
}

func (a *App) initLogger() error {
	return logger.Init(a.config.Logger)
}

func (a *App) initServiceProvider() {
	a.serviceProvider = newServiceProvider(a.config)
}

func (a *App) initHTTPServer() error {
	handler, err := a.serviceProvider.ObjectHandler()
	if err != nil {
		return err
	}
	a.httpServer = &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	return nil
}

func (a *App) Run(ctx context.Context) (runErr error) {
	defer func() { _ = logger.Sync() }()
	defer func() { runErr = errors.Join(runErr, a.serviceProvider.Close()) }()

	listener, err := net.Listen("tcp", a.config.HTTP.Address())
	if err != nil {
		return err
	}
	defer listener.Close()

	served := make(chan error, 1)
	go func() { served <- a.httpServer.Serve(listener) }()
	logger.Info(ctx, "Gateway listening", zap.String("address", listener.Addr().String()))

	select {
	case err := <-served:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		_ = a.httpServer.Close()
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := a.httpServer.Shutdown(shutdownCtx); err != nil {
			_ = a.httpServer.Close()
			return err
		}
		if err := <-served; !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		logger.Info(context.Background(), "Gateway stopped gracefully")
		return nil
	}
}
