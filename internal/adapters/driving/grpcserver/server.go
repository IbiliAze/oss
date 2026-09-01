package grpcserver

import (
	"context"
	"log/slog"
	"net"
	"time"

	vaultletv1 "github.com/IbiliAze/vaultlet/api/gen/vaultlet/v1"
	"github.com/IbiliAze/vaultlet/internal/ports"
	"google.golang.org/grpc"
)

// shutdownTimeout bounds GracefulStop's drain; it must stay under the
// platform's SIGTERM-to-SIGKILL grace period (30s on Kubernetes).
const shutdownTimeout = 10 * time.Second

type Server struct {
	vaultletv1.UnimplementedSecretServiceServer
	store ports.SecretStore
	grpc  *grpc.Server
}

func New(store ports.SecretStore) *Server {
	s := &Server{store: store}
	s.grpc = grpc.NewServer()
	vaultletv1.RegisterSecretServiceServer(s.grpc, s)
	return s
}

func (s *Server) Listen(ctx context.Context, port string) error {
	listener, err := net.Listen("tcp", port)
	if err != nil {
		return err
	}

	slog.Info("Listening on " + port)

	errCh := make(chan error, 1)
	go func() { errCh <- s.grpc.Serve(listener) }()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	slog.Info("Shutting down, draining in-flight RPCs")

	done := make(chan struct{})
	go func() {
		s.grpc.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(shutdownTimeout):
		slog.Warn("Drain deadline exceeded, forcing stop")
		s.grpc.Stop()
	}

	return nil
}

var _ vaultletv1.SecretServiceServer = (*Server)(nil)
