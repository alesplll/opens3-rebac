package storage

import (
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
)

const multipartMetaFilename = "meta.json"

type multipartSessionMeta struct {
	ExpectedParts int32  `json:"expected_parts"`
	ContentType   string `json:"content_type"`
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

	// TODO: consider cleanup on assembly failure or add TTL-based garbage collection
	// without breaking retry semantics for recoverable CompleteMultipartUpload errors.
	sessionMeta, err := r.readMultipartSessionMeta(uploadID)
	if err != nil {
		return nil, err
	}

	if sessionMeta.ExpectedParts > 0 && int32(len(parts)) != sessionMeta.ExpectedParts {
		return nil, domainerrors.ErrInvalidParts
	}

	if err := ensureDirReady(r.dataDir); err != nil {
		return nil, err
	}

	finalPath := r.blobPath(destBlobID)
	tempPath := finalPath + ".tmp"
	file, err := os.OpenFile(tempPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("create assembled blob temp file: %w", err)
	}

	hasher := md5.New()
	writer := io.MultiWriter(file, hasher)
	var written int64

	for _, part := range parts {
		partPath := r.multipartPartPath(uploadID, part.PartNumber)
		partFile, openErr := os.Open(partPath)
		if openErr != nil {
			_ = file.Close()
			_ = os.Remove(tempPath)
			if errors.Is(openErr, os.ErrNotExist) {
				return nil, domainerrors.ErrUploadNotFound
			}

			return nil, fmt.Errorf("open multipart part: %w", openErr)
		}

		partHasher := md5.New()
		n, copyErr := io.Copy(writer, io.TeeReader(newContextReader(ctx, partFile), partHasher))
		closeErr := partFile.Close()
		if copyErr != nil {
			_ = file.Close()
			_ = os.Remove(tempPath)
			if isDiskFull(copyErr) {
				return nil, domainerrors.ErrDiskFull
			}

			return nil, fmt.Errorf("copy multipart part: %w", copyErr)
		}
		if closeErr != nil {
			_ = file.Close()
			_ = os.Remove(tempPath)
			return nil, fmt.Errorf("close multipart part: %w", closeErr)
		}

		if fmt.Sprintf("%x", partHasher.Sum(nil)) != part.ChecksumMD5 {
			_ = file.Close()
			_ = os.Remove(tempPath)
			return nil, domainerrors.ErrChecksumMismatch
		}

		written += n
	}

	if err := file.Sync(); err != nil {
		_ = file.Close()
		_ = os.Remove(tempPath)
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

	if err := r.CleanupMultipart(ctx, uploadID); err != nil {
		return nil, err
	}

	return &model.BlobMeta{
		BlobID:      destBlobID,
		ChecksumMD5: fmt.Sprintf("%x", hasher.Sum(nil)),
		SizeBytes:   written,
		ContentType: sessionMeta.ContentType,
	}, nil
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

func (r *repo) multipartSessionPath(uploadID string) string {
	return filepath.Join(r.multipartDir, uploadID)
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
