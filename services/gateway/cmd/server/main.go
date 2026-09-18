package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/alesplll/opens3-rebac/services/gateway/internal/app"
	"github.com/alesplll/opens3-rebac/services/gateway/internal/config"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := app.Run(ctx, config.Load()); err != nil {
		slog.Error("Gateway stopped", "error", err)
		os.Exit(1)
	}
}
