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
	"github.com/jackc/pgconn"
)

func (r *repo) Create(ctx context.Context, name, ownerID string) (*model.Bucket, error) {
	builder := sq.Insert(bucketsTable).
		PlaceholderFormat(sq.Dollar).
		Columns(nameColumn, ownerIDColumn).
		Values(name, ownerID).
		Suffix("RETURNING id, name, owner_id, created_at")

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, err
	}

	q := db.Query{
		Name:     "bucket_repository:Create",
		QueryRaw: query,
	}

	var b repoModel.Bucket
	err = r.db.DB().ScanOneContext(ctx, &b, q, args...)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, domainerrors.ErrBucketAlreadyExists
		}
		return nil, err
	}

	return converter.FromRepoToDomain(&b), nil
}
