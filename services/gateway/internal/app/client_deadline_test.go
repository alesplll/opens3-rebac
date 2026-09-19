package app

import (
	"context"
	"io"
	"net"
	"testing"
	"time"

	objectservice "github.com/alesplll/opens3-rebac/services/gateway/internal/service/object"
	metadatav1 "github.com/alesplll/opens3-rebac/shared/pkg/go/metadata/v1"
	storagev1 "github.com/alesplll/opens3-rebac/shared/pkg/go/storage/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

// Real gRPC peers are needed here to observe context propagation across transport.
type deadlineMetadata struct {
	metadatav1.UnimplementedMetadataServiceServer
	deadlines chan time.Time
}

func (s deadlineMetadata) GetObjectMeta(ctx context.Context, _ *metadatav1.GetObjectMetaRequest) (*metadatav1.GetObjectMetaResponse, error) {
	deadline, _ := ctx.Deadline()
	s.deadlines <- deadline
	return &metadatav1.GetObjectMetaResponse{BlobId: "blob", SizeBytes: 2}, nil
}

type deadlineStorage struct {
	storagev1.UnimplementedDataStorageServiceServer
	deadlines chan time.Time
	stopped   chan struct{}
}

func (s deadlineStorage) RetrieveObject(_ *storagev1.RetrieveObjectRequest, stream storagev1.DataStorageService_RetrieveObjectServer) error {
	deadline, _ := stream.Context().Deadline()
	s.deadlines <- deadline
	defer close(s.stopped)
	if err := stream.Send(&storagev1.RetrieveObjectResponse{Data: []byte("x")}); err != nil {
		return err
	}
	<-stream.Context().Done()
	return stream.Context().Err()
}

func TestDeadlineReachesMetadataAndStorage(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	deadlines, stopped := make(chan time.Time, 2), make(chan struct{})
	server := grpc.NewServer()
	metadatav1.RegisterMetadataServiceServer(server, deadlineMetadata{deadlines: deadlines})
	storagev1.RegisterDataStorageServiceServer(server, deadlineStorage{deadlines: deadlines, stopped: stopped})
	go server.Serve(listener)
	defer server.Stop()
	conn, err := grpc.NewClient(listener.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	objects := objectservice.NewService(metadatav1.NewMetadataServiceClient(conn), storagev1.NewDataStorageServiceClient(conn), 1<<20)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	expected, _ := ctx.Deadline()
	result, err := objects.Get(ctx, "fixture", "key")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		select {
		case actual := <-deadlines:
			if delta := actual.Sub(expected); actual.IsZero() || delta < -100*time.Millisecond || delta > 100*time.Millisecond {
				t.Fatalf("downstream deadline %v differs from %v", actual, expected)
			}
		case <-ctx.Done():
			t.Fatal("downstream did not receive context deadline")
		}
	}
	_, err = io.Copy(io.Discard, result.Body)
	if status.Code(err) != codes.DeadlineExceeded {
		t.Fatalf("read error %v, want DeadlineExceeded", err)
	}
	select {
	case <-stopped:
	case <-time.After(3 * time.Second):
		t.Fatal("Storage continued after deadline")
	}
}
