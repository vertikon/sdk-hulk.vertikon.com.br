// Package neovault lê os segredos do SEU sistema no NeoVault (mcp-secrets,
// https://api.neovault.com.br/secrets) — a regra do ecossistema: todo segredo
// vive cifrado no NeoVault, nunca em .env solto. Token read-only escopado no
// sistema (SECRETS_TOKEN); o mesmo contrato do `vault-fetch`.
package neovault

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/vertikon/sdk-hulk.vertikon.com.br/secrets"
)

// DefaultBaseURL é a borda de produção do mcp-secrets.
const DefaultBaseURL = "https://api.neovault.com.br/secrets"

// Provider lê os segredos de um sistema (ex.: radar.vertikon.com.br).
type Provider struct {
	BaseURL string
	Token   string
	System  string
	HTTP    *http.Client
}

var _ secrets.Store = (*Provider)(nil)

// New cria o provider. baseURL vazio usa DefaultBaseURL.
func New(baseURL, token, system string) *Provider {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return &Provider{BaseURL: strings.TrimRight(baseURL, "/"), Token: token, System: system,
		HTTP: &http.Client{Timeout: 15 * time.Second}}
}

// FromEnv: SECRETS_TOKEN (obrigatório), SECRETS_API_URL (opcional) e o sistema.
// Sem token devolve nil — o chamador decide se roda sem cofre (dev).
func FromEnv(system string) *Provider {
	tok := os.Getenv("SECRETS_TOKEN")
	if tok == "" {
		return nil
	}
	return New(os.Getenv("SECRETS_API_URL"), tok, system)
}

func (p *Provider) get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.BaseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+p.Token)
	resp, err := p.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("neovault: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return secrets.ErrSecretNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("neovault: GET %s -> %d", path, resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// Get devolve o valor do segredo `name` do sistema.
func (p *Provider) Get(ctx context.Context, name string) (string, error) {
	var s struct {
		Value string `json:"value"`
	}
	err := p.get(ctx, "/v1/tenants/"+url.PathEscape(p.System)+"/secrets/"+url.PathEscape(name), &s)
	return s.Value, err
}

// GetJSON devolve o valor bruto (segredos JSON).
func (p *Provider) GetJSON(ctx context.Context, name string) ([]byte, error) {
	v, err := p.Get(ctx, name)
	return []byte(v), err
}

// List devolve os nomes dos segredos do sistema (sem valores).
func (p *Provider) List(ctx context.Context) ([]string, error) {
	var out struct {
		Names []string `json:"names"`
	}
	err := p.get(ctx, "/v1/tenants/"+url.PathEscape(p.System)+"/secrets", &out)
	return out.Names, err
}

// LoadEnv lê TODOS os segredos do sistema (blocos KEY=VALUE, como o
// `provision` e o `vault-fetch` gravam) e os aplica ao ambiente do processo
// SEM sobrescrever o que já está definido (env explícito do operador vence).
// Devolve só os NOMES das chaves aplicadas — nunca valores.
func (p *Provider) LoadEnv(ctx context.Context) ([]string, error) {
	names, err := p.List(ctx)
	if err != nil {
		return nil, err
	}
	var aplicadas []string
	for _, n := range names {
		v, err := p.Get(ctx, n)
		if err != nil {
			return aplicadas, fmt.Errorf("neovault: segredo %s: %w", n, err)
		}
		for k, val := range ParseEnv(v) {
			if _, existe := os.LookupEnv(k); existe {
				continue
			}
			if err := os.Setenv(k, val); err != nil {
				return aplicadas, err
			}
			aplicadas = append(aplicadas, k)
		}
	}
	return aplicadas, nil
}

// ParseEnv lê um bloco no formato .env (KEY=VALUE por linha; # comenta; aspas
// simples/duplas externas removidas). Linha sem '=' é ignorada.
func ParseEnv(s string) map[string]string {
	out := map[string]string{}
	sc := bufio.NewScanner(strings.NewReader(s))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		k, v, ok := strings.Cut(line, "=")
		k = strings.TrimSpace(k)
		if !ok || k == "" {
			continue
		}
		v = strings.TrimSpace(v)
		if len(v) >= 2 && (v[0] == '"' && v[len(v)-1] == '"' || v[0] == '\'' && v[len(v)-1] == '\'') {
			v = v[1 : len(v)-1]
		}
		out[k] = v
	}
	return out
}
