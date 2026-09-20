package env

import "testing"

func TestStorageConfigRetrieveChunkSize(t *testing.T) {
	t.Setenv("STORAGE_RETRIEVE_CHUNK_SIZE_BYTES", "262144")

	cfg, err := NewStorageConfig()
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.RetrieveChunkSizeBytes(); got != 262144 {
		t.Fatalf("RetrieveChunkSizeBytes() = %d, want 262144", got)
	}
}

func TestStorageConfigRejectsInvalidRetrieveChunkSize(t *testing.T) {
	t.Setenv("STORAGE_RETRIEVE_CHUNK_SIZE_BYTES", "0")

	if _, err := NewStorageConfig(); err == nil {
		t.Fatal("NewStorageConfig() accepted a zero chunk size")
	}
}
