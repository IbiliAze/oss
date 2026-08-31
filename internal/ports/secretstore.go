package ports

import (
	"context"
	"errors"

	"github.com/IbiliAze/vaultlet/internal/domain"
)

// ErrNotFound is returned by Get and Delete when no secret exists at the key.
// Every backend must wrap this so callers can test with errors.Is.
var ErrNotFound = errors.New("secret not found")

type SecretStore interface {
	Get(ctx context.Context, key domain.Key) (domain.Secret, error)
	Put(ctx context.Context, key domain.Key, value []byte) (domain.SecretMeta, error)
	List(ctx context.Context, ns domain.Namespace) ([]domain.SecretMeta, error)
	Delete(ctx context.Context, key domain.Key) error
}
