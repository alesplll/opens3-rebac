package component

import (
	"context"
	"testing"

	desc "github.com/alesplll/opens3-rebac/shared/pkg/go/storage/v1"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestStoreDeleteRetrieve_NotFound(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	data := makePattern(256)
	blobID, _ := storeBlob(t, ctx, data, "application/octet-stream")

	resp, err := client.DeleteObject(ctx, &desc.DeleteObjectRequest{BlobId: blobID})
	require.NoError(t, err)
	require.True(t, resp.GetSuccess())

	stream, err := client.RetrieveObject(ctx, &desc.RetrieveObjectRequest{BlobId: blobID})
	require.NoError(t, err)

	_, err = stream.Recv()
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.NotFound, st.Code())
}

func TestDeleteNonExistent_Success(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	resp, err := client.DeleteObject(ctx, &desc.DeleteObjectRequest{BlobId: "does-not-exist"})
	require.NoError(t, err)
	require.True(t, resp.GetSuccess())
}
