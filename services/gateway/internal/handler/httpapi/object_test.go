package httpapi_test

import (
	"bytes"
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/alesplll/opens3-rebac/services/gateway/internal/handler/httpapi"
	objectservice "github.com/alesplll/opens3-rebac/services/gateway/internal/service/object"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/logger"
	metadatav1 "github.com/alesplll/opens3-rebac/shared/pkg/go/metadata/v1"
	storagev1 "github.com/alesplll/opens3-rebac/shared/pkg/go/storage/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

type fixture struct {
	beforeNextChunk func(context.Context) error
	sync.Mutex
	blobs        map[string][]byte
	meta         map[string]*metadatav1.GetObjectMetaResponse
	storeCount   int
	failCreate   bool
	beforeCommit func(context.Context) error
	onChunk      func()
	failStore    error
}

const fixtureRetrieveChunkSize = 1 << 20

type metadataServer struct {
	metadatav1.UnimplementedMetadataServiceServer
	state *fixture
}

func (s *metadataServer) HeadBucket(_ context.Context, req *metadatav1.HeadBucketRequest) (*metadatav1.HeadBucketResponse, error) {
	return &metadatav1.HeadBucketResponse{Exists: req.GetBucketName() == "fixture"}, nil
}

func (s *metadataServer) CreateObjectVersion(_ context.Context, req *metadatav1.CreateObjectVersionRequest) (*metadatav1.CreateObjectVersionResponse, error) {
	s.state.Lock()
	defer s.state.Unlock()
	if s.state.failCreate {
		return nil, status.Error(codes.Internal, "registration failed")
	}
	if _, ok := s.state.blobs[req.GetBlobId()]; !ok {
		return nil, status.Error(codes.FailedPrecondition, "metadata published before blob commit")
	}
	s.state.meta[req.GetKey()] = &metadatav1.GetObjectMetaResponse{
		BlobId:      req.GetBlobId(),
		VersionId:   "version-1",
		SizeBytes:   req.GetSizeBytes(),
		Etag:        req.GetEtag(),
		ContentType: req.GetContentType(),
	}
	return &metadatav1.CreateObjectVersionResponse{VersionId: "version-1"}, nil
}

func (s *metadataServer) GetObjectMeta(_ context.Context, req *metadatav1.GetObjectMetaRequest) (*metadatav1.GetObjectMetaResponse, error) {
	s.state.Lock()
	defer s.state.Unlock()
	if req.GetBucketName() != "fixture" || s.state.meta[req.GetKey()] == nil {
		return nil, status.Error(codes.NotFound, "object not found")
	}
	return s.state.meta[req.GetKey()], nil
}

type storageServer struct {
	storagev1.UnimplementedDataStorageServiceServer
	state *fixture
}

func (s *storageServer) StoreObject(stream storagev1.DataStorageService_StoreObjectServer) error {
	first, err := stream.Recv()
	if err != nil {
		return err
	}
	if first.GetHeader() == nil {
		return status.Error(codes.InvalidArgument, "missing header")
	}
	if s.state.failStore != nil {
		return s.state.failStore
	}
	var content bytes.Buffer
	for {
		message, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		content.Write(message.GetChunk().GetData())
		if s.state.onChunk != nil {
			s.state.onChunk()
		}
	}
	if first.GetHeader().Size != nil && *first.GetHeader().Size != int64(content.Len()) {
		return status.Error(codes.InvalidArgument, "size mismatch")
	}
	if s.state.beforeCommit != nil {
		if err := s.state.beforeCommit(stream.Context()); err != nil {
			return err
		}
	}
	s.state.Lock()
	s.state.storeCount++
	blobID := fmt.Sprintf("blob-%d", s.state.storeCount)
	s.state.blobs[blobID] = bytes.Clone(content.Bytes())
	s.state.Unlock()
	return stream.SendAndClose(&storagev1.StoreObjectResponse{BlobId: blobID, ChecksumMd5: fmt.Sprintf("%x", md5.Sum(content.Bytes()))})
}

func (s *storageServer) RetrieveObject(req *storagev1.RetrieveObjectRequest, stream storagev1.DataStorageService_RetrieveObjectServer) error {
	s.state.Lock()
	data, ok := s.state.blobs[req.GetBlobId()]
	s.state.Unlock()
	if !ok {
		return status.Error(codes.NotFound, "blob not found")
	}
	if len(data) == 0 {
		return nil
	}
	for offset := 0; offset < len(data); offset += fixtureRetrieveChunkSize {
		if offset > 0 && s.state.beforeNextChunk != nil {
			if err := s.state.beforeNextChunk(stream.Context()); err != nil {
				return err
			}
		}
		end := min(offset+fixtureRetrieveChunkSize, len(data))
		if err := stream.Send(&storagev1.RetrieveObjectResponse{Data: data[offset:end], TotalSize: int64(len(data))}); err != nil {
			return err
		}
	}
	return nil
}

func newGateway(t *testing.T) (*httptest.Server, *fixture) {
	t.Helper()
	state := &fixture{blobs: make(map[string][]byte), meta: make(map[string]*metadatav1.GetObjectMetaResponse)}
	listener := bufconn.Listen(1024 * 1024)
	grpcServer := grpc.NewServer()
	metadatav1.RegisterMetadataServiceServer(grpcServer, &metadataServer{state: state})
	storagev1.RegisterDataStorageServiceServer(grpcServer, &storageServer{state: state})
	go func() { _ = grpcServer.Serve(listener) }()
	t.Cleanup(grpcServer.Stop)
	conn, err := grpc.NewClient("passthrough:///bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
		return listener.Dial()
	}), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	logger.SetNopLogger()
	objects := objectservice.NewService(metadatav1.NewMetadataServiceClient(conn), storagev1.NewDataStorageServiceClient(conn), 1<<20)
	server := httptest.NewServer(httpapi.NewHandler(objects))
	t.Cleanup(server.Close)
	return server, state
}

func TestPutGetRoundTrip(t *testing.T) {
	server, state := newGateway(t)
	for _, tc := range []struct {
		name string
		body string
	}{
		{name: "nested key", body: "hello world"},
		{name: "empty object", body: ""},
		{name: "multiple storage chunks", body: strings.Repeat("0123456789abcdef", 1280*1024)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			key := strings.ReplaceAll(tc.name, " ", "-") + "/file.txt"
			request, err := http.NewRequest(http.MethodPut, server.URL+"/fixture/"+key, strings.NewReader(tc.body))
			if err != nil {
				t.Fatal(err)
			}
			request.Header.Set("Content-Type", "text/plain")
			response, err := server.Client().Do(request)
			if err != nil {
				t.Fatal(err)
			}
			response.Body.Close()
			if response.StatusCode != http.StatusOK || response.Header.Get("X-Object-Version-Id") != "version-1" {
				t.Fatalf("PUT response: status=%d version=%q", response.StatusCode, response.Header.Get("X-Object-Version-Id"))
			}

			got, err := server.Client().Get(server.URL + "/fixture/" + key)
			if err != nil {
				t.Fatal(err)
			}
			defer got.Body.Close()
			body, err := io.ReadAll(got.Body)
			if err != nil {
				t.Fatal(err)
			}
			if got.StatusCode != http.StatusOK || string(body) != tc.body || got.Header.Get("Content-Type") != "text/plain" {
				t.Fatalf("GET response: status=%d body length=%d type=%q", got.StatusCode, len(body), got.Header.Get("Content-Type"))
			}
			if got.Header.Get("ETag") != response.Header.Get("ETag") {
				t.Fatalf("ETag changed between PUT and GET")
			}
		})
	}
	state.Lock()
	defer state.Unlock()
	if state.storeCount != 3 {
		t.Fatalf("stored blobs = %d, want 3", state.storeCount)
	}
}

func TestMissingBucketDoesNotWriteBlob(t *testing.T) {
	server, state := newGateway(t)
	request, err := http.NewRequest(http.MethodPut, server.URL+"/missing/key", strings.NewReader("data"))
	if err != nil {
		t.Fatal(err)
	}
	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", response.StatusCode)
	}
	state.Lock()
	defer state.Unlock()
	if state.storeCount != 0 {
		t.Fatalf("stored blobs = %d, want 0", state.storeCount)
	}
}

func TestMetadataFailureDoesNotReturnSuccess(t *testing.T) {
	server, state := newGateway(t)
	state.failCreate = true
	request, err := http.NewRequest(http.MethodPut, server.URL+"/fixture/key", strings.NewReader("data"))
	if err != nil {
		t.Fatal(err)
	}
	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502", response.StatusCode)
	}
	state.Lock()
	defer state.Unlock()
	if state.storeCount != 1 || len(state.meta) != 0 {
		t.Fatalf("unexpected state after Metadata failure: blobs=%d versions=%d", state.storeCount, len(state.meta))
	}
}

func TestHealthEndpoint(t *testing.T) {
	server, _ := newGateway(t)
	response, err := server.Client().Get(server.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.StatusCode)
	}
}
