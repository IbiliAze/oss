package grpcserver

import (
	"context"
	"encoding/base64"
	"strings"

	"github.com/IbiliAze/vaultlet/internal/app"
	"github.com/IbiliAze/vaultlet/internal/config"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// errUnauthenticated is the one answer every authentication failure gets.
// Distinguishing a missing header from an unknown user or a wrong password
// would tell an unauthenticated caller which usernames exist.
var errUnauthenticated = status.Error(codes.Unauthenticated, "invalid credentials")

// dummyHash is compared against when the username is unknown, so that a
// failed lookup costs the same bcrypt work as a wrong password and the two
// cannot be told apart by timing. The preimage is throwaway; even a match
// is rejected because the user is not in the map.
const dummyHash = "$2y$10$sSy2zu.lkhMSh0tzGrFD2eCyfeqkxmpVwEFCDjlm6Nb9Swq.u7sgO"

// usersByName indexes the configured users for the per-RPC lookup.
func usersByName(users []config.UserConfig) map[string]string {
	m := make(map[string]string, len(users))
	for _, u := range users {
		m[u.Username] = u.PasswordHash
	}
	return m
}

// authenticate resolves the RPC's principal from its "authorization: Basic"
// metadata, verifying the password against the configured bcrypt hash.
func (s *Server) authenticate(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", errUnauthenticated
	}

	auth := md.Get("authorization")
	if len(auth) == 0 {
		return "", errUnauthenticated
	}
	token, ok := strings.CutPrefix(auth[0], "Basic ")
	if !ok {
		return "", errUnauthenticated
	}
	raw, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return "", errUnauthenticated
	}
	username, password, ok := strings.Cut(string(raw), ":")
	if !ok {
		return "", errUnauthenticated
	}

	hash, known := s.users[username]
	if !known {
		hash = dummyHash
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil || !known {
		return "", errUnauthenticated
	}
	return username, nil
}

func (s *Server) unaryAuth() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		principal, err := s.authenticate(ctx)
		if err != nil {
			return nil, err
		}
		return handler(app.WithPrincipal(ctx, principal), req)
	}
}

func (s *Server) streamAuth() grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		principal, err := s.authenticate(ss.Context())
		if err != nil {
			return err
		}
		return handler(srv, authedStream{ServerStream: ss, ctx: app.WithPrincipal(ss.Context(), principal)})
	}
}

// authedStream overrides Context so streaming handlers see the principal.
type authedStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s authedStream) Context() context.Context { return s.ctx }
