package object

import (
	"context"
	"errors"

	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/client/db"
	domainerrors "github.com/alesplll/opens3-rebac/services/metadata/internal/errors/domain_errors"
	"github.com/jackc/pgx/v4"
)

func (r *repo) Delete(ctx context.Context, bucketName, key string) (string, string, error) {
	rawSQL := `
DELETE FROM objects o
USING buckets b
WHERE o.bucket_id = b.id AND b.name = $1 AND o.key = $2
RETURNING o.id,
  (SELECT blob_id FROM versions WHERE object_id = o.id AND id = o.current_version_id)`

	q := db.Query{
		Name:     "object_repository:Delete",
		QueryRaw: rawSQL,
	}

	var objectID, blobID string
	err := r.db.DB().QueryRowContext(ctx, q, bucketName, key).Scan(&objectID, &blobID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", domainerrors.ErrObjectNotFound
		}
		return "", "", err
	}

	return objectID, blobID, nil
}
