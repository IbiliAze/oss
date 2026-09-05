package app

import (
	"context"
	"errors"
	"testing"

	"github.com/IbiliAze/vaultlet/internal/domain"
	"github.com/IbiliAze/vaultlet/internal/ports"
)

// fakeStore is an in-memory ports.SecretStore. It records calls so a test
// can assert the service never touched it on a denied request.
type fakeStore struct {
	secrets  map[string]domain.Secret // keyed by key.String()
	err      error                    // if set, every method returns it
	getCalls int
	putCalls int
}

func (f *fakeStore) Get(_ context.Context, key domain.Key) (domain.Secret, error) {
	f.getCalls++
	if f.err != nil {
		return domain.Secret{}, f.err
	}
	s, ok := f.secrets[key.String()]
	if !ok {
		return domain.Secret{}, ports.ErrNotFound
	}
	return s, nil
}

func (f *fakeStore) Put(_ context.Context, key domain.Key, value []byte) (domain.SecretMeta, error) {
	f.putCalls++
	if f.err != nil {
		return domain.SecretMeta{}, f.err
	}
	return domain.SecretMeta{Key: key}, nil
}
func (f *fakeStore) List(context.Context, domain.Namespace) ([]domain.SecretMeta, error) {
	return nil, nil
}
func (f *fakeStore) Delete(context.Context, domain.Key) error { return nil }

func TestServicePut(t *testing.T) {
	key := domain.MustKey("payments/prod/DB_URL")

	policy, err := NewPolicy([]RuleSpec{
		{Principal: "alice", Namespace: "payments", Actions: []string{"put"}},
	})
	if err != nil {
		t.Fatal(err)
	}

	value := []byte("secret")

	tests := []struct {
		name       string
		ctx        context.Context
		storeErr   error
		wantErr    error
		wantCalled bool
	}{
		{"no principal", context.Background(), nil, ErrPermissionDenied, false},
		{"wrong principal", WithPrincipal(context.Background(), "bob"), nil, ErrPermissionDenied, false},
		{"allowed", WithPrincipal(context.Background(), "alice"), nil, nil, true},
		{"store error passes through", WithPrincipal(context.Background(), "alice"), ports.ErrNotFound, ports.ErrNotFound, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := &fakeStore{
				secrets: map[string]domain.Secret{},
				err:     tc.storeErr,
			}
			svc := NewService(store, policy)

			got, err := svc.Put(tc.ctx, key, value)

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if called := store.putCalls > 0; called != tc.wantCalled {
				t.Errorf("store called = %v, want %v", called, tc.wantCalled)
			}
			if tc.wantErr == nil && got.Key.String() != "payments/prod/DB_URL" {
				t.Errorf("value = %q, want the key", got.Key.String())
			}
		})
	}
}

func TestServiceGet(t *testing.T) {
	key := domain.MustKey("payments/prod/DB_URL")
	secret, err := domain.NewSecret(domain.SecretMeta{
		Key:     key,
		Version: mustVersion(t, "v1"),
	}, []byte("postgres://..."))
	if err != nil {
		t.Fatal(err)
	}

	policy, err := NewPolicy([]RuleSpec{
		{Principal: "alice", Namespace: "payments", Actions: []string{"get"}},
	})
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		ctx        context.Context
		storeErr   error
		wantErr    error
		wantCalled bool
	}{
		{"no principal", context.Background(), nil, ErrPermissionDenied, false},
		{"wrong principal", WithPrincipal(context.Background(), "bob"), nil, ErrPermissionDenied, false},
		{"allowed", WithPrincipal(context.Background(), "alice"), nil, nil, true},
		{"secret is read-only", WithPrincipal(context.Background(), "alice"), ports.ErrReadOnly, ports.ErrReadOnly, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := &fakeStore{
				secrets: map[string]domain.Secret{key.String(): secret},
				err:     tc.storeErr,
			}
			svc := NewService(store, policy)

			got, err := svc.Get(tc.ctx, key)

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if called := store.getCalls > 0; called != tc.wantCalled {
				t.Errorf("store called = %v, want %v", called, tc.wantCalled)
			}
			if tc.wantErr == nil && string(got.Value()) != "postgres://..." {
				t.Errorf("value = %q, want the stored secret", got.Value())
			}
		})
	}
}

func mustVersion(t *testing.T, id string) domain.Version {
	t.Helper()
	v, err := domain.NewVersion(id)
	if err != nil {
		t.Fatal(err)
	}
	return v
}
