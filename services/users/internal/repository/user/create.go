package user

import (
	"context"
	"errors"
	"time"

	sq "github.com/Masterminds/squirrel"
	domainerrors "github.com/alesplll/opens3-rebac/services/users/internal/errors/domain_errors"
	model "github.com/alesplll/opens3-rebac/services/users/internal/model"
	"github.com/alesplll/opens3-rebac/services/users/internal/repository/user/conventer"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/client/db"
	"github.com/jackc/pgconn"
)

func (r *repo) Create(ctx context.Context, userInfo *model.UserInfo, hashedPassword string, createdAt time.Time) (string, error) {
	userInfoRepo := conventer.FromModelToRepoUserInfo(userInfo)
	builder := sq.Insert(usersTableName).
		PlaceholderFormat(sq.Dollar).
		Columns(nameColumn, emailColumn, passwordColumn, createdAtColumn).
		Values(userInfoRepo.Name, userInfoRepo.Email, hashedPassword, createdAt).
		Suffix("RETURNING id")

	query, args, err := builder.ToSql()
	if err != nil {
		return "", err
	}

	q := db.Query{
		Name:     "user_repository:Create",
		QueryRaw: query,
	}

	var userID string

	err = r.db.DB().QueryRowContext(ctx, q, args...).Scan(&userID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return "", domainerrors.ErrEmailAlreadyExists
		}
		return "", err
	}

	return userID, nil
}
