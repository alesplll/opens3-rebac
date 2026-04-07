package component

import (
	"context"
	"testing"

	desc "github.com/alesplll/opens3-rebac/shared/pkg/go/storage/v1"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestMultipartUploadComplete_Success(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	uploadID := initiateMultipart(t, ctx, 2, "video/mp4")

	partOne := []byte("hello ")
	partTwo := []byte("world")
	checksumOne := uploadPart(t, ctx, uploadID, 1, partOne)
	checksumTwo := uploadPart(t, ctx, uploadID, 2, partTwo)

	blobID, checksum := completeMultipart(t, ctx, uploadID, []*desc.PartInfo{
		{PartNumber: 1, ChecksumMd5: checksumOne},
		{PartNumber: 2, ChecksumMd5: checksumTwo},
	})

	body, totalSize := retrieveBlob(t, ctx, blobID, 0, 0)
	require.Equal(t, int64(len(partOne)+len(partTwo)), totalSize)
	require.Equal(t, append(partOne, partTwo...), body)
	require.Equal(t, md5Hex(body), checksum)
}

func TestMultipartUploadComplete_SuccessMultiChunkPart(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	uploadID := initiateMultipart(t, ctx, 2, "application/octet-stream")

	partOne := makePattern(testChunkSize*2 + 123)
	partTwo := makePattern(testChunkSize + 17)
	checksumOne := uploadPart(t, ctx, uploadID, 1, partOne)
	checksumTwo := uploadPart(t, ctx, uploadID, 2, partTwo)

	blobID, checksum := completeMultipart(t, ctx, uploadID, []*desc.PartInfo{
		{PartNumber: 1, ChecksumMd5: checksumOne},
		{PartNumber: 2, ChecksumMd5: checksumTwo},
	})

	body, totalSize := retrieveBlob(t, ctx, blobID, 0, 0)
	expectedBody := append(partOne, partTwo...)
	require.Equal(t, int64(len(expectedBody)), totalSize)
	require.Equal(t, expectedBody, body)
	require.Equal(t, md5Hex(body), checksum)
}

func TestMultipartUploadAbort_Success(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	uploadID := initiateMultipart(t, ctx, 1, "video/mp4")
	_ = uploadPart(t, ctx, uploadID, 1, []byte("hello"))

	abortMultipart(t, ctx, uploadID)

	stream, err := client.UploadPart(ctx)
	require.NoError(t, err)
	err = stream.Send(&desc.UploadPartRequest{
		UploadId:   uploadID,
		PartNumber: 1,
		Data:       []byte("retry"),
	})
	require.NoError(t, err)

	_, err = stream.CloseAndRecv()
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.NotFound, st.Code())
}

func TestMultipartUploadComplete_ChecksumMismatch(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	uploadID := initiateMultipart(t, ctx, 1, "video/mp4")
	_ = uploadPart(t, ctx, uploadID, 1, []byte("hello"))

	_, err := client.CompleteMultipartUpload(ctx, &desc.CompleteMultipartUploadRequest{
		UploadId: uploadID,
		Parts: []*desc.PartInfo{
			{PartNumber: 1, ChecksumMd5: "bad-checksum"},
		},
	})
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.InvalidArgument, st.Code())
}

func TestMultipartUploadPart_RetryOverwritesPart(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	uploadID := initiateMultipart(t, ctx, 1, "video/mp4")

	firstPart := []byte("first body")
	_ = uploadPart(t, ctx, uploadID, 1, firstPart)

	secondPart := []byte("second body")
	secondChecksum := uploadPart(t, ctx, uploadID, 1, secondPart)

	blobID, checksum := completeMultipart(t, ctx, uploadID, []*desc.PartInfo{
		{PartNumber: 1, ChecksumMd5: secondChecksum},
	})

	body, totalSize := retrieveBlob(t, ctx, blobID, 0, 0)
	require.Equal(t, int64(len(secondPart)), totalSize)
	require.Equal(t, secondPart, body)
	require.Equal(t, md5Hex(secondPart), checksum)
}

func TestMultipartUploadComplete_MissingPartReturnsNotFound(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	uploadID := initiateMultipart(t, ctx, 2, "video/mp4")

	partOne := []byte("hello")
	checksumOne := uploadPart(t, ctx, uploadID, 1, partOne)

	_, err := client.CompleteMultipartUpload(ctx, &desc.CompleteMultipartUploadRequest{
		UploadId: uploadID,
		Parts: []*desc.PartInfo{
			{PartNumber: 1, ChecksumMd5: checksumOne},
			{PartNumber: 2, ChecksumMd5: "missing-part-checksum"},
		},
	})
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.NotFound, st.Code())
}

func TestMultipartUploadAbort_AfterPartialUploadPreventsComplete(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	uploadID := initiateMultipart(t, ctx, 2, "video/mp4")

	partOne := []byte("hello")
	checksumOne := uploadPart(t, ctx, uploadID, 1, partOne)

	abortMultipart(t, ctx, uploadID)

	_, err := client.CompleteMultipartUpload(ctx, &desc.CompleteMultipartUploadRequest{
		UploadId: uploadID,
		Parts: []*desc.PartInfo{
			{PartNumber: 1, ChecksumMd5: checksumOne},
		},
	})
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.NotFound, st.Code())
}
