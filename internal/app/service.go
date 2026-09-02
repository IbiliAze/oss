package app

import (
	"context"
	"errors"

	"github.com/IbiliAze/vaultlet/internal/domain"
	"github.com/IbiliAze/vaultlet/internal/ports"
)

var (
	ErrPermissionDenied = errors.New("permission denied")
)

type Service struct {
	store  ports.SecretStore
	policy Policy
}

func NewService(store ports.SecretStore, policy Policy) *Service {
	return &Service{
		store:  store,
		policy: policy,
	}
}

func (s *Service) Get(ctx context.Context, key domain.Key) (domain.Secret, error) {
	principal, ok := PrincipalFromContext(ctx)
	if !ok {
		return domain.Secret{}, ErrPermissionDenied
	}
	if !s.policy.allows(principal, ActionGet, key.Namespace()) {
		return domain.Secret{}, ErrPermissionDenied
	}
	return s.store.Get(ctx, key)
}

func (s *Service) Put(ctx context.Context, key domain.Key, value []byte) (domain.SecretMeta, error) {
	principal, ok := PrincipalFromContext(ctx)
	if !ok {
		return domain.SecretMeta{}, ErrPermissionDenied
	}
	if !s.policy.allows(principal, ActionPut, key.Namespace()) {
		return domain.SecretMeta{}, ErrPermissionDenied
	}
	return s.store.Put(ctx, key, value)
}

func (s *Service) List(ctx context.Context, ns domain.Namespace) ([]domain.SecretMeta, error) {
	principal, ok := PrincipalFromContext(ctx)
	if !ok {
		return nil, ErrPermissionDenied
	}
	if !s.policy.allows(principal, ActionList, ns) {
		return nil, ErrPermissionDenied
	}
	return s.store.List(ctx, ns)
}

func (s *Service) Delete(ctx context.Context, key domain.Key) error {
	principal, ok := PrincipalFromContext(ctx)
	if !ok {
		return ErrPermissionDenied
	}
	if !s.policy.allows(principal, ActionDelete, key.Namespace()) {
		return ErrPermissionDenied
	}
	return s.store.Delete(ctx, key)
}

var _ ports.SecretStore = (*Service)(nil)
