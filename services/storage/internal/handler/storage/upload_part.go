package storage

import (
	desc "github.com/alesplll/opens3-rebac/shared/pkg/go/storage/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
		if req.GetUploadId() != "" && req.GetUploadId() != firstReq.GetUploadId() {
			return nil, status.Errorf(
				codes.InvalidArgument,
				"upload_id changed within stream: got %q, want %q",
				req.GetUploadId(),
				firstReq.GetUploadId(),
			)
		}
		if req.GetPartNumber() != 0 && req.GetPartNumber() != firstReq.GetPartNumber() {
			return nil, status.Errorf(
				codes.InvalidArgument,
				"part_number changed within stream: got %d, want %d",
				req.GetPartNumber(),
				firstReq.GetPartNumber(),
			)
		}
		if req.GetUploadId() == "" && req.GetPartNumber() != 0 {
			return nil, status.Error(codes.InvalidArgument, "part_number requires upload_id in chunk metadata")
		}
		if req.GetUploadId() != "" && req.GetPartNumber() == 0 {
			return nil, status.Error(codes.InvalidArgument, "upload_id requires part_number in chunk metadata")
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
