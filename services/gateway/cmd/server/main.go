package main

import (
	"context"

	"github.com/alesplll/opens3-rebac/services/gateway/internal/app"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/logger"
	"go.uber.org/zap"
)

func main() {
	appCtx := context.Background()

	a, err := app.NewApp(appCtx)
	if err != nil {
		logger.Fatal(appCtx, "failed to init app", zap.Error(err))
	}

	if err := a.Run(); err != nil {
		logger.Fatal(appCtx, "failed to run app", zap.Error(err))
	}
}
