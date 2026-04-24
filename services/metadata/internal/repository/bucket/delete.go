package bucket

import (
	"context"
	"errors"

	sq "github.com/Masterminds/squirrel"
	domainerrors "github.com/alesplll/opens3-rebac/services/metadata/internal/errors/domain_errors"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/client/db"
	"github.com/jackc/pgx/v4"
)

func (r *repo) Delete(ctx context.Context, bucketID string) error {
	builder := sq.Delete(bucketsTable).
		PlaceholderFormat(sq.Dollar).
		Where(sq.Eq{idColumn: bucketID})

	query, args, err := builder.ToSql()
	if err != nil {
		return err
	}

	q := db.Query{
		Name:     "bucket_repository:Delete",
		QueryRaw: query,
	}

	tag, err := r.db.DB().ExecContext(ctx, q, args...)
	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
		return domainerrors.ErrBucketNotFound
	}

	return nil
}

func (r *repo) CountObjects(ctx context.Context, bucketID string) (int64, error) {
	builder := sq.Select("COUNT(*)").
		PlaceholderFormat(sq.Dollar).
		From(objectsTable).
		Where(sq.Eq{bucketIDColumn: bucketID})

	query, args, err := builder.ToSql()
	if err != nil {
		return 0, err
	}

	q := db.Query{
		Name:     "bucket_repository:CountObjects",
		QueryRaw: query,
	}

	var count int64
	err = r.db.DB().QueryRowContext(ctx, q, args...).Scan(&count)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, nil
		}
		return 0, err
	}

	return count, nil
}
