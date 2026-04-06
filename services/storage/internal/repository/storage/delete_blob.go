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
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}

		return fmt.Errorf("remove blob file: %w", err)
	}

	return nil
}
