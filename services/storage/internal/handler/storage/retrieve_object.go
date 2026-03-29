package storage

import (
	desc "github.com/alesplll/opens3-rebac/shared/pkg/go/storage/v1"
)

const chunkSize = 8 * 1024 * 1024 // 8 MB

func (h *handler) RetrieveObject(req *desc.RetrieveObjectRequest, stream desc.DataStorageService_RetrieveObjectServer) error {
	reader, totalSize, err := h.service.RetrieveObject(stream.Context(), req.GetBlobId(), req.GetRangeStart(), req.GetRangeEnd())
	if err != nil {
		return err
	}
	defer reader.Close()

	buf := make([]byte, chunkSize)
	first := true

	for {
		n, readErr := reader.Read(buf)
		if n > 0 {
			resp := &desc.RetrieveObjectResponse{
				Data: buf[:n],
			}
			if first {
				resp.TotalSize = totalSize
				first = false
			}
			if err := stream.Send(resp); err != nil {
				return err
			}
		}
		if readErr != nil {
			break
		}
	}

	return nil
}
