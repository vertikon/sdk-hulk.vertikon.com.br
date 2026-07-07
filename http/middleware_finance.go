package http

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/vertikon/sdk-hulk.vertikon.com.br/security"
)

// papéis com acesso ao domínio financeiro.
var (
	financeReadRoles  = map[string]bool{"admin": true, "service": true, "financeiro": true, "contabilidade": true}
	financeWriteRoles = map[string]bool{"admin": true, "service": true, "financeiro": true}
)

// RequireFinanceRole é um middleware de autorização por PAPEL para o domínio financeiro,
// ciente do método HTTP:
//   - Leitura (GET/HEAD/OPTIONS): admin, service, financeiro, contabilidade.
//   - Escrita (POST/PUT/PATCH/DELETE): admin, service, financeiro — "contabilidade" é READ-ONLY.
//
// Serviços M2M (API key, validados pelo external-gateway) passam direto. Deve ser aplicado
// APÓS o middleware de autenticação (que injeta o usuário no contexto).
func RequireFinanceRole() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// serviço máquina-a-máquina: escopo já validado na borda.
			if _, ok := c.Get(security.ServiceContextKey).(security.Service); ok {
				return next(c)
			}
			u, ok := c.Get(security.UserContextKey).(security.User)
			if !ok {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "não autenticado"})
			}
			allowed := financeWriteRoles
			switch c.Request().Method {
			case http.MethodGet, http.MethodHead, http.MethodOptions:
				allowed = financeReadRoles
			}
			if !allowed[u.Role] {
				return c.JSON(http.StatusForbidden, map[string]string{
					"error": "papel sem permissão financeira (contabilidade é somente-leitura)",
				})
			}
			return next(c)
		}
	}
}
