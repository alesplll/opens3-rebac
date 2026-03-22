package storage

import (
	"context"
)

func (r *repo) DeleteBlob(_ context.Context, _ string) error {
	// TODO: implement actual file deletion
	return nil
}
