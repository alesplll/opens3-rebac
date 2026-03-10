package user

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	sq "github.com/Masterminds/squirrel"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/client/db"
	domainerrors "github.com/alesplll/opens3-rebac/services/users/internal/errors/domain_errors"
	"github.com/jackc/pgconn"
)

func (r *repo) Update(ctx context.Context, id string, name, email *string) error {
	builder := sq.Update(usersTableName).PlaceholderFormat(squirrel.Dollar)

	if name != nil {
		builder = builder.Set(nameColumn, *name)
	}
	if email != nil {
		builder = builder.Set(emailColumn, *email)
	}
	if name == nil && email == nil {
		return fmt.Errorf("%w: %s", domainerrors.ErrInvalidInput, "no fields to update")
	}

	builder = builder.Set(updatedAtColumn, time.Now())
	builder = builder.Where(sq.Eq{idColumn: id})

	query, args, err := builder.ToSql()
	if err != nil {
		return err
	}

	q := db.Query{
		Name:     "user_repository:Update",
		QueryRaw: query,
	}

	result, err := r.db.DB().ExecContext(ctx, q, args...)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domainerrors.ErrEmailAlreadyExists
		}
		return err
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return domainerrors.ErrUserNotFound
	}

	return err
}
