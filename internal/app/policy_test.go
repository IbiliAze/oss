package app

import (
	"testing"

	"github.com/IbiliAze/vaultlet/internal/domain"
)

func TestPolicyAllows(t *testing.T) {
	policy, err := NewPolicy([]RuleSpec{
		{Principal: "alice", Namespace: "payments", Actions: []string{"get", "list"}},
		{Principal: "alice", Namespace: "infra/prod", Actions: []string{"put"}},
		{Principal: "root", Namespace: "*", Actions: []string{"get", "put", "delete"}},
	})
	if err != nil {
		t.Fatalf("NewPolicy: %v", err)
	}

	tests := []struct {
		name      string
		principal string
		action    Action
		ns        string
		want      bool
	}{
		{"exact namespace", "alice", ActionGet, "payments", true},
		{"child namespace inherits", "alice", ActionGet, "payments/prod", true},
		{"parent namespace is not granted", "alice", ActionPut, "infra", false},
		{"action not in rule", "alice", ActionPut, "payments", false},
		{"second rule for same principal", "alice", ActionPut, "infra/prod", true},
		{"sibling namespace", "alice", ActionGet, "billing", false},
		{"wildcard grants everything", "root", ActionDelete, "anything/at/all", true},
		{"unknown principal", "mallory", ActionGet, "payments", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := policy.allows(tc.principal, tc.action, domain.MustNamespace(tc.ns))
			if got != tc.want {
				t.Errorf("allows(%q, %q, %q) = %v, want %v",
					tc.principal, tc.action, tc.ns, got, tc.want)
			}
		})
	}
}

func TestNewPolicyRejectsBadSpecs(t *testing.T) {
	tests := []struct {
		name string
		spec RuleSpec
	}{
		{"unknown action", RuleSpec{Principal: "a", Namespace: "x", Actions: []string{"fly"}}},
		{"bad namespace", RuleSpec{Principal: "a", Namespace: "Not/Valid", Actions: []string{"get"}}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := NewPolicy([]RuleSpec{tc.spec}); err == nil {
				t.Errorf("NewPolicy(%+v) returned nil error, want an error", tc.spec)
			}
		})
	}
}
