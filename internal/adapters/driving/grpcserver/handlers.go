package grpcserver

import (
	"context"
	"errors"
	"log/slog"

	vaultletv1 "github.com/IbiliAze/vaultlet/api/gen/vaultlet/v1"
	"github.com/IbiliAze/vaultlet/internal/domain"
	"github.com/IbiliAze/vaultlet/internal/ports"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *Server) GetSecret(ctx context.Context, req *vaultletv1.GetSecretRequest) (*vaultletv1.GetSecretResponse, error) {
	res := &vaultletv1.GetSecretResponse{}

	key, err := domain.ParseKey(req.Key)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	secret, err := s.store.Get(ctx, key)
	if err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "no secret at %s", key)
		}
		slog.ErrorContext(ctx, "get secret", "key", key, "err", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	res.Secret = &vaultletv1.Secret{
		Value: secret.Value(),
		Meta: &vaultletv1.SecretMeta{
			Key:       secret.Meta().Key.String(),
			Version:   secret.Meta().Version.String(),
			CreatedAt: timestamppb.New(secret.Meta().CreatedAt),
		},
	}

	return res, nil
}

func (s *Server) PutSecret(context.Context, *vaultletv1.PutSecretRequest) (*vaultletv1.PutSecretResponse, error) {
}

func (s *Server) ListSecrets(context.Context, *vaultletv1.ListSecretsRequest) (*vaultletv1.ListSecretsResponse, error) {
}

func (s *Server) DeleteSecret(context.Context, *vaultletv1.DeleteSecretRequest) (*vaultletv1.DeleteSecretResponse, error) {
}
