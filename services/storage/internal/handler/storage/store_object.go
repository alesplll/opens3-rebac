package storage

import (
	desc "github.com/alesplll/opens3-rebac/shared/pkg/go/storage/v1"
)

func (h *handler) StoreObject(stream desc.DataStorageService_StoreObjectServer) error {
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

	meta, err := h.service.StoreObject(
		stream.Context(),
		reader,
		firstReq.GetSize(),
		firstReq.GetContentType(),
	)
	if err != nil {
		return err
	}

	return stream.SendAndClose(&desc.StoreObjectResponse{
		BlobId:      meta.BlobID,
		ChecksumMd5: meta.ChecksumMD5,
	})
}
