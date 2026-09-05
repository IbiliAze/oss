package grpcserver

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

func logRPC(ctx context.Context, method string, started time.Time, err error) {
	slog.InfoContext(ctx, "rpc completed",
		"method", method,
		"duration_ms", float64(time.Since(started))/float64(time.Millisecond),
		"status", status.Code(err).String(),
	)
}

func unaryLogging() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		started := time.Now()
		resp, err := handler(ctx, req)
		logRPC(ctx, info.FullMethod, started, err)
		return resp, err
	}
}

func streamLogging() grpc.StreamServerInterceptor {
	return func(
		srv any,
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		started := time.Now()
		err := handler(srv, ss)
		logRPC(ss.Context(), info.FullMethod, started, err)
		return err
	}
}
