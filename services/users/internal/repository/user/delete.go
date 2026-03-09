package user

import (
	"context"

	"github.com/Masterminds/squirrel"
	sq "github.com/Masterminds/squirrel"
	domainerrors "github.com/alesplll/opens3-rebac/services/users/internal/errors/domain_errors"
	"github.com/alesplll/opens3-rebac/shared/pkg/kit/client/db"
)

func (r *repo) Delete(ctx context.Context, id string) error {
	builder := sq.Delete(usersTableName).
		PlaceholderFormat(squirrel.Dollar).
		Where(sq.Eq{"id": id})

	query, args, err := builder.ToSql()
	if err != nil {
		return err
	}

	q := db.Query{
		Name:     "user_repository:Delete",
		QueryRaw: query,
	}

	result, err := r.db.DB().ExecContext(ctx, q, args...)
	if err != nil {
		return err
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return domainerrors.ErrUserNotFound
	}

	return nil
}
