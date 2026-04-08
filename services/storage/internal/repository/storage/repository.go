package storage

import (
	"github.com/alesplll/opens3-rebac/services/storage/internal/config"
	"github.com/alesplll/opens3-rebac/services/storage/internal/repository"
)

type repo struct {
	dataDir      string
	multipartDir string
}

func NewRepository(cfg config.StorageConfig) repository.StorageRepository {
	return &repo{
		dataDir:      cfg.DataDir(),
		multipartDir: cfg.MultipartDir(),
	}
}

const (
	blobsShardLength        = 2
	stagingUploadsDirname   = "uploads"
	stagingObjectFilename   = "object.bin"
	stagingManifestFilename = "manifest.json"
)
