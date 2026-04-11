package main

import (
	"log"
	"net"

	metadatahandler "github.com/alesplll/opens3-rebac/services/metadata/internal/handler/metadata"
	metadatav1 "github.com/alesplll/opens3-rebac/shared/pkg/go/metadata/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const grpcPort = ":50052"

func main() {
	lis, err := net.Listen("tcp", grpcPort)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", grpcPort, err)
	}

	server := grpc.NewServer()
	metadatav1.RegisterMetadataServiceServer(server, metadatahandler.NewHandler())
	reflection.Register(server)

	log.Printf("metadata gRPC stub listens on %s", grpcPort)

	if err := server.Serve(lis); err != nil {
		log.Fatalf("failed to serve gRPC server: %v", err)
	}
}
