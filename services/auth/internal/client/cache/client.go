package cache

import (
	"context"
	"time"
)

type CacheClient interface {
	HashSet(context.Context, string, any) error
	Set(context.Context, string, any) error
	HGetAll(context.Context, string) ([]any, error)
	Get(context.Context, string) (any, error)
	Expire(context.Context, string, time.Duration) error
	Incr(context.Context, string) (any, error)
	Del(context.Context, string) error
	Ping(context.Context) error

	Close(context.Context) error
}
