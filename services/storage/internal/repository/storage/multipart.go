package storage

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	domainerrors "github.com/alesplll/opens3-rebac/services/storage/internal/errors/domain_errors"
	"github.com/alesplll/opens3-rebac/services/storage/internal/model"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/logger"
	"go.uber.org/zap"
)

const multipartMetaFilename = "meta.json"
const multipartCompletionDirname = "completed"

var afterAssemblePartsCommitHook = func(context.Context) {}

type multipartSessionMeta struct {
	ExpectedParts int32  `json:"expected_parts"`
	ContentType   string `json:"content_type"`
	BlobID        string `json:"blob_id"`
}

func (r *repo) CreateMultipartSession(ctx context.Context, uploadID string, expectedParts int32, contentType string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	sessionDir := r.multipartSessionPath(uploadID)
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		return fmt.Errorf("create multipart session dir: %w", err)
	}

	metaBytes, err := json.Marshal(multipartSessionMeta{
		ExpectedParts: expectedParts,
		ContentType:   contentType,
		BlobID:        blobIDForUpload(uploadID),
	})
	if err != nil {
		return fmt.Errorf("marshal multipart session meta: %w", err)
	}

	if err := os.WriteFile(r.multipartMetaPath(uploadID), metaBytes, 0o644); err != nil {
		return fmt.Errorf("write multipart session meta: %w", err)
	}

	return nil
}

func (r *repo) StorePart(ctx context.Context, uploadID string, partNumber int32, reader io.Reader) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	if !r.multipartSessionExists(uploadID) {
		return "", domainerrors.ErrUploadNotFound
	}

	result, err := writeAtomically(ctx, r.multipartPartPath(uploadID, partNumber), reader, "multipart part")
	if err != nil {
		return "", err
	}

	return result.checksumMD5, nil
}

func (r *repo) AssembleParts(ctx context.Context, uploadID string, parts []model.PartInfo, destBlobID string) (*model.BlobMeta, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	completedMeta, err := r.readCompletedMultipartMeta(uploadID)
	switch {
	case err == nil:
		exists, existsErr := r.blobExists(completedMeta.BlobID)
		if existsErr != nil {
			return nil, existsErr
		}
		if exists {
			return completedMeta, nil
		}

		if removeErr := r.removeCompletedMultipartMeta(uploadID); removeErr != nil {
			logger.Error(
				ctx,
				"failed to remove stale completed multipart meta",
				zap.Error(removeErr),
				zap.String("upload_id", uploadID),
				zap.String("blob_id", completedMeta.BlobID),
			)
		}
	case errors.Is(err, os.ErrNotExist):
		// No completed marker yet, continue with assembly from multipart session.
	default:
		if removeErr := r.removeCompletedMultipartMeta(uploadID); removeErr != nil {
			logger.Error(
				ctx,
				"failed to remove corrupted completed multipart meta",
				zap.Error(removeErr),
				zap.String("upload_id", uploadID),
			)
		}
	}

	// TODO: consider cleanup on assembly failure or add TTL-based garbage collection
	// without breaking retry semantics for recoverable CompleteMultipartUpload errors.
	sessionMeta, err := r.readMultipartSessionMeta(uploadID)
	if err != nil {
		return nil, err
	}

	if sessionMeta.ExpectedParts > 0 && int32(len(parts)) != sessionMeta.ExpectedParts {
		return nil, domainerrors.ErrInvalidParts
	}
	if sessionMeta.BlobID != "" {
		destBlobID = sessionMeta.BlobID
	}

	if err := ensureDirReady(r.dataDir); err != nil {
		return nil, err
	}

	finalPath := r.blobPath(destBlobID)
	file, tempPath, err := createAtomicTempFile(finalPath, "assembled blob")
	if err != nil {
		return nil, err
	}
	cleanupTemp := func() {
		_ = file.Close()
		_ = os.Remove(tempPath)
	}

	hasher := md5.New()
	writer := io.MultiWriter(file, hasher)
	var written int64

	for _, part := range parts {
		partPath := r.multipartPartPath(uploadID, part.PartNumber)
		partFile, openErr := os.Open(partPath)
		if openErr != nil {
			cleanupTemp()
			if errors.Is(openErr, os.ErrNotExist) {
				return nil, domainerrors.ErrUploadNotFound
			}

			return nil, fmt.Errorf("open multipart part: %w", openErr)
		}

		partHasher := md5.New()
		n, copyErr := io.Copy(writer, io.TeeReader(newContextReader(ctx, partFile), partHasher))
		closeErr := partFile.Close()
		if copyErr != nil {
			cleanupTemp()
			if isDiskFull(copyErr) {
				return nil, domainerrors.ErrDiskFull
			}

			return nil, fmt.Errorf("copy multipart part: %w", copyErr)
		}
		if closeErr != nil {
			cleanupTemp()
			return nil, fmt.Errorf("close multipart part: %w", closeErr)
		}

		if fmt.Sprintf("%x", partHasher.Sum(nil)) != part.ChecksumMD5 {
			cleanupTemp()
			return nil, domainerrors.ErrChecksumMismatch
		}

		written += n
	}

	if err := file.Sync(); err != nil {
		cleanupTemp()
		return nil, fmt.Errorf("sync assembled blob temp file: %w", err)
	}

	if err := file.Close(); err != nil {
		_ = os.Remove(tempPath)
		return nil, fmt.Errorf("close assembled blob temp file: %w", err)
	}

	if err := os.Rename(tempPath, finalPath); err != nil {
		_ = os.Remove(tempPath)
		return nil, fmt.Errorf("commit assembled blob file: %w", err)
	}

	meta := &model.BlobMeta{
		BlobID:      destBlobID,
		ChecksumMD5: fmt.Sprintf("%x", hasher.Sum(nil)),
		SizeBytes:   written,
		ContentType: sessionMeta.ContentType,
	}
	if err := r.writeCompletedMultipartMeta(uploadID, meta); err != nil {
		if deleteErr := r.DeleteBlob(context.WithoutCancel(ctx), destBlobID); deleteErr != nil {
			return nil, fmt.Errorf("persist completed multipart meta: %w (cleanup blob: %v)", err, deleteErr)
		}

		return nil, err
	}

	afterAssemblePartsCommitHook(ctx)
	r.cleanupMultipartBestEffort(ctx, uploadID)

	return meta, nil
}

func (r *repo) CleanupMultipart(ctx context.Context, uploadID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if err := os.RemoveAll(r.multipartSessionPath(uploadID)); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}

		return fmt.Errorf("cleanup multipart session: %w", err)
	}

	return nil
}

func (r *repo) cleanupMultipartBestEffort(ctx context.Context, uploadID string) {
	// Complete becomes externally visible after the final rename, so cleanup
	// must still be attempted even if the request context is already cancelled.
	cleanupCtx := context.WithoutCancel(ctx)

	if err := r.CleanupMultipart(cleanupCtx, uploadID); err != nil {
		logger.Error(
			cleanupCtx,
			"failed to cleanup multipart session after assembled blob commit",
			zap.Error(err),
			zap.String("upload_id", uploadID),
		)
	}
}

func (r *repo) writeCompletedMultipartMeta(uploadID string, meta *model.BlobMeta) error {
	if err := ensureDirReady(r.multipartCompletedDir()); err != nil {
		return err
	}

	metaBytes, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("marshal completed multipart meta: %w", err)
	}

	if _, err := writeAtomically(
		context.Background(),
		r.completedMultipartMetaPath(uploadID),
		bytes.NewReader(metaBytes),
		"completed multipart meta",
	); err != nil {
		return err
	}

	return nil
}

func (r *repo) multipartSessionPath(uploadID string) string {
	return filepath.Join(r.multipartDir, uploadID)
}

func (r *repo) multipartCompletedDir() string {
	return filepath.Join(r.multipartDir, multipartCompletionDirname)
}

func (r *repo) completedMultipartMetaPath(uploadID string) string {
	return filepath.Join(r.multipartCompletedDir(), uploadID+".json")
}

func (r *repo) multipartMetaPath(uploadID string) string {
	return filepath.Join(r.multipartSessionPath(uploadID), multipartMetaFilename)
}

func (r *repo) multipartPartPath(uploadID string, partNumber int32) string {
	return filepath.Join(r.multipartSessionPath(uploadID), fmt.Sprintf("part_%d", partNumber))
}

func (r *repo) multipartSessionExists(uploadID string) bool {
	info, err := os.Stat(r.multipartSessionPath(uploadID))
	return err == nil && info.IsDir()
}

func (r *repo) readMultipartSessionMeta(uploadID string) (*multipartSessionMeta, error) {
	metaBytes, err := os.ReadFile(r.multipartMetaPath(uploadID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, domainerrors.ErrUploadNotFound
		}

		return nil, fmt.Errorf("read multipart session meta: %w", err)
	}

	var meta multipartSessionMeta
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		return nil, fmt.Errorf("unmarshal multipart session meta: %w", err)
	}

	return &meta, nil
}

func (r *repo) readCompletedMultipartMeta(uploadID string) (*model.BlobMeta, error) {
	metaBytes, err := os.ReadFile(r.completedMultipartMetaPath(uploadID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, os.ErrNotExist
		}

		return nil, fmt.Errorf("read completed multipart meta: %w", err)
	}

	var meta model.BlobMeta
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		return nil, fmt.Errorf("unmarshal completed multipart meta: %w", err)
	}

	return &meta, nil
}

func (r *repo) removeCompletedMultipartMeta(uploadID string) error {
	if err := os.Remove(r.completedMultipartMetaPath(uploadID)); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}

		return fmt.Errorf("remove completed multipart meta: %w", err)
	}

	return nil
}

func (r *repo) blobExists(blobID string) (bool, error) {
	info, err := os.Stat(r.blobPath(blobID))
	if err == nil {
		return !info.IsDir(), nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}

	return false, fmt.Errorf("stat blob file: %w", err)
}

func blobIDForUpload(uploadID string) string {
	return uploadID
}
