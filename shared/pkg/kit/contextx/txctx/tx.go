package txctx

import (
	"context"

	"github.com/alesplll/opens3-rebac/shared/pkg/kit/contextx"
	"github.com/jackc/pgx/v4"
)

const TxKey contextx.CtxKey = "tx"

func InjectTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, TxKey, tx)
}

func ExtractTx(ctx context.Context) (pgx.Tx, bool) {
	if tx, ok := ctx.Value(TxKey).(pgx.Tx); ok {
		return tx, true
	}
	return nil, false
}
