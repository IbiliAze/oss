package bitwarden

import (
	"context"
	"fmt"
	"time"

	"github.com/IbiliAze/vaultlet/internal/domain"
	"github.com/IbiliAze/vaultlet/internal/ports"

	"github.com/bitwarden/sdk-go/v2"
)

type Store struct {
	client    sdk.BitwardenClientInterface
	projectID string
	orgID     string
}

func New(cfg Config) (*Store, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	client, err := sdk.NewBitwardenClient(optional(cfg.APIURL), optional(cfg.IdentityURL))
	if err != nil {
		return nil, fmt.Errorf("bitwarden: init client: %w", err)
	}

	if err := client.AccessTokenLogin(cfg.AccessToken, nil); err != nil {
		client.Close()
		return nil, fmt.Errorf("bitwarden: access token login: %w", err)
	}

	return &Store{client: client, projectID: cfg.ProjectID, orgID: cfg.OrgID}, nil
}

// Close releases the FFI handle held by the underlying Rust client.
func (s *Store) Close() { s.client.Close() }

func (s *Store) Get(ctx context.Context, key domain.Key) (domain.Secret, error) {
	// The SDK calls into the Rust core over cgo and blocks; it takes no
	// context, so the best we can do is refuse work that is already cancelled.
	if err := ctx.Err(); err != nil {
		return domain.Secret{}, err
	}

	id, err := s.resolveID(key)
	if err != nil {
		return domain.Secret{}, err
	}

	res, err := s.client.Secrets().Get(id)
	if err != nil {
		return domain.Secret{}, fmt.Errorf("bitwarden: get secret %s: %w", key, err)
	}

	// Bitwarden has no version identifier of its own; RevisionDate is the only
	// value that changes on every write, so it stands in as the opaque version.
	version, err := domain.NewVersion(res.RevisionDate.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return domain.Secret{}, fmt.Errorf("bitwarden: secret %s has no revision date: %w", key, err)
	}

	return domain.NewSecret(domain.SecretMeta{
		Key:       key,
		Version:   version,
		CreatedAt: res.CreationDate.UTC(),
	}, []byte(res.Value))
}

// resolveID maps a domain key onto the Bitwarden secret UUID. The SDK only
// addresses secrets by UUID, so every lookup by name costs a List of the whole
// organization first.
func (s *Store) resolveID(key domain.Key) (string, error) {
	res, err := s.client.Secrets().List(s.orgID)
	if err != nil {
		return "", fmt.Errorf("bitwarden: list secrets: %w", err)
	}

	name := key.String()
	for _, ident := range res.Data {
		if ident.Key == name && s.inProject(ident.ProjectIDS) {
			return ident.ID, nil
		}
	}
	return "", fmt.Errorf("bitwarden: %s: %w", key, ports.ErrNotFound)
}

// inProject reports whether a secret belongs to the project this Store is
// scoped to. An unset projectID means the whole organization is in scope.
func (s *Store) inProject(projectIDs []string) bool {
	if s.projectID == "" {
		return true
	}
	for _, id := range projectIDs {
		if id == s.projectID {
			return true
		}
	}
	return false
}

func (s *Store) Put(ctx context.Context, key domain.Key, value []byte) (domain.Version, error) {
	defer s.Close()
	res, err := s.client.Secrets().Update("", "", string(value))
	if err != nil {
		return domain.Version{}, fmt.Errorf("bitwarden: update secret error: %w", err)
	}

	return domain.Version{}, nil
}

func (s *Store) List(ctx context.Context, ns domain.Namespace) ([]domain.SecretMeta, error) {
	defer s.Close()
	res, err := s.client.Secrets().List(s.orgID)
	if err != nil {
		return nil, fmt.Errorf("bitwarden: list secrets error: %w", err)
	}

	return []domain.SecretMeta{}, nil
}

func (s *Store) Delete(ctx context.Context, key domain.Key) error {
	defer s.Close()
	res, err := s.client.Secrets().Delete([]string{})
	if err != nil {
		return fmt.Errorf("bitwarden: delete secrets error: %w", err)
	}

	return nil
}

func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

var _ ports.SecretStore = (*Store)(nil)
