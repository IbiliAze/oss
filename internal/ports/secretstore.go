package secretstore

import (
	"context"

	"github.com/IbiliAze/vaultlet/internal/domain"
)

type SecretStore interface {
	Get(ctx context.Context, key domain.Key) (domain.Secret, error)
	Put(ctx context.Context, key domain.Key, value []byte) (domain.Version, error)
	List(ctx context.Context, ns domain.Namespace) ([]domain.SecretMeta, error)
	Delete(ctx context.Context, key domain.Key) error
}
