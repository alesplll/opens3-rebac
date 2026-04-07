package storage

import (
	"context"
	"io"

	domainerrors "github.com/alesplll/opens3-rebac/services/storage/internal/errors/domain_errors"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/logger"
	"go.uber.org/zap"
)

func (s *storageService) UploadPart(ctx context.Context, uploadID string, partNumber int32, reader io.Reader) (string, error) {
	if partNumber < 1 {
		return "", domainerrors.ErrInvalidPartNumber
	}

	checksum, err := s.repo.StorePart(ctx, uploadID, partNumber, reader)
	if err != nil {
		logger.Error(
			ctx,
			"failed to store multipart part",
			zap.Error(err),
			zap.String("upload_id", uploadID),
			zap.Int32("part_number", partNumber),
		)
		return "", err
	}

	logger.Info(
		ctx,
		"stored multipart part",
		zap.String("upload_id", uploadID),
		zap.Int32("part_number", partNumber),
		zap.String("checksum_md5", checksum),
	)
	return checksum, nil
}
