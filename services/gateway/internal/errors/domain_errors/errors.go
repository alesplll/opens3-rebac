package domain_errors

import "errors"

var (
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden")
	ErrInvalidRange       = errors.New("invalid range")
	ErrInvalidRequest     = errors.New("invalid request")
	ErrBucketAlreadyExist = errors.New("bucket already exists")
	ErrBucketNotFound     = errors.New("bucket not found")
	ErrObjectNotFound     = errors.New("object not found")
	ErrBucketNotEmpty     = errors.New("bucket not empty")
	ErrServiceUnavailable = errors.New("service unavailable")
	ErrInsufficientSpace  = errors.New("insufficient storage")
)
