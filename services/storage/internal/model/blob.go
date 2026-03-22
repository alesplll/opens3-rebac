package model

type BlobMeta struct {
	BlobID      string
	ChecksumMD5 string
	SizeBytes   int64
	ContentType string
}

type PartInfo struct {
	PartNumber  int32
	ChecksumMD5 string
}
