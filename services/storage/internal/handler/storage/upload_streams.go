package storage

import (
	"io"

	desc "github.com/alesplll/opens3-rebac/shared/pkg/go/storage/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type storeObjectStreamPayload struct {
	size        *int64
	contentType string
	reader      io.Reader
}

func readStoreObjectStream(stream desc.DataStorageService_StoreObjectServer) (*storeObjectStreamPayload, error) {
	firstReq, err := recvFirstMessage(stream.Recv)
	if err != nil {
		return nil, err
	}
	firstHeader := firstReq.GetHeader()
	if firstHeader == nil {
		return nil, status.Error(codes.InvalidArgument, "first message must be store_object header")
	}

	reader := newChunkStreamReader(firstHeader.GetData(), func() ([]byte, error) {
		req, err := stream.Recv()
		if err != nil {
			return nil, err
		}
		chunk := req.GetChunk()
		if chunk == nil {
			return nil, status.Error(codes.InvalidArgument, "messages after the first must be store_object chunks")
		}
		return chunk.GetData(), nil
	})

	return &storeObjectStreamPayload{
		size:        firstHeader.Size,
		contentType: firstHeader.GetContentType(),
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
	firstHeader := firstReq.GetHeader()
	if firstHeader == nil {
		return nil, status.Error(codes.InvalidArgument, "first message must be upload_part header")
	}
	if firstHeader.GetUploadId() == "" {
		return nil, status.Error(codes.InvalidArgument, "upload_id is required in the first message")
	}
	if firstHeader.GetPartNumber() == 0 {
		return nil, status.Error(codes.InvalidArgument, "part_number is required in the first message")
	}

	reader := newChunkStreamReader(firstHeader.GetData(), func() ([]byte, error) {
		req, err := stream.Recv()
		if err != nil {
			return nil, err
		}
		chunk := req.GetChunk()
		if chunk == nil {
			return nil, status.Error(codes.InvalidArgument, "messages after the first must be upload_part chunks")
		}

		return chunk.GetData(), nil
	})

	return &uploadPartStreamPayload{
		uploadID:   firstHeader.GetUploadId(),
		partNumber: firstHeader.GetPartNumber(),
		reader:     reader,
	}, nil
}
