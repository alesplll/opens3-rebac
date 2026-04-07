package component

import (
	"net"
	"os"
	"testing"

	storageHandler "github.com/alesplll/opens3-rebac/services/storage/internal/handler/storage"
	storageRepo "github.com/alesplll/opens3-rebac/services/storage/internal/repository/storage"
	storageService "github.com/alesplll/opens3-rebac/services/storage/internal/service/storage"
	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/logger"
	validationInterceptor "github.com/alesplll/opens3-rebac/shared/pkg/go-kit/middleware/validation"
	desc "github.com/alesplll/opens3-rebac/shared/pkg/go/storage/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var client desc.DataStorageServiceClient

func TestMain(m *testing.M) {
	tmpDataDir, err := os.MkdirTemp("", "component-storage-*")
	if err != nil {
		panic(err)
	}

	tmpMultipartDir, err := os.MkdirTemp("", "component-multipart-*")
	if err != nil {
		panic(err)
	}

	cfg := testStorageConfig{dataDir: tmpDataDir, multipartDir: tmpMultipartDir}
	repo := storageRepo.NewRepository(cfg)
	svc := storageService.NewService(repo)
	h := storageHandler.NewHandler(svc)

	const maxMsgSize = 16 * 1024 * 1024 // 16 MB — must exceed handler's 8 MB chunk size

	nopLog := &logger.NoopLogger{}
	srv := grpc.NewServer(
		grpc.MaxRecvMsgSize(maxMsgSize),
		grpc.MaxSendMsgSize(maxMsgSize),
		grpc.UnaryInterceptor(validationInterceptor.ErrorCodesUnaryInterceptor(nopLog)),
		grpc.StreamInterceptor(validationInterceptor.ErrorCodesStreamInterceptor(nopLog)),
	)
	desc.RegisterDataStorageServiceServer(srv, h)

	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		panic(err)
	}

	go func() { _ = srv.Serve(lis) }()

	conn, err := grpc.NewClient(
		lis.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(maxMsgSize),
			grpc.MaxCallSendMsgSize(maxMsgSize),
		),
	)
	if err != nil {
		panic(err)
	}

	client = desc.NewDataStorageServiceClient(conn)

	code := m.Run()

	_ = conn.Close()
	srv.GracefulStop()
	_ = os.RemoveAll(tmpDataDir)
	_ = os.RemoveAll(tmpMultipartDir)

	os.Exit(code)
}
