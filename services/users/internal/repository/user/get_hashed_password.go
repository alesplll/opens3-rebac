package user

import (
	"context"
	"errors"

	sq "github.com/Masterminds/squirrel"
	"github.com/alesplll/opens3-rebac/shared/pkg/kit/client/db"
	domainerrors "github.com/alesplll/opens3-rebac/services/users/internal/errors/domain_errors"
	"github.com/jackc/pgx/v4"
)

func (r *repo) GetUserCredentials(ctx context.Context, email string) (string, string, error) {
	builder := sq.Select(idColumn, passwordColumn).
		PlaceholderFormat(sq.Dollar).
		From(usersTableName).
		Where(sq.Eq{emailColumn: email}).
		Limit(1)

	query, args, err := builder.ToSql()
	if err != nil {
		return "", "", err
	}

	q := db.Query{
		Name:     "user_repository:Get",
		QueryRaw: query,
	}

	var (
		hashedPassword string
		id             string
	)
	err = r.db.DB().QueryRowContext(ctx, q, args...).Scan(&id, &hashedPassword)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", domainerrors.ErrUserNotFound
		}
		return "", "", err
	}

	return id, hashedPassword, nil
}
