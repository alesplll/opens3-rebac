package circuitbreaker

import (
	"context"
	"time"

	"github.com/sony/gobreaker"
	"go.uber.org/zap"
)

type CircuitBreakerCfg interface {
	ServiceName() string
	MaxRequest() uint32
	Timeout() time.Duration
	FailureRate() float64
}

type Logger interface {
	Warn(ctx context.Context, msg string, fields ...zap.Field)
}

func NewCircuitBreaker(ctx context.Context, logger Logger, cfg CircuitBreakerCfg) *gobreaker.CircuitBreaker {
	return gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        cfg.ServiceName(),
		MaxRequests: cfg.MaxRequest(),
		Timeout:     cfg.Timeout(),
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRaitio := float64(counts.TotalFailures) / float64(counts.Requests)
			return failureRaitio >= cfg.FailureRate()
		},
		OnStateChange: func(name string, from, to gobreaker.State) {
			logger.Warn(ctx, "Curcuit breaker change state",
				zap.String("name", name),
				zap.String("from_state", from.String()),
				zap.String("to_state", to.String()),
			)
		},
	})
}
