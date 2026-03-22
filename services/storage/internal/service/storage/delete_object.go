package storage

import (
	"context"
)

func (s *storageService) DeleteObject(ctx context.Context, blobID string) error {
	return s.repo.DeleteBlob(ctx, blobID)
}
