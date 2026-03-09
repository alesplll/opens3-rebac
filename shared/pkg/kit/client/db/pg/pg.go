package pg

import (
	"context"
	"time"

	"github.com/alesplll/opens3-rebac/shared/pkg/kit/client/db"
	"github.com/alesplll/opens3-rebac/shared/pkg/kit/client/db/prettier"
	"github.com/alesplll/opens3-rebac/shared/pkg/kit/contextx/txctx"
	"github.com/georgysavva/scany/pgxscan"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"go.uber.org/zap"
)

type Logger interface {
	Debug(ctx context.Context, msg string, fields ...zap.Field)
}

type pg struct {
	dbc *pgxpool.Pool
	l   Logger
	cfg PGConfig
}

func NewDB(dbc *pgxpool.Pool, logger Logger, cfg PGConfig) db.DB {
	return &pg{
		dbc: dbc,
		l:   logger,
		cfg: cfg,
	}
}

func (p *pg) withOpTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if p.cfg == nil || p.cfg.Timeout() <= 0 {
		return ctx, func() {}
	}
	if dl, ok := ctx.Deadline(); ok {
		if time.Until(dl) <= p.cfg.Timeout() {
			return ctx, func() {}
		}
	}
	return context.WithTimeout(ctx, p.cfg.Timeout())
}

func (p *pg) ScanOneContext(ctx context.Context, dest any, q db.Query, args ...any) error {
	row, err := p.QueryContext(ctx, q, args...)
	if err != nil {
		return err
	}

	return pgxscan.ScanOne(dest, row)
}

func (p *pg) ScanAllContext(ctx context.Context, dest any, q db.Query, args ...any) error {
	rows, err := p.QueryContext(ctx, q, args...)
	if err != nil {
		return err
	}

	return pgxscan.ScanAll(dest, rows)
}

func (p *pg) ExecContext(ctx context.Context, q db.Query, args ...any) (pgconn.CommandTag, error) {
	ctx, cancel := p.withOpTimeout(ctx)
	defer cancel()

	p.logQuery(ctx, q, args...)

	tx, ok := txctx.ExtractTx(ctx)
	if ok {
		return tx.Exec(ctx, q.QueryRaw, args...)
	}

	return p.dbc.Exec(ctx, q.QueryRaw, args...)
}

func (p *pg) QueryContext(ctx context.Context, q db.Query, args ...any) (pgx.Rows, error) {
	ctx, cancel := p.withOpTimeout(ctx)
	defer cancel()

	p.logQuery(ctx, q, args...)

	tx, ok := txctx.ExtractTx(ctx)
	if ok {
		return tx.Query(ctx, q.QueryRaw, args...)
	}

	return p.dbc.Query(ctx, q.QueryRaw, args...)
}

func (p *pg) QueryRowContext(ctx context.Context, q db.Query, args ...any) pgx.Row {
	p.logQuery(ctx, q, args...)

	tx, ok := txctx.ExtractTx(ctx)
	if ok {
		return tx.QueryRow(ctx, q.QueryRaw, args...)
	}

	return p.dbc.QueryRow(ctx, q.QueryRaw, args...)
}

func (p *pg) Ping(ctx context.Context) error {
	ctx, cancel := p.withOpTimeout(ctx)
	defer cancel()

	return p.dbc.QueryRow(ctx, "SELECT 1").Scan(new(int))
}

func (p *pg) BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error) {
	ctx, cancel := p.withOpTimeout(ctx)
	defer cancel()

	return p.dbc.BeginTx(ctx, txOptions)
}

func (p *pg) Close() {
	p.dbc.Close()
}

func (p *pg) logQuery(ctx context.Context, q db.Query, args ...any) {
	if !p.cfg.NeedLog() {
		return
	}

	_, inTx := txctx.ExtractTx(ctx)
	prettyQuery := prettier.Pretty(q.QueryRaw, prettier.PlaceholderDollar, args...)
	p.l.Debug(ctx, "PG Query",
		zap.String("name", q.Name),
		zap.Bool("in_tx", inTx),
		zap.Int("args_len", len(args)),
		zap.String("query", prettyQuery),
	)
}
