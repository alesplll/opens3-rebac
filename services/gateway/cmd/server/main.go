package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/alesplll/opens3-rebac/services/gateway/internal/app"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/logger"
	"go.uber.org/zap"
)

func main() {
	configPath := flag.String("config-path", ".env", "path to config file")
	flag.Parse()
	a, err := app.NewApp(*configPath)
	if err != nil {
		log.Fatal(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := a.Run(ctx); err != nil {
		logger.Error(ctx, "Gateway stopped", zap.Error(err))
		os.Exit(1)
	}
}
