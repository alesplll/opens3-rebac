package storage

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"

	"github.com/google/uuid"

	"github.com/alesplll/opens3-rebac/services/storage/internal/model"
)

func (r *repo) StoreBlob(_ context.Context, _ io.Reader) (*model.BlobMeta, error) {
	// TODO: implement actual file storage
	return &model.BlobMeta{
		BlobID:      uuid.New().String(),
		ChecksumMD5: fmt.Sprintf("%x", md5.Sum([]byte("stub"))),
	}, nil
}
