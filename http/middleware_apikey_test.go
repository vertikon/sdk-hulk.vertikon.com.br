package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/vertikon/sdk-hulk.vertikon.com.br/security"
)

type fakeValidator struct {
	service *security.Service
	err     error
	lastKey string
}

func (f *fakeValidator) ValidateAPIKey(_ context.Context, rawKey string) (*security.Service, error) {
	f.lastKey = rawKey
	if f.err != nil {
		return nil, f.err
	}
	return f.service, nil
}

func runRequest(t *testing.T, mw echo.MiddlewareFunc, configure func(*http.Request)) (*httptest.ResponseRecorder, *security.Service) {
	t.Helper()
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	configure(req)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	var captured *security.Service
	handler := mw(func(c echo.Context) error {
		captured = GetServiceFromContext(c)
		return c.JSON(http.StatusOK, map[string]string{"ok": "true"})
	})
	if err := handler(c); err != nil {
		t.Fatalf("handler retornou erro: %v", err)
	}
	return rec, captured
}

func TestAPIKeyMiddlewareXAPIKeyHeader(t *testing.T) {
	validator := &fakeValidator{service: &security.Service{ID: "k1", Name: "erp-acme", Scopes: []string{"oms:read"}}}
	mw := NewAPIKeyMiddleware(validator)

	rec, svc := runRequest(t, mw, func(r *http.Request) {
		r.Header.Set("X-API-Key", "vtk_test_abc")
	})

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, esperado 200; body: %s", rec.Code, rec.Body.String())
	}
	if svc == nil || svc.Name != "erp-acme" {
		t.Fatalf("service não injetado no contexto: %+v", svc)
	}
	if validator.lastKey != "vtk_test_abc" {
		t.Errorf("validator recebeu chave %q", validator.lastKey)
	}
}

func TestAPIKeyMiddlewareBearerAPIKey(t *testing.T) {
	validator := &fakeValidator{service: &security.Service{ID: "k1", Name: "erp-acme"}}
	mw := NewAPIKeyMiddleware(validator)

	rec, svc := runRequest(t, mw, func(r *http.Request) {
		r.Header.Set("Authorization", "Bearer vtk_live_xyz")
	})

	if rec.Code != http.StatusOK || svc == nil {
		t.Fatalf("status = %d, service = %+v", rec.Code, svc)
	}
}

func TestAPIKeyMiddlewareMissingKey(t *testing.T) {
	mw := NewAPIKeyMiddleware(&fakeValidator{})
	rec, _ := runRequest(t, mw, func(r *http.Request) {})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, esperado 401", rec.Code)
	}
}

func TestAPIKeyMiddlewareInvalidKey(t *testing.T) {
	mw := NewAPIKeyMiddleware(&fakeValidator{err: errors.New("revoked")})
	rec, _ := runRequest(t, mw, func(r *http.Request) {
		r.Header.Set("X-API-Key", "vtk_test_revoked")
	})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, esperado 401", rec.Code)
	}
}

func TestRequireScope(t *testing.T) {
	tests := []struct {
		name       string
		service    *security.Service
		user       *security.User
		wantStatus int
	}{
		{"service with scope", &security.Service{Scopes: []string{"finance:read"}}, nil, http.StatusOK},
		{"service with wildcard", &security.Service{Scopes: []string{"finance:*"}}, nil, http.StatusOK},
		{"service without scope", &security.Service{Scopes: []string{"oms:read"}}, nil, http.StatusForbidden},
		{"jwt user passes", nil, &security.User{ID: "u1", Role: "admin"}, http.StatusOK},
		{"anonymous", nil, nil, http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			if tt.service != nil {
				c.Set(security.ServiceContextKey, *tt.service)
			}
			if tt.user != nil {
				c.Set(security.UserContextKey, *tt.user)
			}

			handler := RequireScope("finance:read")(func(c echo.Context) error {
				return c.JSON(http.StatusOK, map[string]string{"ok": "true"})
			})
			if err := handler(c); err != nil {
				t.Fatalf("handler retornou erro: %v", err)
			}
			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, esperado %d; body: %s", rec.Code, tt.wantStatus, rec.Body.String())
			}
		})
	}
}

func TestHybridAuthMiddlewareAPIKeyPath(t *testing.T) {
	validator := &fakeValidator{service: &security.Service{ID: "k1", Name: "erp-acme", Scopes: []string{"*"}}}
	mw := NewHybridAuthMiddleware(AuthConfig{JWTSecret: "test-secret"}, validator)

	rec, svc := runRequest(t, mw, func(r *http.Request) {
		r.Header.Set("Authorization", "Bearer vtk_test_hybrid")
	})
	if rec.Code != http.StatusOK || svc == nil {
		t.Fatalf("status = %d, service = %+v", rec.Code, svc)
	}
}

func TestHybridAuthMiddlewareRejectsGarbageJWT(t *testing.T) {
	mw := NewHybridAuthMiddleware(AuthConfig{JWTSecret: "test-secret"}, &fakeValidator{})
	rec, _ := runRequest(t, mw, func(r *http.Request) {
		r.Header.Set("Authorization", "Bearer not-a-real-token")
	})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, esperado 401", rec.Code)
	}
}
