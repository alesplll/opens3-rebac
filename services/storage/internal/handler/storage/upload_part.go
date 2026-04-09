package storage

import (
	desc "github.com/alesplll/opens3-rebac/shared/pkg/go/storage/v1"
)

func (h *handler) UploadPart(stream desc.DataStorageService_UploadPartServer) error {
	payload, err := readUploadPartStream(stream)
	if err != nil {
		return err
	}

	checksum, err := h.service.UploadPart(
		stream.Context(),
		payload.uploadID,
		payload.partNumber,
		payload.reader,
	)
	if err != nil {
		return err
	}

	return stream.SendAndClose(&desc.UploadPartResponse{
		PartChecksumMd5: checksum,
	})
}
