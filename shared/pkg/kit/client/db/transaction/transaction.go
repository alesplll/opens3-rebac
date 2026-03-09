package transaction

import (
	"context"

	"github.com/alesplll/opens3-rebac/shared/pkg/kit/client/db"
	"github.com/alesplll/opens3-rebac/shared/pkg/kit/contextx/txctx"
	"github.com/jackc/pgx/v4"
	"github.com/pkg/errors"
)

type manager struct {
	db db.Transactor
}

// NewTransactionManager creates a new transaction manager that implements db.TxManager interface
func NewTransactionManager(db db.Transactor) db.TxManager {
	return &manager{
		db: db,
	}
}

// transaction is the main function that executes user-provided handler within a transaction
func (m *manager) transaction(ctx context.Context, opts pgx.TxOptions, fn db.Handler) (err error) {
	// If this is a nested transaction, skip initiating new transaction and execute handler
	tx, ok := txctx.ExtractTx(ctx)
	if ok {
		return fn(ctx)
	}

	// Start new transaction
	tx, err = m.db.BeginTx(ctx, opts)
	if err != nil {
		return errors.Wrap(err, "can't begin transaction")
	}

	// Put transaction into context
	ctx = txctx.InjectTx(ctx, tx)

	// Setup defer function for transaction rollback or commit
	defer func() {
		// Recover from panic
		if r := recover(); r != nil {
			err = errors.Errorf("panic recovered: %v", r)
		}

		// Rollback transaction if error occurred
		if err != nil {
			if errRollback := tx.Rollback(ctx); errRollback != nil {
				err = errors.Wrapf(err, "errRollback: %v", errRollback)
			}

			return
		}

		// If no errors, commit transaction
		if nil == err {
			err = tx.Commit(ctx)
			if err != nil {
				err = errors.Wrap(err, "tx commit failed")
			}
		}
	}()

	// Execute code inside transaction
	// If function fails, return error and defer function will rollback
	// otherwise transaction will be committed
	if err = fn(ctx); err != nil {
		err = errors.Wrap(err, "failed executing code inside transaction")
	}

	return err
}

func (m *manager) ReadCommitted(ctx context.Context, f db.Handler) error {
	txOpts := pgx.TxOptions{IsoLevel: pgx.ReadCommitted}
	return m.transaction(ctx, txOpts, f)
}
