package object

import (
	"context"

	sq "github.com/Masterminds/squirrel"
	"github.com/alesplll/opens3-rebac/services/metadata/internal/model"
	repoModel "github.com/alesplll/opens3-rebac/services/metadata/internal/repository/object/model"
	"github.com/alesplll/opens3-rebac/services/metadata/internal/repository/object/converter"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/client/db"
)

const defaultMaxKeys = 1000

func (r *repo) List(ctx context.Context, bucketName, prefix, continuationToken string, maxKeys int32) ([]*model.ObjectListItem, string, bool, error) {
	if maxKeys <= 0 || maxKeys > defaultMaxKeys {
		maxKeys = defaultMaxKeys
	}

	builder := sq.Select(
		"o.id AS object_id",
		"v.id AS version_id",
		"o.key",
		"v.etag",
		"v.size_bytes",
		"v.content_type",
		"v.created_at AS last_modified",
	).
		PlaceholderFormat(sq.Dollar).
		From("objects o").
		Join("buckets b ON o.bucket_id = b.id").
		Join("versions v ON v.id = o.current_version_id").
		Where(sq.Eq{"b.name": bucketName}).
		Where(sq.Eq{"v.is_deleted": false}).
		OrderBy("o.key ASC").
		Limit(uint64(maxKeys))

	if prefix != "" {
		builder = builder.Where(sq.Like{"o.key": prefix + "%"})
	}

	if continuationToken != "" {
		builder = builder.Where(sq.Gt{"o.key": continuationToken})
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, "", false, err
	}

	q := db.Query{
		Name:     "object_repository:List",
		QueryRaw: query,
	}

	var rows []repoModel.ObjectListItem
	if err := r.db.DB().ScanAllContext(ctx, &rows, q, args...); err != nil {
		return nil, "", false, err
	}

	items := make([]*model.ObjectListItem, 0, len(rows))
	for i := range rows {
		items = append(items, converter.ListItemToDomain(&rows[i]))
	}

	var nextToken string
	isTruncated := len(rows) == int(maxKeys)
	if isTruncated {
		nextToken = rows[len(rows)-1].Key
	}

	return items, nextToken, isTruncated, nil
}
