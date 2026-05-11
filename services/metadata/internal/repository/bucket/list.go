package bucket

import (
	"context"

	sq "github.com/Masterminds/squirrel"
	"github.com/alesplll/opens3-rebac/services/metadata/internal/model"
	repoModel "github.com/alesplll/opens3-rebac/services/metadata/internal/repository/bucket/model"
	"github.com/alesplll/opens3-rebac/services/metadata/internal/repository/bucket/converter"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/client/db"
)

func (r *repo) List(ctx context.Context, ownerID string) ([]*model.Bucket, error) {
	builder := sq.Select(idColumn, nameColumn, ownerIDColumn, createdAtColumn).
		PlaceholderFormat(sq.Dollar).
		From(bucketsTable).
		Where(sq.Eq{ownerIDColumn: ownerID}).
		OrderBy(createdAtColumn + " DESC")

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, err
	}

	q := db.Query{
		Name:     "bucket_repository:List",
		QueryRaw: query,
	}

	var rows []repoModel.Bucket
	if err := r.db.DB().ScanAllContext(ctx, &rows, q, args...); err != nil {
		return nil, err
	}

	buckets := make([]*model.Bucket, 0, len(rows))
	for i := range rows {
		buckets = append(buckets, converter.FromRepoToDomain(&rows[i]))
	}

	return buckets, nil
}
