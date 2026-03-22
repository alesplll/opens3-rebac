package storage

import (
	"bytes"
	"context"
	"io"
)

func (r *repo) RetrieveBlob(_ context.Context, _ string, _ int64, _ int64) (io.ReadCloser, int64, error) {
	// TODO: implement actual file retrieval
	data := []byte("stub blob content")
	return io.NopCloser(bytes.NewReader(data)), int64(len(data)), nil
}
