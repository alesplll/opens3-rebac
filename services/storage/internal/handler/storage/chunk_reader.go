package storage

import (
	"errors"
	"io"
)

type chunkReader struct {
	recvChunk func() ([]byte, error)
	pending   []byte
	done      bool
}

func (r *chunkReader) Read(p []byte) (int, error) {
	for len(r.pending) == 0 && !r.done {
		chunk, err := r.recvChunk()
		if err != nil {
			if errors.Is(err, io.EOF) {
				r.done = true
				break
			}

			return 0, err
		}

		r.pending = chunk
	}

	if len(r.pending) == 0 {
		return 0, io.EOF
	}

	n := copy(p, r.pending)
	r.pending = r.pending[n:]
	return n, nil
}
