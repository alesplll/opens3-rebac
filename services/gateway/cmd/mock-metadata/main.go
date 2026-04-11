package main

import (
	"context"
	"log"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	metadatav1 "github.com/alesplll/opens3-rebac/shared/pkg/go/metadata/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const defaultGRPCPort = "50052"

type bucketRecord struct {
	id        string
	name      string
	ownerID   string
	createdAt int64
}

type objectVersion struct {
	objectID     string
	versionID    string
	blobID       string
	sizeBytes    int64
	etag         string
	contentType  string
	lastModified int64
}

type objectRecord struct {
	bucketName string
	key        string
	versions   map[string]*objectVersion
	current    string
}

type server struct {
	metadatav1.UnimplementedMetadataServiceServer

	mu         sync.RWMutex
	bucketSeq  uint64
	objectSeq  uint64
	versionSeq uint64
	buckets    map[string]*bucketRecord
	objects    map[string]*objectRecord
}

func newServer() *server {
	return &server{
		buckets: make(map[string]*bucketRecord),
		objects: make(map[string]*objectRecord),
	}
}

func main() {
	lis, err := net.Listen("tcp", ":"+grpcPort())
	if err != nil {
		log.Fatalf("listen failed: %v", err)
	}

	grpcServer := grpc.NewServer()
	metadatav1.RegisterMetadataServiceServer(grpcServer, newServer())

	log.Printf("mock metadata listening on %s", lis.Addr().String())
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("serve failed: %v", err)
	}
}

func grpcPort() string {
	port := strings.TrimSpace(os.Getenv("GRPC_PORT"))
	port = strings.TrimPrefix(port, ":")
	if port == "" {
		return defaultGRPCPort
	}
	return port
}

func (s *server) CreateBucket(_ context.Context, req *metadatav1.CreateBucketRequest) (*metadatav1.CreateBucketResponse, error) {
	name := strings.TrimSpace(req.GetName())
	ownerID := strings.TrimSpace(req.GetOwnerId())
	if name == "" || ownerID == "" {
		return nil, status.Error(codes.InvalidArgument, "name and owner_id are required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.buckets[name]; exists {
		return nil, status.Error(codes.AlreadyExists, "bucket already exists")
	}

	now := nowMillis()
	bucket := &bucketRecord{
		id:        s.nextID("bucket", &s.bucketSeq),
		name:      name,
		ownerID:   ownerID,
		createdAt: now,
	}
	s.buckets[name] = bucket

	return &metadatav1.CreateBucketResponse{BucketId: bucket.id, CreatedAt: bucket.createdAt}, nil
}

func (s *server) DeleteBucket(_ context.Context, req *metadatav1.DeleteBucketRequest) (*metadatav1.DeleteBucketResponse, error) {
	bucketName := strings.TrimSpace(req.GetBucketName())
	if bucketName == "" {
		return nil, status.Error(codes.InvalidArgument, "bucket_name is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.buckets[bucketName]; !exists {
		return nil, status.Error(codes.NotFound, "bucket not found")
	}
	for _, object := range s.objects {
		if object.bucketName == bucketName && object.current != "" {
			return nil, status.Error(codes.FailedPrecondition, "bucket is not empty")
		}
	}

	delete(s.buckets, bucketName)
	return &metadatav1.DeleteBucketResponse{Success: true}, nil
}

func (s *server) GetBucket(_ context.Context, req *metadatav1.GetBucketRequest) (*metadatav1.GetBucketResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	bucket, err := s.requireBucket(req.GetBucketName())
	if err != nil {
		return nil, err
	}

	return &metadatav1.GetBucketResponse{Bucket: toBucketInfo(bucket)}, nil
}

func (s *server) ListBuckets(_ context.Context, req *metadatav1.ListBucketsRequest) (*metadatav1.ListBucketsResponse, error) {
	ownerID := strings.TrimSpace(req.GetOwnerId())
	if ownerID == "" {
		return nil, status.Error(codes.InvalidArgument, "owner_id is required")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]*metadatav1.BucketInfo, 0)
	for _, bucket := range s.buckets {
		if bucket.ownerID == ownerID {
			items = append(items, toBucketInfo(bucket))
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].GetName() < items[j].GetName() })

	return &metadatav1.ListBucketsResponse{Buckets: items}, nil
}

func (s *server) HeadBucket(_ context.Context, req *metadatav1.HeadBucketRequest) (*metadatav1.HeadBucketResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	bucket, err := s.requireBucket(req.GetBucketName())
	if err != nil {
		return nil, err
	}

	return &metadatav1.HeadBucketResponse{Exists: true, BucketId: bucket.id, OwnerId: bucket.ownerID}, nil
}

func (s *server) CreateObjectVersion(_ context.Context, req *metadatav1.CreateObjectVersionRequest) (*metadatav1.CreateObjectVersionResponse, error) {
	bucketName := strings.TrimSpace(req.GetBucketName())
	key := strings.TrimSpace(req.GetKey())
	blobID := strings.TrimSpace(req.GetBlobId())
	etag := strings.Trim(strings.TrimSpace(req.GetEtag()), `"`)
	contentType := strings.TrimSpace(req.GetContentType())
	if bucketName == "" || key == "" || blobID == "" || etag == "" {
		return nil, status.Error(codes.InvalidArgument, "bucket_name, key, blob_id and etag are required")
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := s.requireBucket(bucketName); err != nil {
		return nil, err
	}

	objectKey := bucketObjectKey(bucketName, key)
	object := s.objects[objectKey]
	if object == nil {
		object = &objectRecord{
			bucketName: bucketName,
			key:        key,
			versions:   make(map[string]*objectVersion),
		}
		s.objects[objectKey] = object
	}

	objectID := ""
	if object.current != "" {
		objectID = object.versions[object.current].objectID
	} else {
		objectID = s.nextID("object", &s.objectSeq)
	}

	versionID := s.nextID("version", &s.versionSeq)
	version := &objectVersion{
		objectID:     objectID,
		versionID:    versionID,
		blobID:       blobID,
		sizeBytes:    req.GetSizeBytes(),
		etag:         etag,
		contentType:  contentType,
		lastModified: nowMillis(),
	}
	object.current = versionID
	object.versions[versionID] = version

	return &metadatav1.CreateObjectVersionResponse{ObjectId: version.objectID, VersionId: version.versionID, CreatedAt: version.lastModified}, nil
}

func (s *server) GetObjectMeta(_ context.Context, req *metadatav1.GetObjectMetaRequest) (*metadatav1.GetObjectMetaResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, version, err := s.requireObjectVersion(req.GetBucketName(), req.GetKey(), req.GetVersionId())
	if err != nil {
		return nil, err
	}

	return &metadatav1.GetObjectMetaResponse{
		ObjectId:     version.objectID,
		VersionId:    version.versionID,
		BlobId:       version.blobID,
		SizeBytes:    version.sizeBytes,
		Etag:         version.etag,
		ContentType:  version.contentType,
		LastModified: version.lastModified,
	}, nil
}

func (s *server) DeleteObjectMeta(_ context.Context, req *metadatav1.DeleteObjectMetaRequest) (*metadatav1.DeleteObjectMetaResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	object, version, err := s.requireObjectVersion(req.GetBucketName(), req.GetKey(), "")
	if err != nil {
		return nil, err
	}

	delete(s.objects, bucketObjectKey(object.bucketName, object.key))
	return &metadatav1.DeleteObjectMetaResponse{ObjectId: version.objectID, BlobId: version.blobID, Success: true}, nil
}

func (s *server) ListObjects(_ context.Context, req *metadatav1.ListObjectsRequest) (*metadatav1.ListObjectsResponse, error) {
	bucketName := strings.TrimSpace(req.GetBucketName())
	prefix := req.GetPrefix()
	maxKeys := int(req.GetMaxKeys())
	if maxKeys <= 0 {
		maxKeys = 1000
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, err := s.requireBucket(bucketName); err != nil {
		return nil, err
	}

	items := make([]*metadatav1.ObjectInfo, 0)
	for _, object := range s.objects {
		if object.bucketName != bucketName || object.current == "" {
			continue
		}
		if prefix != "" && !strings.HasPrefix(object.key, prefix) {
			continue
		}
		version := object.versions[object.current]
		items = append(items, &metadatav1.ObjectInfo{
			ObjectId:     version.objectID,
			VersionId:    version.versionID,
			Key:          object.key,
			Etag:         version.etag,
			SizeBytes:    version.sizeBytes,
			ContentType:  version.contentType,
			LastModified: version.lastModified,
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].GetKey() < items[j].GetKey() })

	start := 0
	if token := strings.TrimSpace(req.GetContinuationToken()); token != "" {
		start = sort.Search(len(items), func(i int) bool { return items[i].GetKey() > token })
	}
	if start > len(items) {
		start = len(items)
	}
	end := start + maxKeys
	if end > len(items) {
		end = len(items)
	}

	page := items[start:end]
	nextToken := ""
	isTruncated := end < len(items)
	if isTruncated && len(page) > 0 {
		nextToken = page[len(page)-1].GetKey()
	}

	return &metadatav1.ListObjectsResponse{Objects: page, NextContinuationToken: nextToken, IsTruncated: isTruncated}, nil
}

func (s *server) HealthCheck(_ context.Context, _ *metadatav1.HealthCheckRequest) (*metadatav1.HealthCheckResponse, error) {
	return &metadatav1.HealthCheckResponse{Status: metadatav1.HealthCheckResponse_SERVING}, nil
}

func (s *server) requireBucket(bucketName string) (*bucketRecord, error) {
	bucketName = strings.TrimSpace(bucketName)
	if bucketName == "" {
		return nil, status.Error(codes.InvalidArgument, "bucket_name is required")
	}
	bucket := s.buckets[bucketName]
	if bucket == nil {
		return nil, status.Error(codes.NotFound, "bucket not found")
	}
	return bucket, nil
}

func (s *server) requireObjectVersion(bucketName, key, versionID string) (*objectRecord, *objectVersion, error) {
	if _, err := s.requireBucket(bucketName); err != nil {
		return nil, nil, err
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, nil, status.Error(codes.InvalidArgument, "key is required")
	}
	object := s.objects[bucketObjectKey(bucketName, key)]
	if object == nil || object.current == "" {
		return nil, nil, status.Error(codes.NotFound, "object not found")
	}
	if strings.TrimSpace(versionID) == "" {
		versionID = object.current
	}
	version := object.versions[versionID]
	if version == nil {
		return nil, nil, status.Error(codes.NotFound, "version not found")
	}
	return object, version, nil
}

func (s *server) nextID(prefix string, seq *uint64) string {
	*seq += 1
	return prefix + "-" + strconv.FormatUint(*seq, 10)
}

func toBucketInfo(bucket *bucketRecord) *metadatav1.BucketInfo {
	return &metadatav1.BucketInfo{
		BucketId:  bucket.id,
		Name:      bucket.name,
		OwnerId:   bucket.ownerID,
		CreatedAt: bucket.createdAt,
	}
}

func bucketObjectKey(bucketName, key string) string {
	return strings.TrimSpace(bucketName) + "/" + strings.TrimSpace(key)
}

func nowMillis() int64 {
	return time.Now().UnixMilli()
}
