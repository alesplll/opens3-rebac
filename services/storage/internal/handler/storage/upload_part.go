package storage

import (
	"bytes"
	"io"

	desc "github.com/alesplll/opens3-rebac/shared/pkg/storage/v1"
)

func (h *handler) UploadPart(stream desc.DataStorageService_UploadPartServer) error {
	var buf bytes.Buffer
	var uploadID string
	var partNumber int32

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if uploadID == "" {
			uploadID = req.GetUploadId()
			partNumber = req.GetPartNumber()
		}

		buf.Write(req.GetData())
	}

	checksum, err := h.service.UploadPart(stream.Context(), uploadID, partNumber, &buf)
	if err != nil {
		return err
	}

	return stream.SendAndClose(&desc.UploadPartResponse{
		PartChecksumMd5: checksum,
	})
}
