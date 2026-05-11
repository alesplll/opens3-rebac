package tests

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDelete_AllowsNullCurrentVersionBlob(t *testing.T) {
	ctx, client, cleanup, repo := newRepository(t)
	defer cleanup()

	execSQL(t, ctx, client, "insert_bucket", `
		INSERT INTO buckets (id, name, owner_id)
		VALUES ('bucket-1', 'my-bucket', 'user-1')
	`)
	execSQL(t, ctx, client, "insert_object", `
		INSERT INTO objects (id, bucket_id, key, current_version_id, status)
		VALUES ('object-1', 'bucket-1', 'photos/cat.jpg', NULL, 'active')
	`)

	objectID, blobID, err := repo.Delete(ctx, "my-bucket", "photos/cat.jpg")

	require.NoError(t, err)
	require.Equal(t, "object-1", objectID)
	require.Empty(t, blobID)
}
