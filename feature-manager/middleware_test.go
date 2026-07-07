package featuremanager

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/vertikon/sdk-hulk.vertikon.com.br/security"
)

func TestRequireModule_WithoutAuth(t *testing.T) {
	// Setup
	e := echo.New()
	fm := NewService()

	// Rota protegida sem autenticação
	e.GET("/test", func(c echo.Context) error {
		return c.JSON(200, map[string]string{"status": "ok"})
	}, RequireModule(fm, "mod.test"))

	// Request sem autenticação
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	// Deve retornar 401 (não autenticado)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestRequireModule_WithAuth_NoTenantID(t *testing.T) {
	// Setup
	e := echo.New()
	fm := NewService()

	// Rota protegida
	e.GET("/test", func(c echo.Context) error {
		return c.JSON(200, map[string]string{"status": "ok"})
	}, RequireModule(fm, "mod.test"))

	// Request com usuário mas sem TenantID (modo desenvolvimento)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Injeta usuário sem TenantID (modo desenvolvimento)
	user := security.User{
		ID:       "user-123",
		Email:    "test@example.com",
		Role:     "user",
		TenantID: uuid.Nil, // Modo desenvolvimento
	}
	c.Set(security.UserContextKey, user)

	// Deve permitir acesso (stub mode)
	handler := RequireModule(fm, "mod.test")(func(c echo.Context) error {
		return c.JSON(200, map[string]string{"status": "ok"})
	})
	err := handler(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRequireModule_WithAuth_WithTenantID_StubMode(t *testing.T) {
	// Setup
	e := echo.New()
	fm := NewService()

	// Rota protegida
	e.GET("/test", func(c echo.Context) error {
		return c.JSON(200, map[string]string{"status": "ok"})
	}, RequireModule(fm, "mod.test"))

	// Request com usuário e TenantID (mas stub mode permite tudo)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Injeta usuário com TenantID
	tenantID := uuid.New()
	user := security.User{
		ID:       "user-123",
		Email:    "test@example.com",
		Role:     "user",
		TenantID: tenantID,
	}
	c.Set(security.UserContextKey, user)

	// Em stub mode, sempre permite (HasAccess retorna true)
	handler := RequireModule(fm, "mod.test")(func(c echo.Context) error {
		return c.JSON(200, map[string]string{"status": "ok"})
	})
	err := handler(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRequireModule_MultipleModules(t *testing.T) {
	// Testa que múltiplos middlewares podem ser aplicados
	e := echo.New()
	fm := NewService()

	// Rota que requer 2 módulos
	e.GET("/advanced", func(c echo.Context) error {
		return c.JSON(200, map[string]string{"status": "ok"})
	},
		RequireModule(fm, "mod.fiscal.issuer"),
		RequireModule(fm, "mod.ai.vision"),
	)

	// Request com usuário (stub mode permite tudo)
	req := httptest.NewRequest(http.MethodGet, "/advanced", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	user := security.User{
		ID:       "user-123",
		Email:    "test@example.com",
		Role:     "user",
		TenantID: uuid.Nil, // Modo desenvolvimento
	}
	c.Set(security.UserContextKey, user)

	// Deve permitir acesso
	handler1 := RequireModule(fm, "mod.fiscal.issuer")(
		RequireModule(fm, "mod.ai.vision")(
			func(c echo.Context) error {
				return c.JSON(200, map[string]string{"status": "ok"})
			},
		),
	)
	err := handler1(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}
