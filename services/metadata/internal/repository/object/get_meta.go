package object

import (
	"context"
	"errors"

	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/client/db"
	domainerrors "github.com/alesplll/opens3-rebac/services/metadata/internal/errors/domain_errors"
	"github.com/alesplll/opens3-rebac/services/metadata/internal/model"
	repoModel "github.com/alesplll/opens3-rebac/services/metadata/internal/repository/object/model"
	"github.com/alesplll/opens3-rebac/services/metadata/internal/repository/object/converter"
	"github.com/jackc/pgx/v4"
)

func (r *repo) GetMeta(ctx context.Context, bucketName, key, versionID string) (*model.ObjectMeta, error) {
	var rawSQL string
	var args []any

	if versionID != "" {
		rawSQL = `
SELECT o.id AS object_id,
       v.id AS version_id,
       v.blob_id,
       v.size_bytes,
       v.etag,
       v.content_type,
       v.created_at AS last_modified
FROM objects o
JOIN buckets b ON o.bucket_id = b.id
JOIN versions v ON v.id = $3
WHERE b.name = $1 AND o.key = $2 AND v.is_deleted = false`
		args = []any{bucketName, key, versionID}
	} else {
		rawSQL = `
SELECT o.id AS object_id,
       v.id AS version_id,
       v.blob_id,
       v.size_bytes,
       v.etag,
       v.content_type,
       v.created_at AS last_modified
FROM objects o
JOIN buckets b ON o.bucket_id = b.id
JOIN versions v ON v.id = o.current_version_id
WHERE b.name = $1 AND o.key = $2 AND v.is_deleted = false`
		args = []any{bucketName, key}
	}

	q := db.Query{
		Name:     "object_repository:GetMeta",
		QueryRaw: rawSQL,
	}

	var m repoModel.ObjectMeta
	err := r.db.DB().ScanOneContext(ctx, &m, q, args...)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.ErrObjectNotFound
		}
		return nil, err
	}

	return converter.MetaToDomain(&m), nil
}
