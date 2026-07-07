package vault

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/hashicorp/vault/api"
	"github.com/vertikon/sdk-hulk.vertikon.com.br/secrets"
)

type Provider struct {
	client *api.Client
}

// Verify compliance
var _ secrets.Store = (*Provider)(nil)

func New() (*Provider, error) {
	config := api.DefaultConfig()

	// 1. Address
	if v := os.Getenv("HULK_VAULT_ADDR"); v != "" {
		config.Address = v
	}

	client, err := api.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create vault client: %w", err)
	}

	// 2. Authentication
	// Priority 1: HULK_VAULT_TOKEN
	if v := os.Getenv("HULK_VAULT_TOKEN"); v != "" {
		client.SetToken(v)
	} else if v := os.Getenv("VAULT_TOKEN"); v != "" {
		client.SetToken(v)
	} else {
		// Priority 2: AppRole (Future)
		// For now, fail if no token
		return nil, fmt.Errorf("vault token not found (HULK_VAULT_TOKEN or VAULT_TOKEN)")
	}

	return &Provider{client: client}, nil
}

func (p *Provider) Get(ctx context.Context, key string) (string, error) {
	data, err := p.readData(ctx, key)
	if err != nil {
		return "", err
	}

	// 1. If key "value" exists, return it
	if v, ok := data["value"]; ok {
		return fmt.Sprint(v), nil
	}

	// 2. Fallback: return JSON representation of the whole secret
	b, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (p *Provider) GetJSON(ctx context.Context, key string) ([]byte, error) {
	data, err := p.readData(ctx, key)
	if err != nil {
		return nil, err
	}
	return json.Marshal(data)
}

// readData handles the logical read and basic KVv2 unpacking if needed
func (p *Provider) readData(ctx context.Context, key string) (map[string]interface{}, error) {
	// Logical.ReadWithContext? available in newer api?
	// api.Client usually has Logical().Read()
	// Using context if possible?
	// client.SetContext(ctx)? No.
	// client.Logical() usually doesn't take context directly in older versions, but check.
	// Recent versions use Request object.
	// For simplicity, just use Read (blocking).

	secret, err := p.client.Logical().Read(key)
	if err != nil {
		return nil, fmt.Errorf("vault read failed: %w", err)
	}
	if secret == nil {
		return nil, secrets.ErrSecretNotFound
	}

	// Handle KV v2 "data" wrapper
	data := secret.Data
	if v2Data, ok := data["data"]; ok {
		if mapData, ok := v2Data.(map[string]interface{}); ok {
			data = mapData
		}
	}

	return data, nil
}
