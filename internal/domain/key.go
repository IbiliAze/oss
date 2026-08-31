// Package domain holds the core types of vaultlet. It has no dependencies
// beyond the standard library and performs no I/O.
package domain

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// Key grammar:
//
//	key       = namespace "/" name
//	namespace = segment *( "/" segment )
//	segment   = 1*63( lowercase | digit | "-" )
//	name      = 1*128( alnum | "_" | "-" | "." )
//
// Example: payments/prod/DB_URL

const (
	maxNamespaceDepth = 8
	maxSegmentLen     = 63
	maxNameLen        = 128
)

var (
	segmentRe = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)
	nameRe    = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*$`)
)

var (
	ErrInvalidKey       = errors.New("invalid key")
	ErrInvalidNamespace = errors.New("invalid namespace")
)

// Namespace is a slash-separated hierarchy such as "payments/prod".
type Namespace struct {
	segments []string
}

// Key identifies a single secret: a Namespace plus a leaf name.
type Key struct {
	ns   Namespace
	name string
}

// ParseNamespace validates and constructs a Namespace from its string form.
func ParseNamespace(s string) (Namespace, error) {
	s = strings.Trim(s, "/")
	if s == "" {
		return Namespace{}, fmt.Errorf("%w: empty", ErrInvalidNamespace)
	}
	segs := strings.Split(s, "/")
	if len(segs) > maxNamespaceDepth {
		return Namespace{}, fmt.Errorf("%w: depth %d exceeds %d", ErrInvalidNamespace, len(segs), maxNamespaceDepth)
	}
	for _, seg := range segs {
		if len(seg) > maxSegmentLen || !segmentRe.MatchString(seg) {
			return Namespace{}, fmt.Errorf("%w: bad segment %q", ErrInvalidNamespace, seg)
		}
	}
	return Namespace{segments: segs}, nil
}

// MustNamespace is for tests and static configuration only.
func MustNamespace(s string) Namespace {
	ns, err := ParseNamespace(s)
	if err != nil {
		panic(err)
	}
	return ns
}

// ParseKey validates and constructs a Key from "ns/.../name".
func ParseKey(s string) (Key, error) {
	s = strings.Trim(s, "/")
	i := strings.LastIndex(s, "/")
	if i <= 0 || i == len(s)-1 {
		return Key{}, fmt.Errorf("%w: %q must be namespace/name", ErrInvalidKey, s)
	}
	ns, err := ParseNamespace(s[:i])
	if err != nil {
		return Key{}, fmt.Errorf("%w: %v", ErrInvalidKey, err)
	}
	name := s[i+1:]
	if len(name) > maxNameLen || !nameRe.MatchString(name) {
		return Key{}, fmt.Errorf("%w: bad name %q", ErrInvalidKey, name)
	}
	return Key{ns: ns, name: name}, nil
}

// MustKey is for tests and static configuration only.
func MustKey(s string) Key {
	k, err := ParseKey(s)
	if err != nil {
		panic(err)
	}
	return k
}

// NewKey builds a Key from an already-valid Namespace and a name.
func NewKey(ns Namespace, name string) (Key, error) {
	return ParseKey(ns.String() + "/" + name)
}

// --- Key methods -------------------------------------------------------

func (k Key) IsZero() bool { return k.name == "" }

// Namespace returns the key's enclosing namespace.
func (k Key) Namespace() Namespace { return k.ns }

// Name returns the leaf name, without the namespace.
func (k Key) Name() string { return k.name }

// String renders the canonical "namespace/name" form. It is the wire form
// adapters should use when a backend stores secrets under a flat name.
func (k Key) String() string {
	if k.IsZero() {
		return ""
	}
	return k.ns.String() + "/" + k.name
}

// --- Namespace methods -------------------------------------------------------

func (n Namespace) String() string { return strings.Join(n.segments, "/") }
func (n Namespace) IsZero() bool   { return len(n.segments) == 0 }
func (n Namespace) Depth() int     { return len(n.segments) }
func (n Namespace) Segments() []string {
	out := make([]string, len(n.segments))
	copy(out, n.segments)
	return out
}

// Parent returns the enclosing namespace, or the zero Namespace at the root.
func (n Namespace) Parent() Namespace {
	if len(n.segments) <= 1 {
		return Namespace{}
	}
	return Namespace{segments: n.segments[:len(n.segments)-1]}
}

// Contains reports whether other is n itself or nested beneath it.
// payments.Contains(payments/prod) == true.
func (n Namespace) Contains(other Namespace) bool {
	if len(other.segments) < len(n.segments) {
		return false
	}
	for i, seg := range n.segments {
		if other.segments[i] != seg {
			return false
		}
	}
	return true
}
