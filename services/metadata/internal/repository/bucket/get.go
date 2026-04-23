package bucket

import (
	"context"
	"errors"

	sq "github.com/Masterminds/squirrel"
	domainerrors "github.com/alesplll/opens3-rebac/services/metadata/internal/errors/domain_errors"
	"github.com/alesplll/opens3-rebac/services/metadata/internal/model"
	repoModel "github.com/alesplll/opens3-rebac/services/metadata/internal/repository/bucket/model"
	"github.com/alesplll/opens3-rebac/services/metadata/internal/repository/bucket/converter"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/client/db"
	"github.com/jackc/pgx/v4"
)

func (r *repo) Get(ctx context.Context, name string) (*model.Bucket, error) {
	builder := sq.Select(idColumn, nameColumn, ownerIDColumn, createdAtColumn).
		PlaceholderFormat(sq.Dollar).
		From(bucketsTable).
		Where(sq.Eq{nameColumn: name}).
		Limit(1)

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, err
	}

	q := db.Query{
		Name:     "bucket_repository:Get",
		QueryRaw: query,
	}

	var b repoModel.Bucket
	err = r.db.DB().ScanOneContext(ctx, &b, q, args...)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.ErrBucketNotFound
		}
		return nil, err
	}

	return converter.FromRepoToDomain(&b), nil
}
