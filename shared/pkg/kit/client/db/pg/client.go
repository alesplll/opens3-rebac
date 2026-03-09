package pg

import (
	"context"
	"time"

	"github.com/alesplll/opens3-rebac/shared/pkg/kit/client/db"
	"github.com/pkg/errors"

	"github.com/jackc/pgx/v4/pgxpool"
)

type pgClient struct {
	masterDBC db.DB
}

type PGConfig interface {
	DSN() string
	Timeout() time.Duration
	NeedLog() bool
}

func NewPGClient(ctx context.Context, logger Logger, cfg PGConfig) (db.Client, error) {
	dbc, err := pgxpool.Connect(ctx, cfg.DSN())
	if err != nil {
		return nil, errors.Errorf("failed to connect to db: %v", err.Error())
	}

	return &pgClient{
		masterDBC: NewDB(dbc, logger, cfg),
	}, nil
}

func (c *pgClient) DB() db.DB {
	return c.masterDBC
}

func (c *pgClient) Close() error {
	if c.masterDBC != nil {
		c.masterDBC.Close()
	}

	return nil
}
