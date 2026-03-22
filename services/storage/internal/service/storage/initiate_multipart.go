package storage

import (
	"context"

	"github.com/google/uuid"
)

func (s *storageService) InitiateMultipartUpload(_ context.Context, _ int32, _ string) (string, error) {
	// TODO: implement multipart session creation
	return uuid.New().String(), nil
}
