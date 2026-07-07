package featuremanager

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	sdkhttp "github.com/vertikon/sdk-hulk.vertikon.com.br/http"
)

// RequireModule cria uma barreira HTTP. Se o cliente não tiver o módulo, recebe 403.
func RequireModule(manager FeatureManager, moduleCode string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// 1. Recupera o usuário autenticado do contexto (injetado pelo Auth Middleware)
			user := sdkhttp.GetUserFromContext(c)
			if user == nil {
				// Se não tem usuário, o Auth Middleware falhou ou não foi chamado antes
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error":   "authentication_required",
					"details": "User identity not found in context. Ensure AuthMiddleware is applied before RequireModule.",
				})
			}

			// 2. Verifica se o usuário tem TenantID
			// Se não tiver, assume que é desenvolvimento e permite (compatibilidade)
			// Em desenvolvimento, TenantID será uuid.Nil, então sempre permite
			tenantID := user.TenantID
			if tenantID == uuid.Nil {
				// Modo desenvolvimento: permite acesso (stub mode)
				return next(c)
			}

			// 3. Consulta o Feature Manager (Hoje é Stub, amanhã é Redis/Banco)
			if !manager.HasAccess(c.Request().Context(), tenantID, moduleCode) {
				return c.JSON(http.StatusForbidden, map[string]interface{}{
					"error":       "module_not_enabled",
					"message":     "Seu plano atual não inclui acesso a este módulo.",
					"module_code": moduleCode,
					"action":      "CONTACT_SALES",
					"tenant_id":   user.TenantID.String(),
				})
			}

			// 4. Tudo certo, passa para o Controller
			return next(c)
		}
	}
}
