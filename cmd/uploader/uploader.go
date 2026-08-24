package main

import (
	"io"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"

	pb "github.com/IbiliAze/oss/gen/storage/v1"
)

func (s *Server) Upload(stream grpc.ClientStreamingServer[pb.UploadRequest, pb.UploadResponse]) error {
	ctx := stream.Context()

	// Who's calling — TCP address of the client.
	if p, ok := peer.FromContext(ctx); ok {
		log.Printf("upload from %s", p.Addr)
	}

	// Request headers, incl. anything the client attached itself.
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		log.Printf("user-agent=%v", md.Get("user-agent"))
		log.Printf("metadata=%v", md)
	}

	// Full RPC name, e.g. /storage.v1.UploaderService/Upload
	if m, ok := grpc.Method(ctx); ok {
		log.Printf("method=%s", m)
	}

	// Only set if the client passed a deadline.
	if dl, ok := ctx.Deadline(); ok {
		log.Printf("deadline=%s", dl)
	}

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		_ = req
	}
	return stream.SendAndClose(&pb.UploadResponse{
		ObjectId: "",
		Sha256:   "",
	})

}
