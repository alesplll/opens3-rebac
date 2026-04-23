package object

import (
	"context"

	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/client/db"
)

// UpsertObject creates an object record if it does not exist yet (same bucket+key),
// or returns the existing object ID. Used inside CreateObjectVersion transaction.
func (r *repo) UpsertObject(ctx context.Context, bucketName, key string) (string, error) {
	rawSQL := `
INSERT INTO objects (bucket_id, key)
SELECT id, $2 FROM buckets WHERE name = $1
ON CONFLICT (bucket_id, key) DO UPDATE SET bucket_id = EXCLUDED.bucket_id
RETURNING id`

	q := db.Query{
		Name:     "object_repository:UpsertObject",
		QueryRaw: rawSQL,
	}

	var objectID string
	err := r.db.DB().QueryRowContext(ctx, q, bucketName, key).Scan(&objectID)
	if err != nil {
		return "", err
	}

	return objectID, nil
}
