package config

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMustRepoRoot(t *testing.T) {
	root := MustRepoRoot(t)

	require.DirExists(t, root)
	require.FileExists(t, filepath.Join(root, "go.work"))
	require.FileExists(t, filepath.Join(root, "e2e", ".env"))
}

func TestMustServiceMigrationDir(t *testing.T) {
	root := MustRepoRoot(t)
	migrationDir := MustServiceMigrationDir(t, "metadata")

	require.Equal(t, filepath.Join(root, "services", "metadata", "migrations"), migrationDir)
	require.DirExists(t, migrationDir)
	require.FileExists(t, filepath.Join(migrationDir, "20260423000000_init.sql"))
}
