package file

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync" // Although read-only usually, maybe reloadable later?

	"github.com/vertikon/sdk-hulk.vertikon.com.br/secrets"
)

// Provider implements secrets.Store using a local JSON file.
// Intended for Development use only.
type Provider struct {
	path string
	data map[string]interface{}
	mu   sync.RWMutex
}

// Verify interface compliance
var _ secrets.Store = (*Provider)(nil)

// New creates a new File Provider loading secrets from the given path.
func New(path string) (*Provider, error) {
	p := &Provider{
		path: path,
		data: make(map[string]interface{}),
	}

	if err := p.load(); err != nil {
		return nil, fmt.Errorf("failed to load secrets file: %w", err)
	}

	return p, nil
}

func (p *Provider) load() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// If file doesn't exist, we might treat as empty or error?
	// For dev safety, warn if missing but maybe allow empty?
	// Blueprint says "dev.secrets.json".
	content, err := os.ReadFile(p.path)
	if err != nil {
		if os.IsNotExist(err) {
			// Initialize empty
			return nil
		}
		return err
	}

	if len(content) == 0 {
		return nil
	}

	return json.Unmarshal(content, &p.data)
}

func (p *Provider) Get(ctx context.Context, key string) (string, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	val, ok := p.data[key]
	if !ok {
		return "", secrets.ErrSecretNotFound
	}

	// Handle types
	switch v := val.(type) {
	case string:
		return v, nil
	case float64:
		return fmt.Sprintf("%g", v), nil // JSON numbers
	case bool:
		return fmt.Sprintf("%t", v), nil
	default:
		// Complex type? Return serialized or error?
		// Get() expects string.
		// Try to marshal it back to string if it's an object/array?
		// No, usually Get() is for leaf values.
		b, err := json.Marshal(v)
		if err != nil {
			return "", fmt.Errorf("failed to convert value to string: %w", err)
		}
		return string(b), nil
	}
}

func (p *Provider) GetJSON(ctx context.Context, key string) ([]byte, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	val, ok := p.data[key]
	if !ok {
		return nil, secrets.ErrSecretNotFound
	}

	// If it's already a JSON string, do we return raw bytes of that string?
	// Or do we marshal the value (e.g. map) to JSON?
	// Usually GetJSON implies retrieving a structured secret.
	return json.Marshal(val)
}
