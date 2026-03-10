package ratelimiter

import (
	"context"

	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/ratelimiter"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type RateLimiterInterceptor struct {
	rateLimiter *ratelimiter.TokenBucketLimiter
}

func NewRateLimiterInterceptor(ctx context.Context, cfg ratelimiter.RateLimiterConfig) *RateLimiterInterceptor {
	return &RateLimiterInterceptor{
		rateLimiter: ratelimiter.NewTokenBucketLimiter(ctx, cfg),
	}
}

func (r *RateLimiterInterceptor) Unary(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	if !r.rateLimiter.Allow() {
		return nil, status.Error(codes.ResourceExhausted, "to many requests")
	}

	return handler(ctx, req)
}
