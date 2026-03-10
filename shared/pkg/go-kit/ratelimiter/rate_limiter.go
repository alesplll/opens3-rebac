package ratelimiter

import (
	"context"
	"time"
)

type TokenBucketLimiter struct {
	tokenBucketCh chan struct{}
	interval      time.Duration
}

type RateLimiterConfig interface {
	Limit() int64
	Period() time.Duration
}

func NewTokenBucketLimiter(ctx context.Context, cfg RateLimiterConfig) *TokenBucketLimiter {
	replenishmentInterval := cfg.Period().Nanoseconds() / cfg.Limit()
	if replenishmentInterval < 1 {
		return nil
	}

	limiter := &TokenBucketLimiter{
		tokenBucketCh: make(chan struct{}, cfg.Limit()),
		interval:      time.Duration(replenishmentInterval),
	}

	for i := 0; i < int(cfg.Limit()); i++ {
		limiter.tokenBucketCh <- struct{}{}
	}

	go limiter.startPeriodicReplenishment(ctx)

	return limiter
}

func (l *TokenBucketLimiter) startPeriodicReplenishment(ctx context.Context) {
	ticker := time.NewTicker(l.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			select {
			case l.tokenBucketCh <- struct{}{}:
			default:
				// bucket full — skip to avoid blocking
			}
		}
	}
}

func (l *TokenBucketLimiter) Allow() bool {
	select {
	case <-l.tokenBucketCh:
		return true
	default:
		return false
	}
}
