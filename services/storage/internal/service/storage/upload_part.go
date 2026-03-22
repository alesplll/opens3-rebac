package storage

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
)

func (s *storageService) UploadPart(_ context.Context, _ string, _ int32, _ io.Reader) (string, error) {
	// TODO: implement part storage
	return fmt.Sprintf("%x", md5.Sum([]byte("stub-part"))), nil
}
