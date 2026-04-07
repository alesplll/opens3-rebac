package storage

import (
	"errors"
	"io"
)

func recvFirstMessage[T any](recv func() (T, error)) (T, error) {
	firstReq, err := recv()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return firstReq, io.ErrUnexpectedEOF
		}
		return firstReq, err
	}

	return firstReq, nil
}

func newChunkStreamReader(initial []byte, recvChunk func() ([]byte, error)) io.Reader {
	return &chunkReader{
		recvChunk: recvChunk,
		pending:   initial,
	}
}
