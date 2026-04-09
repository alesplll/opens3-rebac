package component

import (
	"bytes"
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"testing"

	desc "github.com/alesplll/opens3-rebac/shared/pkg/go/storage/v1"
	"github.com/stretchr/testify/require"
)

const testChunkSize = 64 * 1024 // 64 KB

type testStorageConfig struct {
	dataDir      string
	multipartDir string
}

func (c testStorageConfig) DataDir() string      { return c.dataDir }
func (c testStorageConfig) MultipartDir() string { return c.multipartDir }

func storeBlob(t *testing.T, ctx context.Context, data []byte, contentType string) (string, string) {
	t.Helper()

	stream, err := client.StoreObject(ctx)
	require.NoError(t, err)

	buf := bytes.NewReader(data)
	chunk := make([]byte, testChunkSize)
	first := true

	for {
		n, readErr := buf.Read(chunk)
		if n > 0 {
			if first {
				req := &desc.StoreObjectRequest{
					Payload: &desc.StoreObjectRequest_Header{
						Header: &desc.StoreObjectHeader{
							Size:        int64Ptr(int64(len(data))),
							ContentType: contentType,
							Data:        chunk[:n],
						},
					},
				}
				err := stream.Send(req)
				require.NoError(t, err)
				first = false
			} else {
				req := &desc.StoreObjectRequest{
					Payload: &desc.StoreObjectRequest_Chunk{
						Chunk: &desc.StoreObjectChunk{Data: chunk[:n]},
					},
				}
				err := stream.Send(req)
				require.NoError(t, err)
			}
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			require.NoError(t, readErr)
		}
	}

	// Handle empty blob: must send at least one message
	if first {
		err := stream.Send(&desc.StoreObjectRequest{
			Payload: &desc.StoreObjectRequest_Header{
				Header: &desc.StoreObjectHeader{
					Size:        int64Ptr(0),
					ContentType: contentType,
				},
			},
		})
		require.NoError(t, err)
	}

	resp, err := stream.CloseAndRecv()
	require.NoError(t, err)

	return resp.GetBlobId(), resp.GetChecksumMd5()
}

func retrieveBlob(t *testing.T, ctx context.Context, blobID string, offset, length int64) ([]byte, int64) {
	t.Helper()

	stream, err := client.RetrieveObject(ctx, &desc.RetrieveObjectRequest{
		BlobId: blobID,
		Offset: offset,
		Length: length,
	})
	require.NoError(t, err)

	var buf bytes.Buffer
	var totalSize int64
	first := true

	for {
		resp, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			require.NoError(t, err)
		}

		if first {
			totalSize = resp.GetTotalSize()
			first = false
		}

		_, writeErr := buf.Write(resp.GetData())
		require.NoError(t, writeErr)
	}

	return buf.Bytes(), totalSize
}

func initiateMultipart(t *testing.T, ctx context.Context, expectedParts int32, contentType string) string {
	t.Helper()

	resp, err := client.InitiateMultipartUpload(ctx, &desc.InitiateMultipartUploadRequest{
		ExpectedParts: expectedParts,
		ContentType:   contentType,
	})
	require.NoError(t, err)
	require.NotEmpty(t, resp.GetUploadId())

	return resp.GetUploadId()
}

func uploadPart(t *testing.T, ctx context.Context, uploadID string, partNumber int32, data []byte) string {
	t.Helper()

	stream, err := client.UploadPart(ctx)
	require.NoError(t, err)

	buf := bytes.NewReader(data)
	chunk := make([]byte, testChunkSize)
	first := true

	for {
		n, readErr := buf.Read(chunk)
		if n > 0 {
			if first {
				req := &desc.UploadPartRequest{
					Payload: &desc.UploadPartRequest_Header{
						Header: &desc.UploadPartHeader{
							UploadId:   uploadID,
							PartNumber: partNumber,
							Data:       chunk[:n],
						},
					},
				}
				err := stream.Send(req)
				require.NoError(t, err)
				first = false
			} else {
				req := &desc.UploadPartRequest{
					Payload: &desc.UploadPartRequest_Chunk{
						Chunk: &desc.UploadPartChunk{Data: chunk[:n]},
					},
				}
				err := stream.Send(req)
				require.NoError(t, err)
			}
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			require.NoError(t, readErr)
		}
	}

	if first {
		err := stream.Send(&desc.UploadPartRequest{
			Payload: &desc.UploadPartRequest_Header{
				Header: &desc.UploadPartHeader{
					UploadId:   uploadID,
					PartNumber: partNumber,
				},
			},
		})
		require.NoError(t, err)
	}

	resp, err := stream.CloseAndRecv()
	require.NoError(t, err)
	return resp.GetPartChecksumMd5()
}

func completeMultipart(t *testing.T, ctx context.Context, uploadID string, parts []*desc.PartInfo) (string, string) {
	t.Helper()

	resp, err := client.CompleteMultipartUpload(ctx, &desc.CompleteMultipartUploadRequest{
		UploadId: uploadID,
		Parts:    parts,
	})
	require.NoError(t, err)

	return resp.GetBlobId(), resp.GetChecksumMd5()
}

func abortMultipart(t *testing.T, ctx context.Context, uploadID string) {
	t.Helper()

	resp, err := client.AbortMultipartUpload(ctx, &desc.AbortMultipartUploadRequest{
		UploadId: uploadID,
	})
	require.NoError(t, err)
	require.True(t, resp.GetSuccess())
}

func md5Hex(data []byte) string {
	sum := md5.Sum(data)
	return fmt.Sprintf("%x", sum)
}

func makePattern(size int) []byte {
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(i % 256)
	}
	return data
}

func int64Ptr(v int64) *int64 {
	return &v
}
