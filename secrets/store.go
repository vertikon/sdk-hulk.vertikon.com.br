package secrets

import (
	"context"
	"errors"
)

var (
	// ErrSecretNotFound indicates that the requested secret was not found in the store
	ErrSecretNotFound = errors.New("secret not found")
)

// Store defines the interface for interacting with a secret management system
// (e.g. HashiCorp Vault, AWS Secrets Manager, or local file for dev)
type Store interface {
	// Get retrieves a single secret value as a string.
	// Common for simple keys like "DB_PASSWORD".
	Get(ctx context.Context, key string) (string, error)

	// GetJSON retrieves a secret value and attempts to return the raw bytes.
	// Useful for complex secrets stored as JSON blobs.
	GetJSON(ctx context.Context, key string) ([]byte, error)
}
