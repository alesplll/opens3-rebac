package storage

import (
	"errors"
	"io"

	desc "github.com/alesplll/opens3-rebac/shared/pkg/go/storage/v1"
)

func (h *handler) StoreObject(stream desc.DataStorageService_StoreObjectServer) error {
	firstReq, err := stream.Recv()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return io.ErrUnexpectedEOF
		}
		return err
	}

	reader := &chunkReader{
		recvChunk: func() ([]byte, error) {
			req, err := stream.Recv()
			if err != nil {
				return nil, err
			}

			return req.GetData(), nil
		},
		pending: firstReq.GetData(),
	}

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
