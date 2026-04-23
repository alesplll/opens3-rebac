package bucket

import (
	"context"
	"errors"

	sq "github.com/Masterminds/squirrel"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/client/db"
	"github.com/jackc/pgx/v4"
)

func (r *repo) Head(ctx context.Context, name string) (bool, string, string, error) {
	builder := sq.Select(idColumn, ownerIDColumn).
		PlaceholderFormat(sq.Dollar).
		From(bucketsTable).
		Where(sq.Eq{nameColumn: name}).
		Limit(1)

	query, args, err := builder.ToSql()
	if err != nil {
		return false, "", "", err
	}

	q := db.Query{
		Name:     "bucket_repository:Head",
		QueryRaw: query,
	}

	var bucketID, ownerID string
	err = r.db.DB().QueryRowContext(ctx, q, args...).Scan(&bucketID, &ownerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, "", "", nil
		}
		return false, "", "", err
	}

	return true, bucketID, ownerID, nil
}
