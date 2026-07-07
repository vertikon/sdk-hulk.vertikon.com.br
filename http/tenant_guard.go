package http

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// NewTenantGuardMiddleware impõe isolamento de tenant no plano interno (/api/v1).
//
// Quando o chamador é um JWT de USUÁRIO com tenant vinculado (TenantID != uuid.Nil),
// o guard garante o isolamento:
//   - se o header X-Tenant-ID estiver ausente, injeta o tenant do usuário;
//   - se estiver presente e divergir, responde 403.
//
// É backward-compatible por construção: usuários SEM tenant (TenantID == uuid.Nil —
// o caso de todos os usuários antes do vínculo, i.e. global/superadmin) e SERVIÇOS
// (API key, cujo tenant já é validado na borda /ext/v1 por CanAccessTenant) passam
// sem qualquer alteração.
func NewTenantGuardMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if user := GetUserFromContext(c); user != nil && user.TenantID != uuid.Nil {
				ut := user.TenantID.String()
				switch c.Request().Header.Get("X-Tenant-ID") {
				case "":
					c.Request().Header.Set("X-Tenant-ID", ut)
				case ut:
					// header confere com o tenant do usuário — ok
				default:
					return c.JSON(http.StatusForbidden, map[string]string{
						"error": "tenant não permitido para este usuário",
					})
				}
			}
			return next(c)
		}
	}
}
