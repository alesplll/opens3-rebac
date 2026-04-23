package model

import "time"

type Object struct {
	ID               string
	BucketID         string
	Key              string
	CurrentVersionID *string
}

type Version struct {
	ID          string
	ObjectID    string
	BlobID      string
	SizeBytes   int64
	Etag        string
	ContentType string
	CreatedAt   time.Time
	IsDeleted   bool
}

// ObjectMeta combines object and version data for GetObjectMeta response.
type ObjectMeta struct {
	ObjectID     string
	VersionID    string
	BlobID       string
	SizeBytes    int64
	Etag         string
	ContentType  string
	LastModified time.Time
}

// ObjectListItem is used for ListObjects response.
type ObjectListItem struct {
	ObjectID     string
	VersionID    string
	Key          string
	Etag         string
	SizeBytes    int64
	ContentType  string
	LastModified time.Time
}
