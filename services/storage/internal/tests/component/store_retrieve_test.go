package component

import (
	"context"
	"testing"

	desc "github.com/alesplll/opens3-rebac/shared/pkg/go/storage/v1"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestStoreAndRetrieveFull_SmallBlob(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	data := makePattern(137)
	blobID, checksum := storeBlob(t, ctx, data, "application/octet-stream")

	require.NotEmpty(t, blobID)
	require.Equal(t, md5Hex(data), checksum)

	got, totalSize := retrieveBlob(t, ctx, blobID, 0, 0)
	require.Equal(t, int64(137), totalSize)
	require.Equal(t, data, got)
}

func TestStoreAndRetrieveFull_MultiChunkBlob(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	data := makePattern(2_621_440) // 2.5 MB
	blobID, checksum := storeBlob(t, ctx, data, "application/octet-stream")

	require.Equal(t, md5Hex(data), checksum)

	got, totalSize := retrieveBlob(t, ctx, blobID, 0, 0)
	require.Equal(t, int64(2_621_440), totalSize)
	require.Equal(t, data, got)
}

func TestStoreAndRetrieveRange_MiddleSlice(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	data := makePattern(2_621_440)
	blobID, _ := storeBlob(t, ctx, data, "application/octet-stream")

	got, totalSize := retrieveBlob(t, ctx, blobID, 1000, 5000)
	require.Equal(t, int64(2_621_440), totalSize)
	require.Equal(t, data[1000:6000], got)
}

func TestStoreAndRetrieveRange_OffsetToEnd(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	data := makePattern(1024)
	blobID, _ := storeBlob(t, ctx, data, "application/octet-stream")

	got, totalSize := retrieveBlob(t, ctx, blobID, 500, 0)
	require.Equal(t, int64(1024), totalSize)
	require.Equal(t, data[500:], got)
}

func TestStoreAndRetrieveRange_BeyondEnd(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	data := makePattern(100)
	blobID, _ := storeBlob(t, ctx, data, "application/octet-stream")

	got, totalSize := retrieveBlob(t, ctx, blobID, 50, 9999)
	require.Equal(t, int64(100), totalSize)
	require.Equal(t, data[50:], got)
}

func TestStoreAndRetrieve_EmptyBlob(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	blobID, checksum := storeBlob(t, ctx, []byte{}, "application/octet-stream")

	require.NotEmpty(t, blobID)
	require.Equal(t, "d41d8cd98f00b204e9800998ecf8427e", checksum)

	got, totalSize := retrieveBlob(t, ctx, blobID, 0, 0)
	require.Equal(t, int64(0), totalSize)
	require.Empty(t, got)
}

func TestRetrieve_NotFound(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	stream, err := client.RetrieveObject(ctx, &desc.RetrieveObjectRequest{
		BlobId: "nonexistent-blob-id",
	})
	require.NoError(t, err)

	_, err = stream.Recv()
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.NotFound, st.Code())
}
