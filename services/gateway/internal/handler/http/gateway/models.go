package gateway

import "encoding/xml"

type listAllMyBucketsResult struct {
	XMLName xml.Name         `xml:"ListAllMyBucketsResult"`
	Buckets []bucketListItem `xml:"Buckets>Bucket"`
}

type bucketListItem struct {
	Name         string `xml:"Name"`
	CreationDate string `xml:"CreationDate"`
}

type listBucketResult struct {
	XMLName               xml.Name         `xml:"ListBucketResult"`
	Name                  string           `xml:"Name"`
	Prefix                string           `xml:"Prefix,omitempty"`
	ContinuationToken     string           `xml:"ContinuationToken,omitempty"`
	NextContinuationToken string           `xml:"NextContinuationToken,omitempty"`
	MaxKeys               int32            `xml:"MaxKeys"`
	IsTruncated           bool             `xml:"IsTruncated"`
	Contents              []objectListItem `xml:"Contents"`
}

type objectListItem struct {
	Key          string `xml:"Key"`
	LastModified string `xml:"LastModified"`
	ETag         string `xml:"ETag"`
	Size         int64  `xml:"Size"`
}

type initiateMultipartUploadResult struct {
	XMLName  xml.Name `xml:"InitiateMultipartUploadResult"`
	Bucket   string   `xml:"Bucket"`
	Key      string   `xml:"Key"`
	UploadID string   `xml:"UploadId"`
}

type completeMultipartUploadXML struct {
	XMLName xml.Name                  `xml:"CompleteMultipartUpload"`
	Parts   []completeMultipartPartXML `xml:"Part"`
}

type completeMultipartPartXML struct {
	PartNumber int32  `xml:"PartNumber"`
	ETag       string `xml:"ETag"`
}

type completeMultipartUploadResult struct {
	XMLName   xml.Name `xml:"CompleteMultipartUploadResult"`
	Bucket    string   `xml:"Bucket"`
	Key       string   `xml:"Key"`
	ETag      string   `xml:"ETag"`
	VersionID string   `xml:"VersionId,omitempty"`
}

type errorResponse struct {
	XMLName   xml.Name `xml:"Error"`
	Code      string   `xml:"Code"`
	Message   string   `xml:"Message"`
	Resource  string   `xml:"Resource"`
	RequestID string   `xml:"RequestId"`
}
