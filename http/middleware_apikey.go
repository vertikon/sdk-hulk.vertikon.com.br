package http

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/vertikon/sdk-hulk.vertikon.com.br/security"
)

// extractAPIKey busca a API key no header X-API-Key ou em Authorization: Bearer vtk_...
func extractAPIKey(c echo.Context) string {
	if key := c.Request().Header.Get("X-API-Key"); key != "" {
		return key
	}
	authHeader := c.Request().Header.Get("Authorization")
	parts := strings.Split(authHeader, " ")
	if len(parts) == 2 && parts[0] == "Bearer" && security.IsAPIKey(parts[1]) {
		return parts[1]
	}
	return ""
}

// NewAPIKeyMiddleware cria o middleware de autenticação máquina-a-máquina (API key).
// Exige uma API key válida; injeta security.Service no contexto.
func NewAPIKeyMiddleware(validator security.APIKeyValidator) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			key := extractAPIKey(c)
			if key == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Missing API key"})
			}
			svc, err := resolveService(c, validator, key)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid API key"})
			}
			c.Set(security.ServiceContextKey, *svc)
			return next(c)
		}
	}
}

// NewHybridAuthMiddleware aceita JWT de usuário OU API key de sistema na mesma rota.
// API keys (prefixo vtk_) são validadas pelo validator; demais tokens seguem o fluxo JWT padrão.
func NewHybridAuthMiddleware(cfg AuthConfig, validator security.APIKeyValidator) echo.MiddlewareFunc {
	jwtMiddleware := NewAuthMiddleware(cfg)
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		jwtNext := jwtMiddleware(next)
		return func(c echo.Context) error {
			if key := extractAPIKey(c); key != "" {
				svc, err := resolveService(c, validator, key)
				if err != nil {
					return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid API key"})
				}
				c.Set(security.ServiceContextKey, *svc)
				return next(c)
			}
			return jwtNext(c)
		}
	}
}

// RequireScope autoriza a requisição por scope ("dominio:acao").
// Sistemas (API key) precisam do scope; usuários (JWT) passam — RBAC/Feature Manager cobre humanos.
// Sem identidade nenhuma no contexto, responde 401.
func RequireScope(required string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if svc := GetServiceFromContext(c); svc != nil {
				if !svc.HasScope(required) {
					return c.JSON(http.StatusForbidden, map[string]string{
						"error": "Insufficient scope",
						"scope": required,
					})
				}
				return next(c)
			}
			if user := GetUserFromContext(c); user != nil {
				return next(c)
			}
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Authentication required"})
		}
	}
}

func resolveService(c echo.Context, validator security.APIKeyValidator, key string) (*security.Service, error) {
	if validator == nil {
		validator = security.GetAPIKeyValidator()
	}
	if validator == nil {
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "no API key validator registered")
	}
	return validator.ValidateAPIKey(c.Request().Context(), key)
}

// GetServiceFromContext extrai o sistema consumidor autenticado do contexto Echo
func GetServiceFromContext(c echo.Context) *security.Service {
	svc, ok := c.Get(security.ServiceContextKey).(security.Service)
	if !ok {
		return nil
	}
	return &svc
}

// GetServiceFromSDKContext extrai o sistema consumidor autenticado do contexto SDK
func GetServiceFromSDKContext(c Context) *security.Service {
	if echoCtx, ok := c.(*EchoContext); ok {
		return GetServiceFromContext(echoCtx.ctx)
	}
	return nil
}
