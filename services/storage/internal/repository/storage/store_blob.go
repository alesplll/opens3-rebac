package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"

	domainerrors "github.com/alesplll/opens3-rebac/services/storage/internal/errors/domain_errors"
	"github.com/alesplll/opens3-rebac/services/storage/internal/model"
)

func (r *repo) StoreBlob(ctx context.Context, blobID string, reader io.Reader) (*model.BlobMeta, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if err := ensureDirReady(r.blobsRootPath()); err != nil {
		return nil, err
	}

	stagingDir := r.singlePartUploadPath(blobID)
	if err := ensureDirReady(stagingDir); err != nil {
		return nil, err
	}

	if err := r.writeSinglePartManifest(blobID); err != nil {
		return nil, err
	}

	result, err := writeAtomically(ctx, r.singlePartObjectPath(blobID), reader, "staged blob")
	if err != nil {
		return nil, err
	}

	if err := ensureDirReady(filepath.Dir(r.blobPath(blobID))); err != nil {
		return nil, err
	}

	if err := os.Rename(r.singlePartObjectPath(blobID), r.blobPath(blobID)); err != nil {
		return nil, fmt.Errorf("publish staged blob file: %w", err)
	}

	r.cleanupSinglePartStagingBestEffort(blobID)

	return &model.BlobMeta{
		BlobID:      blobID,
		ChecksumMD5: result.checksumMD5,
		SizeBytes:   result.sizeBytes,
	}, nil
}

func (r *repo) blobPath(blobID string) string {
	return filepath.Join(r.blobsRootPath(), blobShard(blobID), blobID)
}

func (r *repo) blobsRootPath() string {
	return r.dataDir
}

func blobShard(blobID string) string {
	if len(blobID) < blobsShardLength {
		return blobID
	}

	return blobID[:blobsShardLength]
}

func (r *repo) stagingRootPath() string {
	return r.multipartDir
}

func (r *repo) stagingUploadsPath() string {
	return filepath.Join(r.stagingRootPath(), stagingUploadsDirname)
}

func (r *repo) singlePartUploadPath(blobID string) string {
	return filepath.Join(r.stagingUploadsPath(), blobID)
}

func (r *repo) singlePartObjectPath(blobID string) string {
	return filepath.Join(r.singlePartUploadPath(blobID), stagingObjectFilename)
}

func (r *repo) writeSinglePartManifest(blobID string) error {
	manifestBytes, err := json.Marshal(struct {
		BlobID string `json:"blob_id"`
		State  string `json:"state"`
	}{
		BlobID: blobID,
		State:  "staged",
	})
	if err != nil {
		return fmt.Errorf("marshal staged blob manifest: %w", err)
	}

	if _, err := writeAtomically(
		context.Background(),
		filepath.Join(r.singlePartUploadPath(blobID), stagingManifestFilename),
		bytes.NewReader(manifestBytes),
		"staged blob manifest",
	); err != nil {
		return err
	}

	return nil
}

func (r *repo) cleanupSinglePartStagingBestEffort(blobID string) {
	// Single-part uploads are already externally visible once the staged file
	// is renamed into blobs/, so stale staging is only a cleanup concern.
	_ = os.RemoveAll(r.singlePartUploadPath(blobID))
}

func ensureWritableDiskSpace(dir string) error {
	var stats syscall.Statfs_t
	if err := syscall.Statfs(dir, &stats); err != nil {
		return fmt.Errorf("statfs data dir: %w", err)
	}

	if stats.Bavail == 0 || stats.Bsize == 0 {
		return domainerrors.ErrDiskFull
	}

	return nil
}

func isDiskFull(err error) bool {
	return errors.Is(err, syscall.ENOSPC)
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func newContextReader(ctx context.Context, reader io.Reader) io.Reader {
	return &contextReader{
		ctx:    ctx,
		reader: reader,
	}
}

func (r *contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}

	n, err := r.reader.Read(p)
	if err != nil {
		return n, err
	}

	if err := r.ctx.Err(); err != nil {
		return n, err
	}

	return n, nil
}
