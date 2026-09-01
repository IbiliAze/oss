package grpcserver

import (
	"context"
	"crypto/tls"
	"log/slog"
	"net"
	"time"

	vaultletv1 "github.com/IbiliAze/vaultlet/api/gen/vaultlet/v1"
	"github.com/IbiliAze/vaultlet/internal/config"
	"github.com/IbiliAze/vaultlet/internal/ports"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// shutdownTimeout bounds GracefulStop's drain; it must stay under the
// platform's SIGTERM-to-SIGKILL grace period (30s on Kubernetes).
const shutdownTimeout = 10 * time.Second

type Server struct {
	vaultletv1.UnimplementedSecretServiceServer
	store ports.SecretStore
	grpc  *grpc.Server
	users map[string]string
}

func New(store ports.SecretStore, tlsCfg config.TLSConfig, auth config.AuthConfig) (*Server, error) {
	s := &Server{store: store}

	cert, err := tls.LoadX509KeyPair(tlsCfg.CertFile, tlsCfg.KeyFile)
	if err != nil {
		return nil, err
	}
	// TLS 1.3 floor: the same policy the CLI pins on its side of the dial.
	creds := credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS13,
	})
	s.users = usersByName(auth.Users)
	s.grpc = grpc.NewServer(
		grpc.Creds(creds),
		grpc.ChainUnaryInterceptor(s.unaryAuth()),
		grpc.ChainStreamInterceptor(s.streamAuth()),
	)
	vaultletv1.RegisterSecretServiceServer(s.grpc, s)

	return s, nil
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
