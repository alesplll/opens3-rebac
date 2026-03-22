package storage

import (
	"context"
	"crypto/md5"
	"fmt"

	"github.com/google/uuid"

	"github.com/alesplll/opens3-rebac/services/storage/internal/model"
)

func (s *storageService) CompleteMultipartUpload(_ context.Context, _ string, _ []model.PartInfo) (*model.BlobMeta, error) {
	// TODO: implement part concatenation
	return &model.BlobMeta{
		BlobID:      uuid.New().String(),
		ChecksumMD5: fmt.Sprintf("%x", md5.Sum([]byte("stub-complete"))),
	}, nil
}
