package neovault

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"testing"

	"github.com/vertikon/sdk-hulk.vertikon.com.br/secrets"
)

func cofreFalso(t *testing.T, dados map[string]string) *Provider {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/v1/tenants/radar.vertikon.com.br/secrets":
			var names []string
			for n := range dados {
				names = append(names, n)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"names": names})
		default:
			n := r.URL.Path[len("/v1/tenants/radar.vertikon.com.br/secrets/"):]
			v, ok := dados[n]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"value": v})
		}
	}))
	t.Cleanup(srv.Close)
	return New(srv.URL, "tok", "radar.vertikon.com.br")
}

func TestLoadEnv(t *testing.T) {
	p := cofreFalso(t, map[string]string{
		".env-id-radar":     "VERTIKON_OIDC_CLIENT_ID=cl-radar\nVERTIKON_OIDC_CLIENT_SECRET=\"s3cr3t\"\n# comentário\n",
		".env-ragaas-radar": "RAGAAS_API_KEY=rag_sk_x\nRADAR_JA_DEFINIDA=do-cofre\n",
	})
	t.Setenv("RADAR_JA_DEFINIDA", "do-operador")
	for _, k := range []string{"VERTIKON_OIDC_CLIENT_ID", "VERTIKON_OIDC_CLIENT_SECRET", "RAGAAS_API_KEY"} {
		t.Setenv(k, "") // registra para limpeza ao fim do teste
		_ = os.Unsetenv(k)
	}
	aplicadas, err := p.LoadEnv(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if os.Getenv("VERTIKON_OIDC_CLIENT_SECRET") != "s3cr3t" || os.Getenv("RAGAAS_API_KEY") != "rag_sk_x" {
		t.Fatal("segredos do cofre não chegaram ao ambiente")
	}
	if os.Getenv("RADAR_JA_DEFINIDA") != "do-operador" || slices.Contains(aplicadas, "RADAR_JA_DEFINIDA") {
		t.Fatal("env explícito do operador não pode ser sobrescrito pelo cofre")
	}
	if len(aplicadas) != 3 {
		t.Fatalf("aplicadas = %v", aplicadas)
	}
}

func TestGet(t *testing.T) {
	p := cofreFalso(t, map[string]string{"x": "1"})
	if _, err := p.Get(context.Background(), "nao-existe"); !errors.Is(err, secrets.ErrSecretNotFound) {
		t.Fatalf("404 deveria ser ErrSecretNotFound: %v", err)
	}
	p.Token = "errado"
	if _, err := p.Get(context.Background(), "x"); err == nil {
		t.Fatal("token errado deveria falhar")
	}
}
