package tests

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/alesplll/opens3-rebac/services/metadata/internal/repository"
	objectRepo "github.com/alesplll/opens3-rebac/services/metadata/internal/repository/object"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/client/db"
	pgclient "github.com/alesplll/opens3-rebac/shared/pkg/go-kit/client/db/pg"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/contextx/txctx"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"go.uber.org/zap"
)

type testLogger struct{}

func (testLogger) Debug(context.Context, string, ...zap.Field) {}

type testPGConfig struct {
	dsn string
}

func (c testPGConfig) DSN() string            { return c.dsn }
func (c testPGConfig) Timeout() time.Duration { return 5 * time.Second }
func (c testPGConfig) NeedLog() bool          { return false }

func newRepositoryTxContext(t *testing.T) (context.Context, db.Client, repositoryCleanup) {
	t.Helper()

	ctx := context.Background()
	dsn := os.Getenv("METADATA_TEST_PG_DSN")
	if dsn == "" {
		dsn = "host=localhost port=5433 dbname=metadata_db user=metadata_user password=metadata_password sslmode=disable"
	}

	client, err := pgclient.NewPGClient(ctx, testLogger{}, testPGConfig{dsn: dsn})
	if err != nil {
		t.Skipf("skipping repository integration test: cannot connect to postgres: %v", err)
	}

	t.Cleanup(func() {
		_ = client.Close()
	})

	if err := client.DB().Ping(ctx); err != nil {
		t.Skipf("skipping repository integration test: postgres is unavailable: %v", err)
	}

	tx, err := client.DB().BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		t.Fatalf("begin transaction: %v", err)
	}

	txCtx := txctx.InjectTx(ctx, tx)

	cleanup := func() {
		_ = tx.Rollback(ctx)
	}

	return txCtx, client, cleanup
}

type repositoryCleanup func()

func newRepository(t *testing.T) (context.Context, db.Client, repositoryCleanup, repository.ObjectRepository) {
	t.Helper()

	ctx, client, cleanup := newRepositoryTxContext(t)

	if err := createTempSchema(ctx, client); err != nil {
		cleanup()
		t.Fatalf("create temp schema: %v", err)
	}

	return ctx, client, cleanup, objectRepo.NewRepository(client)
}

func createTempSchema(ctx context.Context, client db.Client) error {
	statements := []string{
		`CREATE TEMP TABLE buckets (
			id TEXT PRIMARY KEY,
			name TEXT UNIQUE NOT NULL,
			owner_id TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
		`CREATE TEMP TABLE objects (
			id TEXT PRIMARY KEY,
			bucket_id TEXT NOT NULL,
			key TEXT NOT NULL,
			current_version_id TEXT NULL,
			pending_version_id TEXT NULL,
			status TEXT NOT NULL DEFAULT 'active'
		)`,
		`CREATE TEMP TABLE versions (
			id TEXT PRIMARY KEY,
			object_id TEXT NOT NULL,
			blob_id TEXT NULL,
			size_bytes BIGINT NOT NULL DEFAULT 0,
			etag TEXT NOT NULL DEFAULT '',
			content_type TEXT NOT NULL DEFAULT 'application/octet-stream',
			version_number BIGINT NOT NULL,
			kind TEXT NOT NULL DEFAULT 'blob',
			state TEXT NOT NULL DEFAULT 'pending',
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			committed_at TIMESTAMPTZ NULL,
			aborted_at TIMESTAMPTZ NULL
		)`,
	}

	for i, stmt := range statements {
		if _, err := client.DB().ExecContext(ctx, db.Query{
			Name:     fmt.Sprintf("repository_tests:create_temp_schema_%d", i),
			QueryRaw: stmt,
		}); err != nil {
			return err
		}
	}

	return nil
}

func execSQL(t *testing.T, ctx context.Context, client db.Client, queryName, rawSQL string, args ...any) pgconn.CommandTag {
	t.Helper()

	tag, err := client.DB().ExecContext(ctx, db.Query{
		Name:     queryName,
		QueryRaw: rawSQL,
	}, args...)
	if err != nil {
		t.Fatalf("%s: %v", queryName, err)
	}

	return tag
}
