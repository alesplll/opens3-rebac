package storage

import (
	"context"
	"errors"
	"fmt"
	"os"
)

func (r *repo) DeleteBlob(ctx context.Context, blobID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if err := os.Remove(r.blobPath(blobID)); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove blob file: %w", err)
		}
	}

	if err := r.removeCompletedMultipartMeta(blobID); err != nil {
		return err
	}

	return nil
}
