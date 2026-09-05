package app

import (
	"context"
	"testing"
)

func TestPrincipalRoundTrip(t *testing.T) {
	ctx := WithPrincipal(context.Background(), "alice")

	name, ok := PrincipalFromContext(ctx)
	if !ok {
		t.Fatal("PrincipalFromContext returned ok=false for a context with a principal")
	}
	if name != "alice" {
		t.Errorf("got principal %q, want %q", name, "alice")
	}
}

func TestPrincipalMissing(t *testing.T) {
	name, ok := PrincipalFromContext(context.Background())
	if ok {
		t.Errorf("got ok=true with principal %q, want ok=false for a bare context", name)
	}
}
