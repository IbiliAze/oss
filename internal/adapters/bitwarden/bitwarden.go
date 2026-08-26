package bitwarden

import (
	"context"

	"github.com/IbiliAze/vaultlet/internal/domain"
	"github.com/IbiliAze/vaultlet/internal/ports"

	"github.com/bitwarden/sdk-go/v2"
)

type Store struct {
	client    *sdk.BitwardenClient
	projectID string
}

func New(cfg Config) (*Store, error) {
	return &Store{}, nil
}

func (*Store) Get(ctx context.Context, key domain.Key) (domain.Secret, error) {
	return domain.Secret{}, nil
}

func (*Store) Put(ctx context.Context, key domain.Key, value []byte) (domain.Version, error) {
	return domain.Version{}, nil
}

func (*Store) List(ctx context.Context, ns domain.Namespace) ([]domain.SecretMeta, error) {
	return []domain.SecretMeta{}, nil
}

func (*Store) Delete(ctx context.Context, key domain.Key) error {
	return nil
}

var _ ports.SecretStore = (*Store)(nil)
