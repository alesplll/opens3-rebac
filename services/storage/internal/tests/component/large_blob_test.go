package component

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStoreAndRetrieveFull_20MB(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	size := 20 * 1024 * 1024 // 20 MB
	data := makePattern(size)

	t.Logf("Storing %d MB blob...", size/1024/1024)
	blobID, checksum := storeBlob(t, ctx, data, "application/octet-stream")

	require.NotEmpty(t, blobID)
	require.Equal(t, md5Hex(data), checksum)
	t.Logf("Stored: blob_id=%s, md5=%s", blobID, checksum)

	t.Log("Retrieving full blob...")
	got, totalSize := retrieveBlob(t, ctx, blobID, 0, 0)

	require.Equal(t, int64(size), totalSize)
	require.Len(t, got, size)
	require.Equal(t, data, got)
	t.Log("20 MB blob: store + retrieve OK, bytes match")
}
