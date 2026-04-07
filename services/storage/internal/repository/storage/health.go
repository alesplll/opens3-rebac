package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

func (r *repo) HealthCheck(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	info, err := os.Stat(r.dataDir)
	if err != nil {
		return fmt.Errorf("stat data dir: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("data dir is not a directory: %s", r.dataDir)
	}

	if err := ensureWritableDiskSpace(r.dataDir); err != nil {
		return err
	}

	tempFile, err := os.CreateTemp(r.dataDir, "healthcheck-*")
	if err != nil {
		return fmt.Errorf("create temp healthcheck file: %w", err)
	}

	tempPath := tempFile.Name()
	if _, err := tempFile.WriteString("ok"); err != nil {
		_ = tempFile.Close()
		_ = os.Remove(tempPath)
		return fmt.Errorf("write temp healthcheck file: %w", err)
	}

	if err := tempFile.Close(); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("close temp healthcheck file: %w", err)
	}

	if err := os.Remove(filepath.Clean(tempPath)); err != nil {
		return fmt.Errorf("remove temp healthcheck file: %w", err)
	}

	return nil
}
