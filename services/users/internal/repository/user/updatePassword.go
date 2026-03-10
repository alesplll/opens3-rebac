package user

import (
	"context"
	"time"

	"github.com/Masterminds/squirrel"
	sq "github.com/Masterminds/squirrel"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/client/db"
	domainerrors "github.com/alesplll/opens3-rebac/services/users/internal/errors/domain_errors"
)

func (r *repo) UpdatePassword(ctx context.Context, id string, hashedPassword string) error {
	builder := sq.Update(usersTableName).
		PlaceholderFormat(squirrel.Dollar).
		Set(passwordColumn, hashedPassword).
		Set(updatedAtColumn, time.Now()).
		Where(sq.Eq{idColumn: id})

	query, args, err := builder.ToSql()
	if err != nil {
		return err
	}

	q := db.Query{
		Name:     "user_repository:UpdatePassword",
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
	return err
}

func (r *repo) LogPassword(ctx context.Context, id string, ip_address string) error {
	builder := sq.Insert(passwordLogsTableName).
		PlaceholderFormat(squirrel.Dollar).
		Columns(passwordLogsUserIdColumn, passwordLogsIpAddressColumn).
		Values(id, ip_address)

	query, args, err := builder.ToSql()
	if err != nil {
		return err
	}

	q := db.Query{
		Name:     "user_repository:LogPassword",
		QueryRaw: query,
	}

	_, err = r.db.DB().ExecContext(ctx, q, args...)
	if err != nil {
		return err
	}

	return nil
}
