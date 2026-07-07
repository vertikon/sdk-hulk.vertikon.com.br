package http

import (
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/vertikon/sdk-hulk.vertikon.com.br/security"
)

func TestGetUserFromContext_ValidUser(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	user := security.User{
		ID:    "user-123",
		Email: "test@example.com",
		Role:  "admin",
	}
	c.Set(security.UserContextKey, user)

	result := GetUserFromContext(c)
	assert.NotNil(t, result)
	assert.Equal(t, "user-123", result.ID)
	assert.Equal(t, "test@example.com", result.Email)
	assert.Equal(t, "admin", result.Role)
}

func TestGetUserFromContext_NoUser(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	result := GetUserFromContext(c)
	assert.Nil(t, result)
}

func TestGetUserFromSDKContext_ValidUser(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	echoCtx := e.NewContext(req, rec)

	user := security.User{
		ID:    "user-456",
		Email: "user@example.com",
		Role:  "user",
	}
	echoCtx.Set(security.UserContextKey, user)

	sdkCtx := &EchoContext{ctx: echoCtx}
	result := GetUserFromSDKContext(sdkCtx)
	assert.NotNil(t, result)
	assert.Equal(t, "user-456", result.ID)
	assert.Equal(t, "user@example.com", result.Email)
	assert.Equal(t, "user", result.Role)
}

func TestGetUserFromSDKContext_NoUser(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	echoCtx := e.NewContext(req, rec)

	sdkCtx := &EchoContext{ctx: echoCtx}
	result := GetUserFromSDKContext(sdkCtx)
	assert.Nil(t, result)
}

type InvalidContext struct{}

func (c *InvalidContext) Bind(i interface{}) error                            { return nil }
func (c *InvalidContext) JSON(code int, i interface{}) error                  { return nil }
func (c *InvalidContext) Param(name string) string                            { return "" }
func (c *InvalidContext) QueryParam(name string) string                       { return "" }
func (c *InvalidContext) FormValue(name string) string                        { return "" }
func (c *InvalidContext) FormFile(name string) (*multipart.FileHeader, error) { return nil, nil }
func (c *InvalidContext) Request() *http.Request                              { return nil }
func (c *InvalidContext) Response() http.ResponseWriter                       { return nil }
func (c *InvalidContext) NoContent(code int) error                            { return nil }

func TestGetUserFromSDKContext_InvalidContext(t *testing.T) {
	// Criar um contexto que não é EchoContext
	result := GetUserFromSDKContext(&InvalidContext{})
	assert.Nil(t, result)
}
