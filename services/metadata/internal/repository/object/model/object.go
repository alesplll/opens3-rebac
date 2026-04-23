package model

import "time"

type ObjectMeta struct {
	ObjectID     string    `db:"object_id"`
	VersionID    string    `db:"version_id"`
	BlobID       string    `db:"blob_id"`
	SizeBytes    int64     `db:"size_bytes"`
	Etag         string    `db:"etag"`
	ContentType  string    `db:"content_type"`
	LastModified time.Time `db:"last_modified"`
}

type ObjectListItem struct {
	ObjectID     string    `db:"object_id"`
	VersionID    string    `db:"version_id"`
	Key          string    `db:"key"`
	Etag         string    `db:"etag"`
	SizeBytes    int64     `db:"size_bytes"`
	ContentType  string    `db:"content_type"`
	LastModified time.Time `db:"last_modified"`
}
