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
			req := &desc.StoreObjectRequest{Data: chunk[:n]}
			if first {
				req.Size = int64(len(data))
				req.ContentType = contentType
				first = false
			}
			err := stream.Send(req)
			require.NoError(t, err)
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
			Data:        nil,
			Size:        0,
			ContentType: contentType,
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
