package storage

import (
	"bytes"
	"io"

	desc "github.com/alesplll/opens3-rebac/shared/pkg/storage/v1"
)

func (h *handler) StoreObject(stream desc.DataStorageService_StoreObjectServer) error {
	var buf bytes.Buffer
	var size int64
	var contentType string

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if size == 0 {
			size = req.GetSize()
			contentType = req.GetContentType()
		}

		buf.Write(req.GetData())
	}

	meta, err := h.service.StoreObject(stream.Context(), &buf, size, contentType)
	if err != nil {
		return err
	}

	return stream.SendAndClose(&desc.StoreObjectResponse{
		BlobId:      meta.BlobID,
		ChecksumMd5: meta.ChecksumMD5,
	})
}
