package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/vertikon/sdk-hulk.vertikon.com.br/security"
)

func runGuard(t *testing.T, setUser *security.User, header string) (status int, called bool, injected string) {
	t.Helper()
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	if header != "" {
		req.Header.Set("X-Tenant-ID", header)
	}
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	if setUser != nil {
		c.Set(security.UserContextKey, *setUser)
	}
	next := func(c echo.Context) error {
		called = true
		injected = c.Request().Header.Get("X-Tenant-ID")
		return c.NoContent(http.StatusOK)
	}
	_ = NewTenantGuardMiddleware()(next)(c)
	return rec.Code, called, injected
}

func TestTenantGuard_NoUser_Passes(t *testing.T) {
	_, called, _ := runGuard(t, nil, "")
	if !called {
		t.Fatal("sem usuário no contexto, deveria seguir (serviço/anon)")
	}
}

func TestTenantGuard_NilTenant_Passes(t *testing.T) {
	_, called, hdr := runGuard(t, &security.User{ID: "u1"}, "qualquer-coisa")
	if !called {
		t.Fatal("usuário sem tenant (global) deveria passar")
	}
	if hdr != "qualquer-coisa" {
		t.Fatalf("header não deveria ser alterado p/ usuário global, got %q", hdr)
	}
}

func TestTenantGuard_TenantInjected(t *testing.T) {
	tid := uuid.New()
	_, called, hdr := runGuard(t, &security.User{ID: "u1", TenantID: tid}, "")
	if !called {
		t.Fatal("deveria seguir injetando o tenant")
	}
	if hdr != tid.String() {
		t.Fatalf("tenant não injetado: got %q want %q", hdr, tid.String())
	}
}

func TestTenantGuard_TenantMatch_Passes(t *testing.T) {
	tid := uuid.New()
	code, called, _ := runGuard(t, &security.User{ID: "u1", TenantID: tid}, tid.String())
	if !called || code == http.StatusForbidden {
		t.Fatalf("header conferindo deveria passar (code=%d, called=%v)", code, called)
	}
}

func TestTenantGuard_TenantMismatch_403(t *testing.T) {
	tid := uuid.New()
	code, called, _ := runGuard(t, &security.User{ID: "u1", TenantID: tid}, uuid.New().String())
	if called {
		t.Fatal("header divergente NÃO deveria chamar o próximo")
	}
	if code != http.StatusForbidden {
		t.Fatalf("esperava 403, got %d", code)
	}
}
