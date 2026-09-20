package service

import (
	"context"
	"errors"
	"io"
)

var (
	ErrBucketNotFound     = errors.New("bucket not found")
	ErrBodyRead           = errors.New("could not read request body")
	ErrEmptyStorageStream = errors.New("storage returned an empty stream")
)

type PutResult struct {
	ETag      string
	VersionID string
}

type GetResult struct {
	Body        io.Reader
	Size        int64
	ETag        string
	VersionID   string
	ContentType string
}

type ObjectService interface {
	Put(ctx context.Context, bucket, key string, body io.Reader, size *int64, contentType string) (PutResult, error)
	Get(ctx context.Context, bucket, key string) (GetResult, error)
}
