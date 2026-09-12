package object

import (
	"context"
	"time"

	"github.com/alesplll/opens3-rebac/services/metadata/internal/model"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/client/db"
)

// InsertVersion adds a new version record for an existing object.
// Used inside CreateObjectVersion transaction after UpsertObject.
func (r *repo) InsertVersion(ctx context.Context, objectID, blobID string, sizeBytes int64, etag, contentType string) (string, time.Time, error) {
	// Lock the object row while allocating its next monotonic version number.
	// This is intentionally a separate statement: under READ COMMITTED the insert
	// then gets a fresh snapshot after a concurrent holder releases the row lock.
	// Callers must keep both statements in the surrounding transaction.
	lockQuery := db.Query{
		Name:     "object_repository:LockObjectForVersionInsert",
		QueryRaw: "SELECT id FROM objects WHERE id = $1 FOR UPDATE",
	}
	var lockedObjectID string
	if err := r.db.DB().QueryRowContext(ctx, lockQuery, objectID).Scan(&lockedObjectID); err != nil {
		return "", time.Time{}, err
	}

	query := `
INSERT INTO versions (
    object_id,
    blob_id,
    size_bytes,
    etag,
    content_type,
    version_number,
    state,
    committed_at
)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    (SELECT COALESCE(MAX(version_number), 0) + 1 FROM versions WHERE object_id = $1),
    $6,
    now()
)
RETURNING id, created_at`
	args := []any{objectID, blobID, sizeBytes, etag, contentType, model.VersionStateCommitted}

	q := db.Query{
		Name:     "object_repository:InsertVersion",
		QueryRaw: query,
	}

	var versionID string
	var createdAt time.Time
	err := r.db.DB().QueryRowContext(ctx, q, args...).Scan(&versionID, &createdAt)
	if err != nil {
		return "", time.Time{}, err
	}

	return versionID, createdAt, nil
}
