package main

import (
	"context"
	"fmt"
	"log"

	pb "github.com/IbiliAze/oss/gen/storage/v1"
)

func doUpload(ctx context.Context, c pb.UploaderServiceClient) error {
	log.Printf("doUpload was invoked")

	stream, err := c.Upload(ctx)
	if err != nil {
		return fmt.Errorf("open upload stream: %w", err)
	}

	if err := stream.Send(&pb.UploadRequest{
		Payload: &pb.UploadRequest_Header{
			Header: &pb.UploadHeader{
				Filename:    "file_1.txt",
				ContentType: "application/json",
				SizeBytes:   123,
			},
		},
	}); err != nil {
		return fmt.Errorf("send header: %w", err)
	}

	resp, err := stream.CloseAndRecv()
	if err != nil {
		return fmt.Errorf("close and recv: %w", err)
	}

	log.Printf("uploaded: object_id=%s sha256=%s", resp.ObjectId, resp.Sha256)
	return nil
}
