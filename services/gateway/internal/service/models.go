package service

import (
	"io"
	"time"
)

type CreateBucketRequest struct {
	UserID string
	Bucket string
}

type CreateBucketResponse struct {
	BucketID  string
	CreatedAt time.Time
}

type DeleteBucketRequest struct {
	UserID string
	Bucket string
}

type ListBucketsRequest struct {
	UserID string
}

type BucketInfo struct {
	Name      string
	CreatedAt time.Time
}

type ListBucketsResponse struct {
	Buckets []BucketInfo
}

type HeadBucketRequest struct {
	UserID string
	Bucket string
}

type PutObjectRequest struct {
	UserID      string
	Bucket      string
	Key         string
	Body        io.Reader
	Size        int64
	ContentType string
}

type PutObjectResponse struct {
	ETag      string
	VersionID string
}

type ByteRange struct {
	Start int64
	End   int64
}

type GetObjectRequest struct {
	UserID    string
	Bucket    string
	Key       string
	VersionID string
	Range     *ByteRange
	Writer    io.Writer
}

type GetObjectResponse struct {
	ContentType   string
	ContentLength int64
	ETag          string
	VersionID     string
	LastModified  time.Time
	Range         *ByteRange
	TotalSize     int64
}

type HeadObjectRequest struct {
	UserID    string
	Bucket    string
	Key       string
	VersionID string
}

type HeadObjectResponse struct {
	ContentType   string
	ContentLength int64
	ETag          string
	VersionID     string
	LastModified  time.Time
}

type DeleteObjectRequest struct {
	UserID string
	Bucket string
	Key    string
}

type ListObjectsRequest struct {
	UserID             string
	Bucket             string
	Prefix             string
	ContinuationToken  string
	MaxKeys            int32
}

type ObjectInfo struct {
	Key          string
	ETag         string
	Size         int64
	ContentType  string
	LastModified time.Time
	VersionID    string
}

type ListObjectsResponse struct {
	Objects               []ObjectInfo
	NextContinuationToken string
	IsTruncated           bool
}

type CreateMultipartUploadRequest struct {
	UserID      string
	Bucket      string
	Key         string
	ContentType string
}

type CreateMultipartUploadResponse struct {
	UploadID string
}

type UploadPartRequest struct {
	UserID     string
	Bucket     string
	Key        string
	UploadID   string
	PartNumber int32
	Body       io.Reader
}

type UploadPartResponse struct {
	ETag string
}

type CompletedPart struct {
	PartNumber int32
	ETag       string
}

type CompleteMultipartUploadRequest struct {
	UserID   string
	Bucket   string
	Key      string
	UploadID string
	Parts    []CompletedPart
}

type CompleteMultipartUploadResponse struct {
	ETag      string
	VersionID string
}

type AbortMultipartUploadRequest struct {
	UserID   string
	Bucket   string
	Key      string
	UploadID string
}
