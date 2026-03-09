package redis

import (
	"context"
	"time"

	"github.com/alesplll/opens3-rebac/services/auth/internal/client/cache"
	"github.com/alesplll/opens3-rebac/services/auth/internal/config"
	"github.com/alesplll/opens3-rebac/shared/pkg/kit/logger"
	"github.com/gomodule/redigo/redis"
	"go.uber.org/zap"
)

type handler func(ctx context.Context, conn redis.Conn) error

type client struct {
	pool *redis.Pool
}

func NewClient(pool *redis.Pool) cache.CacheClient {
	return &client{
		pool: pool,
	}
}

func (c *client) HashSet(ctx context.Context, key string, values any) error {
	err := c.execute(ctx, func(ctx context.Context, conn redis.Conn) error {
		_, redisErr := conn.Do("HSET", redis.Args{key}.AddFlat(values)...)
		return redisErr
	})
	return err
}

func (c *client) Set(ctx context.Context, key string, value any) error {
	err := c.execute(ctx, func(ctx context.Context, conn redis.Conn) error {
		_, redisErr := conn.Do("SET", redis.Args{key}.Add(value)...)
		return redisErr
	})
	return err
}

func (c *client) HGetAll(ctx context.Context, key string) ([]any, error) {
	var values []any
	err := c.execute(ctx, func(ctx context.Context, conn redis.Conn) error {
		var redisErr error
		values, redisErr = redis.Values(conn.Do("HGETALL", key))
		return redisErr
	})
	if err != nil {
		return nil, err
	}

	return values, nil
}

func (c *client) Get(ctx context.Context, key string) (any, error) {
	var value any
	err := c.execute(ctx, func(ctx context.Context, conn redis.Conn) error {
		var redisErr error
		value, redisErr = conn.Do("GET", key)
		return redisErr
	})
	if err != nil {
		return nil, err
	}

	return value, nil
}

func (c *client) Expire(ctx context.Context, key string, expiration time.Duration) error {
	err := c.execute(ctx, func(ctx context.Context, conn redis.Conn) error {
		_, redisErr := conn.Do("EXPIRE", key, int(expiration.Seconds()))
		return redisErr
	})
	return err
}

func (c *client) Incr(ctx context.Context, key string) (any, error) {
	var reply any
	err := c.execute(ctx, func(ctx context.Context, conn redis.Conn) error {
		var redisErr error
		reply, redisErr = conn.Do("INCR", key)
		return redisErr
	})
	if err != nil {
		return 0, err
	}
	return reply, err
}

func (c *client) Del(ctx context.Context, key string) error {
	err := c.execute(ctx, func(ctx context.Context, conn redis.Conn) error {
		_, redisErr := conn.Do("DEL", key)
		return redisErr
	})
	return err
}

func (c *client) Ping(ctx context.Context) error {
	err := c.execute(ctx, func(ctx context.Context, conn redis.Conn) error {
		_, redisErr := conn.Do("PING")
		return redisErr
	})

	return err
}

func (c *client) execute(ctx context.Context, handler handler) error {
	conn, err := c.getConnect(ctx)
	if err != nil {
		return err
	}
	defer func() {
		closeErr := conn.Close()
		if closeErr != nil {
			logger.Error(ctx, "failed to close redis connection with error", zap.Error(err))
		}
	}()

	return handler(ctx, conn)
}

func (c *client) getConnect(ctx context.Context) (redis.Conn, error) {
	getConnTimeoutCtx, cancel := context.WithTimeout(ctx, config.AppConfig().Redis.ConnTimeout())
	defer cancel()

	conn, err := c.pool.GetContext(getConnTimeoutCtx)
	if err != nil {
		logger.Error(ctx, "failed to get redis connection", zap.Error(err))

		_ = conn.Close()
		return nil, err
	}

	return conn, nil
}

func (c *client) Close(ctx context.Context) error {
	return c.pool.Close()
}
