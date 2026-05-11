package model

import "time"

type ObjectStatus string

const (
	ObjectStatusActive    ObjectStatus = "active"
	ObjectStatusUploading ObjectStatus = "uploading"
	ObjectStatusDeleting  ObjectStatus = "deleting"
)

type VersionKind string

const (
	VersionKindBlob         VersionKind = "blob"
	VersionKindDeleteMarker VersionKind = "delete_marker"
)

type VersionState string

const (
	VersionStatePending   VersionState = "pending"
	VersionStateCommitted VersionState = "committed"
	VersionStateAborted   VersionState = "aborted"
)

type Object struct {
	ID               string
	BucketID         string
	Key              string
	CurrentVersionID *string
	PendingVersionID *string
	Status           ObjectStatus
}

type Version struct {
	ID            string
	ObjectID      string
	BlobID        *string
	SizeBytes     int64
	Etag          string
	ContentType   string
	VersionNumber int64
	Kind          VersionKind
	State         VersionState
	CreatedAt     time.Time
	CommittedAt   *time.Time
	AbortedAt     *time.Time
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
