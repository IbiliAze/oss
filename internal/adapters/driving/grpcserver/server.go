package grpcserver

import (
	"log/slog"
	"net"

	vaultletv1 "github.com/IbiliAze/vaultlet/api/gen/vaultlet/v1"
	"github.com/IbiliAze/vaultlet/internal/ports"
	"google.golang.org/grpc"
)

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

func (s *Server) Listen(port string) {
	listener, err := net.Listen("tcp", port)
	if err != nil {
		slog.Error(err.Error())
	}

	slog.Info("Listening on " + port)

	if err := s.grpc.Serve(listener); err != nil {
		slog.Error(err.Error())
	}
}

var _ vaultletv1.SecretServiceServer = (*Server)(nil)
