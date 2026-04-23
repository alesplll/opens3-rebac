package object

import (
	"context"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/client/db"
)

// InsertVersion adds a new version record for an existing object.
// Used inside CreateObjectVersion transaction after UpsertObject.
func (r *repo) InsertVersion(ctx context.Context, objectID, blobID string, sizeBytes int64, etag, contentType string) (string, time.Time, error) {
	builder := sq.Insert(versionsTable).
		PlaceholderFormat(sq.Dollar).
		Columns("object_id", "blob_id", "size_bytes", "etag", "content_type").
		Values(objectID, blobID, sizeBytes, etag, contentType).
		Suffix("RETURNING id, created_at")

	query, args, err := builder.ToSql()
	if err != nil {
		return "", time.Time{}, err
	}

	q := db.Query{
		Name:     "object_repository:InsertVersion",
		QueryRaw: query,
	}

	var versionID string
	var createdAt time.Time
	err = r.db.DB().QueryRowContext(ctx, q, args...).Scan(&versionID, &createdAt)
	if err != nil {
		return "", time.Time{}, err
	}

	return versionID, createdAt, nil
}
