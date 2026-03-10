package db

import (
	"context"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
)

// Handler defines a function type that executes within a database transaction.
// It takes a context.Context and returns an error. This is typically used to encapsulate
// database operations that need to be executed atomically as part of a transaction.
type Handler func(ctx context.Context) error

// TxManager is a transaction manager that executes a user-specified handler within a transaction
type TxManager interface {
	ReadCommitted(ctx context.Context, f Handler) error
}

// Transactor interface for working with transactions
type Transactor interface {
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
}

// Client defines an interface for database client operations
type Client interface {
	DB() DB       // Returns the database instance
	Close() error // Closes the database connection
}

// Query represents a database query with a name and raw SQL string
type Query struct {
	Name     string // Name of the query
	QueryRaw string // Raw SQL query string
}

// SQLExecer combines both NamedExecer and QueryExecer interfaces
type SQLExecer interface {
	NamedExecer
	QueryExecer
}

// NamedExecer defines methods for scanning query results into Go types
type NamedExecer interface {
	// ScanOneContext scans a single row into dest
	ScanOneContext(ctx context.Context, dest any, q Query, args ...any) error
	// ScanAllContext scans multiple rows into dest
	ScanAllContext(ctx context.Context, dest any, q Query, args ...any) error
}

// QueryExecer defines basic database query execution methods
type QueryExecer interface {
	// ExecContext executes a query that doesn't return rows
	ExecContext(ctx context.Context, q Query, args ...any) (pgconn.CommandTag, error)
	// QueryContext executes a query that returns multiple rows
	QueryContext(ctx context.Context, q Query, args ...any) (pgx.Rows, error)
	// QueryRowContext executes a query that returns a single row
	QueryRowContext(ctx context.Context, q Query, args ...any) pgx.Row
}

// Pinger defines a method to check database connectivity
type Pinger interface {
	// Ping checks if the database is reachable
	Ping(ctx context.Context) error
}

// DB combines SQL execution, ping capability and close functionality
type DB interface {
	SQLExecer
	Transactor
	Pinger
	Close() // Closes the database connection
}
