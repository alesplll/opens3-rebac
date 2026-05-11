package tests

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestList_ReturnsOnlyActiveCommittedBlobVersions(t *testing.T) {
	ctx, client, cleanup, repo := newRepository(t)
	defer cleanup()

	execSQL(t, ctx, client, "insert_bucket", `
		INSERT INTO buckets (id, name, owner_id)
		VALUES ('bucket-1', 'my-bucket', 'user-1')
	`)
	execSQL(t, ctx, client, "insert_objects", `
		INSERT INTO objects (id, bucket_id, key, current_version_id, status)
		VALUES
			('object-1', 'bucket-1', 'photos/active.jpg', 'version-1', 'active'),
			('object-2', 'bucket-1', 'photos/uploading.jpg', 'version-2', 'uploading'),
			('object-3', 'bucket-1', 'photos/deleted.txt', 'version-3', 'active')
	`)
	execSQL(t, ctx, client, "insert_versions", `
		INSERT INTO versions (id, object_id, blob_id, size_bytes, etag, content_type, version_number, kind, state, committed_at)
		VALUES
			('version-1', 'object-1', 'blob-1', 100, '"etag-1"', 'image/jpeg', 1, 'blob', 'committed', now()),
			('version-2', 'object-2', 'blob-2', 200, '"etag-2"', 'image/jpeg', 1, 'blob', 'committed', now()),
			('version-3', 'object-3', NULL, 0, '', 'application/octet-stream', 1, 'delete_marker', 'committed', now())
	`)

	items, nextToken, isTruncated, err := repo.List(ctx, "my-bucket", "photos/", "", 100)

	require.NoError(t, err)
	require.False(t, isTruncated)
	require.Empty(t, nextToken)
	require.Len(t, items, 1)
	require.Equal(t, "object-1", items[0].ObjectID)
	require.Equal(t, "version-1", items[0].VersionID)
	require.Equal(t, "photos/active.jpg", items[0].Key)
}
