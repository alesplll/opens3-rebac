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

func NewApp(cfg *config.Config) (*App, error) {
	a := &App{config: cfg}
	if err := a.initDeps(); err != nil {
		return nil, err
	}
	return a, nil
}

func (a *App) initDeps() error {
	if err := logger.Init(a.config.Logger); err != nil {
		return err
	}
	provider, err := newServiceProvider(a.config)
	if err != nil {
		return err
	}
	a.serviceProvider = provider
	a.httpServer = &http.Server{
		Handler:           provider.ObjectHandler(),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	return nil
}

func (a *App) Run(ctx context.Context) error {
	defer a.serviceProvider.Close()
	defer func() { _ = logger.Sync() }()
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
