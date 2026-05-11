package postgres

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/alesplll/opens3-rebac/e2e/suite/config"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type Execer interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

type QueryRower interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func MustConnect(tb testing.TB, ctx context.Context, cfg config.PGConfig) *pgxpool.Pool {
	tb.Helper()

	pool, err := pgxpool.Connect(ctx, cfg.DSN())
	if err != nil {
		tb.Fatalf("connect postgres: %v", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		tb.Fatalf("ping postgres: %v", err)
	}

	return pool
}

func MustResetPublicSchema(tb testing.TB, ctx context.Context, execer Execer) {
	tb.Helper()

	MustExec(tb, ctx, execer, `DROP SCHEMA IF EXISTS public CASCADE`)
	MustExec(tb, ctx, execer, `CREATE SCHEMA public`)
}

func MustApplyMigrations(tb testing.TB, ctx context.Context, execer Execer, migrationDir string) {
	tb.Helper()

	pattern := filepath.Join(migrationDir, "*.sql")
	files, err := filepath.Glob(pattern)
	if err != nil {
		tb.Fatalf("glob migrations %q: %v", pattern, err)
	}

	sort.Strings(files)
	if len(files) == 0 {
		tb.Fatalf("no migration files found in %q", migrationDir)
	}

	for _, fileName := range files {
		statements, err := migrationUpStatements(fileName)
		if err != nil {
			tb.Fatalf("read migration %q: %v", fileName, err)
		}

		for _, stmt := range statements {
			MustExec(tb, ctx, execer, stmt)
		}
	}
}

func MustFullCleanup(tb testing.TB, ctx context.Context, execer Execer, tables ...string) {
	tb.Helper()

	if len(tables) == 0 {
		tb.Fatalf("full cleanup requires at least one table")
	}

	MustExec(tb, ctx, execer, fmt.Sprintf("TRUNCATE TABLE %s RESTART IDENTITY CASCADE", strings.Join(tables, ", ")))
}

func MustExec(tb testing.TB, ctx context.Context, execer Execer, query string, args ...any) {
	tb.Helper()

	if _, err := execer.Exec(ctx, query, args...); err != nil {
		tb.Fatalf("exec query %q: %v", query, err)
	}
}

func MustQueryRow(tb testing.TB, ctx context.Context, queryRower QueryRower, query string, args ...any) pgx.Row {
	tb.Helper()

	return queryRower.QueryRow(ctx, query, args...)
}

func migrationUpStatements(fileName string) ([]string, error) {
	migrationSQL, err := os.ReadFile(fileName)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	upSQL, _, found := strings.Cut(string(migrationSQL), "-- +goose Down")
	if !found {
		return nil, fmt.Errorf("missing goose down marker")
	}

	upSQL = strings.TrimSpace(strings.TrimPrefix(upSQL, "-- +goose Up"))
	rawStatements := strings.Split(upSQL, ";\n")
	statements := make([]string, 0, len(rawStatements))
	for _, stmt := range rawStatements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		statements = append(statements, stmt)
	}

	return statements, nil
}
