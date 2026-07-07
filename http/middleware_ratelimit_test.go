package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/vertikon/sdk-hulk.vertikon.com.br/security"
)

func fireRequest(e *echo.Echo, key string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if key != "" {
		req.Header.Set("X-Test-Key", key)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func newLimitedEcho(rps float64, burst int) *echo.Echo {
	e := echo.New()
	mw := NewRateLimitMiddleware(RateLimitConfig{
		RPS:   rps,
		Burst: burst,
		KeyFunc: func(c echo.Context) string {
			return c.Request().Header.Get("X-Test-Key")
		},
	})
	e.GET("/", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"ok": "true"})
	}, mw)
	return e
}

func TestRateLimitExceeded(t *testing.T) {
	e := newLimitedEcho(1, 2) // burst de 2

	if rec := fireRequest(e, "a"); rec.Code != http.StatusOK {
		t.Fatalf("req 1: status = %d", rec.Code)
	}
	if rec := fireRequest(e, "a"); rec.Code != http.StatusOK {
		t.Fatalf("req 2: status = %d", rec.Code)
	}
	rec := fireRequest(e, "a")
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("req 3: status = %d, esperado 429", rec.Code)
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Error("Retry-After ausente no 429")
	}
	if rec.Header().Get("X-RateLimit-Limit") != "2" {
		t.Errorf("X-RateLimit-Limit = %q", rec.Header().Get("X-RateLimit-Limit"))
	}
}

func TestRateLimitKeysAreIndependent(t *testing.T) {
	e := newLimitedEcho(1, 1)

	if rec := fireRequest(e, "a"); rec.Code != http.StatusOK {
		t.Fatalf("chave a: status = %d", rec.Code)
	}
	if rec := fireRequest(e, "a"); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("chave a (2ª): status = %d, esperado 429", rec.Code)
	}
	// Outra chave não é afetada
	if rec := fireRequest(e, "b"); rec.Code != http.StatusOK {
		t.Fatalf("chave b: status = %d, esperado 200", rec.Code)
	}
}

func TestDefaultRateLimitKeyPrefersService(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if key := defaultRateLimitKey(c); key[:3] != "ip:" {
		t.Errorf("sem identidade: chave = %q, esperado prefixo ip:", key)
	}
	c.Set(security.UserContextKey, security.User{ID: "u1"})
	if key := defaultRateLimitKey(c); key != "user:u1" {
		t.Errorf("com usuário: chave = %q", key)
	}
	c.Set(security.ServiceContextKey, security.Service{ID: "k1"})
	if key := defaultRateLimitKey(c); key != "svc:k1" {
		t.Errorf("com serviço: chave = %q (serviço deve ter precedência)", key)
	}
}
