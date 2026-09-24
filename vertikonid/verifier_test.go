package vertikonid

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type fixture struct {
	priv *rsa.PrivateKey
	v    *Verifier
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	jwks := map[string]any{"keys": []map[string]string{{
		"kty": "RSA", "kid": "k1", "alg": "RS256", "use": "sig",
		"n": base64.RawURLEncoding.EncodeToString(priv.N.Bytes()),
		"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(priv.E)).Bytes()),
	}}}
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/jwks.json", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(jwks)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return &fixture{priv: priv, v: New(srv.URL)}
}

func (f *fixture) sign(t *testing.T, kid string, claims map[string]any) string {
	t.Helper()
	head, _ := json.Marshal(map[string]string{"alg": "RS256", "typ": "JWT", "kid": kid})
	pay, _ := json.Marshal(claims)
	signing := base64.RawURLEncoding.EncodeToString(head) + "." + base64.RawURLEncoding.EncodeToString(pay)
	digest := sha256.Sum256([]byte(signing))
	sig, err := rsa.SignPKCS1v15(rand.Reader, f.priv, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	return signing + "." + base64.RawURLEncoding.EncodeToString(sig)
}

func baseClaims(f *fixture) map[string]any {
	return map[string]any{
		"sub": "user-1", "iss": f.v.Issuer, "tenant_id": "escola",
		"vertical": "educacao", "plan": "pro", "scope": "rag:query agent:execute",
		"exp": time.Now().Add(15 * time.Minute).Unix(),
	}
}

func TestVerifyOK(t *testing.T) {
	f := newFixture(t)
	c, err := f.v.Verify(context.Background(), f.sign(t, "k1", baseClaims(f)))
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if c.TenantID != "escola" || c.Vertical != "educacao" || c.Suspended() {
		t.Fatalf("claims inesperadas: %+v", c)
	}
	if !c.HasScope("rag:query") || c.HasScope("admin") {
		t.Fatalf("HasScope errado: %q", c.Scope)
	}
}

func TestRecusa(t *testing.T) {
	casos := []struct {
		nome string
		mexe func(map[string]any)
		kid  string
		quer error
	}{
		{"expirado", func(c map[string]any) { c["exp"] = time.Now().Add(-time.Minute).Unix() }, "k1", ErrTokenExpired},
		{"sem exp (seria eterno)", func(c map[string]any) { delete(c, "exp") }, "k1", ErrTokenExpired},
		{"issuer errado", func(c map[string]any) { c["iss"] = "https://atacante.example" }, "k1", ErrIssuerMismatch},
		{"sem iss", func(c map[string]any) { delete(c, "iss") }, "k1", ErrIssuerMismatch},
		{"kid desconhecido", func(map[string]any) {}, "k9", ErrKeyNotFound},
	}
	for _, tc := range casos {
		t.Run(tc.nome, func(t *testing.T) {
			f := newFixture(t)
			cl := baseClaims(f)
			tc.mexe(cl)
			if _, err := f.v.Verify(context.Background(), f.sign(t, tc.kid, cl)); !errors.Is(err, tc.quer) {
				t.Fatalf("esperava %v, veio %v", tc.quer, err)
			}
		})
	}
}

func TestAssinaturaInvalida(t *testing.T) {
	f := newFixture(t)
	outra, _ := rsa.GenerateKey(rand.Reader, 2048)
	forjado := &fixture{priv: outra, v: f.v}
	if _, err := f.v.Verify(context.Background(), forjado.sign(t, "k1", baseClaims(f))); !errors.Is(err, ErrSignatureInvalid) {
		t.Fatalf("esperava ErrSignatureInvalid, veio %v", err)
	}
}

func TestMalformado(t *testing.T) {
	f := newFixture(t)
	if _, err := f.v.Verify(context.Background(), "não.é.jwt.valido"); !errors.Is(err, ErrTokenMalformed) {
		t.Fatalf("esperava ErrTokenMalformed, veio %v", err)
	}
}

func TestMiddleware(t *testing.T) {
	f := newFixture(t)
	var visto *Claims
	h := Middleware(f.v)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		visto, _ = FromContext(r.Context())
		w.WriteHeader(http.StatusNoContent)
	}))
	suspenso := baseClaims(f)
	suspenso["plan"] = PlanSuspended

	casos := []struct {
		nome, auth string
		quer       int
	}{
		{"sem token", "", http.StatusUnauthorized},
		{"esquema errado", "Basic abc", http.StatusUnauthorized},
		{"token inválido", "Bearer x.y.z", http.StatusUnauthorized},
		{"tenant suspenso", "Bearer " + f.sign(t, "k1", suspenso), http.StatusForbidden},
		{"ok", "Bearer " + f.sign(t, "k1", baseClaims(f)), http.StatusNoContent},
	}
	for _, tc := range casos {
		t.Run(tc.nome, func(t *testing.T) {
			visto = nil
			req := httptest.NewRequest(http.MethodGet, "/api/v1/itens", nil)
			if tc.auth != "" {
				req.Header.Set("Authorization", tc.auth)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != tc.quer {
				t.Fatalf("status %d, esperava %d (%s)", rec.Code, tc.quer, rec.Body.String())
			}
			if tc.quer == http.StatusNoContent && (visto == nil || visto.TenantID != "escola") {
				t.Fatalf("claims não chegaram ao handler: %+v", visto)
			}
			if tc.quer != http.StatusNoContent && visto != nil {
				t.Fatal("handler não pode rodar quando o acesso é negado")
			}
		})
	}
}
