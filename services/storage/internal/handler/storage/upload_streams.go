package storage

import (
	"io"

	desc "github.com/alesplll/opens3-rebac/shared/pkg/go/storage/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type storeObjectStreamPayload struct {
	size        int64
	contentType string
	reader      io.Reader
}

func readStoreObjectStream(stream desc.DataStorageService_StoreObjectServer) (*storeObjectStreamPayload, error) {
	firstReq, err := recvFirstMessage(stream.Recv)
	if err != nil {
		return nil, err
	}

	reader := newChunkStreamReader(firstReq.GetData(), func() ([]byte, error) {
		req, err := stream.Recv()
		if err != nil {
			return nil, err
		}
		if req.GetSize() != 0 {
			return nil, status.Error(codes.InvalidArgument, "size is only allowed in the first message")
		}
		if req.GetContentType() != "" {
			return nil, status.Error(codes.InvalidArgument, "content_type is only allowed in the first message")
		}

		return req.GetData(), nil
	})

	return &storeObjectStreamPayload{
		size:        firstReq.GetSize(),
		contentType: firstReq.GetContentType(),
		reader:      reader,
	}, nil
}

type uploadPartStreamPayload struct {
	uploadID   string
	partNumber int32
	reader     io.Reader
}

func readUploadPartStream(stream desc.DataStorageService_UploadPartServer) (*uploadPartStreamPayload, error) {
	firstReq, err := recvFirstMessage(stream.Recv)
	if err != nil {
		return nil, err
	}
	if firstReq.GetUploadId() == "" {
		return nil, status.Error(codes.InvalidArgument, "upload_id is required in the first message")
	}
	if firstReq.GetPartNumber() == 0 {
		return nil, status.Error(codes.InvalidArgument, "part_number is required in the first message")
	}

	reader := newChunkStreamReader(firstReq.GetData(), func() ([]byte, error) {
		req, err := stream.Recv()
		if err != nil {
			return nil, err
		}
		if req.GetUploadId() != "" {
			return nil, status.Error(codes.InvalidArgument, "upload_id is only allowed in the first message")
		}
		if req.GetPartNumber() != 0 {
			return nil, status.Error(codes.InvalidArgument, "part_number is only allowed in the first message")
		}

		return req.GetData(), nil
	})

	return &uploadPartStreamPayload{
		uploadID:   firstReq.GetUploadId(),
		partNumber: firstReq.GetPartNumber(),
		reader:     reader,
	}, nil
}
