package env

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStorageConfigRetrieveChunkSize(t *testing.T) {
	t.Setenv("STORAGE_RETRIEVE_CHUNK_SIZE_BYTES", "262144")

	cfg, err := NewStorageConfig()
	require.NoError(t, err)
	require.Equal(t, 262144, cfg.ChunkSizeBytes())
}

func TestStorageConfigRejectsInvalidRetrieveChunkSize(t *testing.T) {
	t.Setenv("STORAGE_RETRIEVE_CHUNK_SIZE_BYTES", "0")

	_, err := NewStorageConfig()
	require.ErrorContains(t, err, "STORAGE_RETRIEVE_CHUNK_SIZE_BYTES")
}
