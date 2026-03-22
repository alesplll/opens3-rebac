package storage

import (
	"context"
)

func (s *storageService) AbortMultipartUpload(_ context.Context, _ string) error {
	// TODO: implement multipart cleanup
	return nil
}
