package storage

import (
	"context"
)

func (r *repo) HealthCheck(_ context.Context) error {
	// TODO: check DATA_DIR accessibility and disk space
	return nil
}
