package storage

import (
	desc "github.com/alesplll/opens3-rebac/shared/pkg/go/storage/v1"
)

func (h *handler) StoreObject(stream desc.DataStorageService_StoreObjectServer) error {
	payload, err := readStoreObjectStream(stream)
	if err != nil {
		return err
	}

	meta, err := h.service.StoreObject(
		stream.Context(),
		payload.reader,
		payload.size,
		payload.contentType,
	)
	if err != nil {
		return err
	}

	return stream.SendAndClose(&desc.StoreObjectResponse{
		BlobId:      meta.BlobID,
		ChecksumMd5: meta.ChecksumMD5,
	})
}
