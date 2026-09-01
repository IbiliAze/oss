// Package app will hold vaultlet's application services — the policy and
// audit layer between the gRPC handlers and ports.SecretStore. For now it
// defines only the principal: the authenticated identity that layer will
// make decisions about.
package app

import "context"

// principalKey is unexported so nothing outside this package can forge a
// principal into a context; WithPrincipal is the only door in.
type principalKey struct{}

// WithPrincipal returns a child context carrying the authenticated principal.
// It is called exactly once per RPC, by the server's auth interceptor.
func WithPrincipal(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, principalKey{}, name)
}

// PrincipalFromContext returns the authenticated principal, or false if the
// context never passed authentication.
func PrincipalFromContext(ctx context.Context) (string, bool) {
	name, ok := ctx.Value(principalKey{}).(string)
	return name, ok
}
