package tests

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	storageRepo "github.com/alesplll/opens3-rebac/services/storage/internal/repository/storage"
	"github.com/stretchr/testify/require"
)

func TestCreateMultipartSession_Success(t *testing.T) {
	t.Parallel()

	multipartDir := t.TempDir()
	repository := storageRepo.NewRepository(testStorageConfig{
		dataDir:      t.TempDir(),
		multipartDir: multipartDir,
	})

	err := repository.CreateMultipartSession(context.Background(), "upload-1", 3, "video/mp4")
	require.NoError(t, err)

	metaPath := filepath.Join(multipartDir, "upload-1", "meta.json")
	metaRaw, err := os.ReadFile(metaPath)
	require.NoError(t, err)

	var meta struct {
		ExpectedParts int32  `json:"expected_parts"`
		ContentType   string `json:"content_type"`
	}
	err = json.Unmarshal(metaRaw, &meta)
	require.NoError(t, err)
	require.Equal(t, int32(3), meta.ExpectedParts)
	require.Equal(t, "video/mp4", meta.ContentType)
}
