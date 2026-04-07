package storage

import (
	desc "github.com/alesplll/opens3-rebac/shared/pkg/go/storage/v1"
)

func (h *handler) UploadPart(stream desc.DataStorageService_UploadPartServer) error {
	firstReq, err := recvFirstMessage(stream.Recv)
	if err != nil {
		return err
	}

	reader := newChunkStreamReader(firstReq.GetData(), func() ([]byte, error) {
		req, err := stream.Recv()
		if err != nil {
			return nil, err
		}

		return req.GetData(), nil
	})

	checksum, err := h.service.UploadPart(
		stream.Context(),
		firstReq.GetUploadId(),
		firstReq.GetPartNumber(),
		reader,
	)
	if err != nil {
		return err
	}

	return stream.SendAndClose(&desc.UploadPartResponse{
		PartChecksumMd5: checksum,
	})
}
