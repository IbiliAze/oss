package main

import (
	"context"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/IbiliAze/oss/gen/storage/v1"
)

var addr string = "0.0.0.0:50051"

func main() {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))

	if err != nil {
		log.Fatalf("Failed to connect: %v\n", err)

	}
	defer conn.Close()

	c := pb.NewUploaderServiceClient(conn)

	if err := doUpload(context.Background(), c); err != nil {
		log.Fatalf("doUpload failed with: %v\n", err)
	}
}
