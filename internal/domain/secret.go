package domain

import (
	"errors"
	"fmt"
	"time"
)

var (
	ErrEmptyValue     = errors.New("secret value must not be empty")
	ErrInvalidVersion = errors.New("invalid version")
)

// Version is an opaque, backend-specific identifier for one revision of a
// secret. Two Versions are equal iff they refer to the same revision; no
// ordering is implied — use SecretMeta.CreatedAt for that.
type Version struct {
	id string
}

func NewVersion(id string) (Version, error) {
	if id == "" {
		return Version{}, ErrInvalidVersion
	}
	return Version{id: id}, nil
}

func (v Version) String() string { return v.id }
func (v Version) IsZero() bool   { return v.id == "" }

func (v Version) MarshalText() ([]byte, error) { return []byte(v.id), nil }
func (v *Version) UnmarshalText(b []byte) error {
	parsed, err := NewVersion(string(b))
	if err != nil {
		return err
	}
	*v = parsed
	return nil
}

// SecretMeta describes a secret without exposing its value.
// Safe to log and to return from List.
type SecretMeta struct {
	Key       Key
	Version   Version
	CreatedAt time.Time
}

// Secret is a single revision of a secret, including its value.
type Secret struct {
	meta  SecretMeta
	value []byte
}

func NewSecret(meta SecretMeta, value []byte) (Secret, error) {
	if meta.Key.IsZero() {
		return Secret{}, ErrInvalidKey
	}
	if meta.Version.IsZero() {
		return Secret{}, ErrInvalidVersion
	}
	if len(value) == 0 {
		return Secret{}, ErrEmptyValue
	}
	// Defensive copy: callers (and adapters) must not be able to mutate
	// the value after construction.
	v := make([]byte, len(value))
	copy(v, value)
	return Secret{meta: meta, value: v}, nil
}

func (s Secret) Meta() SecretMeta     { return s.meta }
func (s Secret) Key() Key             { return s.meta.Key }
func (s Secret) Version() Version     { return s.meta.Version }
func (s Secret) CreatedAt() time.Time { return s.meta.CreatedAt }

// Value returns a copy of the secret's bytes.
func (s Secret) Value() []byte {
	v := make([]byte, len(s.value))
	copy(v, s.value)
	return v
}

// String redacts the value so a Secret can never be logged by accident.
func (s Secret) String() string {
	return fmt.Sprintf("Secret{%s@%s}", s.meta.Key, s.meta.Version)
}

// GoString covers %#v as well.
func (s Secret) GoString() string { return s.String() }
