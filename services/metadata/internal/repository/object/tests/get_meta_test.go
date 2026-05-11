package tests

import (
	"testing"

	domainerrors "github.com/alesplll/opens3-rebac/services/metadata/internal/errors/domain_errors"
	"github.com/stretchr/testify/require"
)

func TestGetMeta_VersionMustBelongToRequestedObject(t *testing.T) {
	ctx, client, cleanup, repo := newRepository(t)
	defer cleanup()

	execSQL(t, ctx, client, "insert_bucket", `
		INSERT INTO buckets (id, name, owner_id)
		VALUES ('bucket-1', 'my-bucket', 'user-1')
	`)
	execSQL(t, ctx, client, "insert_objects", `
		INSERT INTO objects (id, bucket_id, key, status)
		VALUES
			('object-1', 'bucket-1', 'photos/cat.jpg', 'active'),
			('object-2', 'bucket-1', 'photos/dog.jpg', 'active')
	`)
	execSQL(t, ctx, client, "insert_versions", `
		INSERT INTO versions (id, object_id, blob_id, size_bytes, etag, content_type, version_number, kind, state, committed_at)
		VALUES
			('version-1', 'object-2', 'blob-2', 2048, '"etag-2"', 'image/jpeg', 1, 'blob', 'committed', now())
	`)

	got, err := repo.GetMeta(ctx, "my-bucket", "photos/cat.jpg", "version-1")

	require.Nil(t, got)
	require.ErrorIs(t, err, domainerrors.ErrObjectNotFound)
}
