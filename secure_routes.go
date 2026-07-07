package hulk

import (
	"context"
	"os"
	"strings"

	httpx "github.com/vertikon/sdk-hulk.vertikon.com.br/http"
	"go.uber.org/zap"
)

// SecureRoutes aplica autenticação híbrida (JWT de usuário OU API key M2M) a um
// grupo de rotas de um módulo. É a receita padrão para fechar o gap de módulos
// que registravam rotas sem middleware (ver PLANEJAMENTO-EXPOSICAO §0.2).
//
// Escape hatch para desenvolvimento local:
//
//	HULK_ALLOW_ANON_MODULES="payroll-engine,audit-log"  // por módulo
//	HULK_ALLOW_ANON_MODULES="*"                          // tudo (NUNCA em produção)
//
// Retorna true se o middleware foi aplicado.
func SecureRoutes(ctx Context, moduleID string, router httpx.Router) bool {
	if allowAnon(moduleID) {
		ctx.Log().Warn("Auth DESABILITADA por HULK_ALLOW_ANON_MODULES (apenas desenvolvimento)",
			zap.String("module", moduleID))
		return false
	}

	jwtSecret := os.Getenv("HULK_JWT_SECRET")
	if jwtSecret == "" && ctx.Secrets() != nil {
		if secret, err := ctx.Secrets().Get(context.Background(), "endurance/dev/vertikon/jwt/secret"); err == nil {
			jwtSecret = secret
		}
	}
	if jwtSecret == "" {
		ctx.Log().Warn("JWT Secret não configurado - rotas do módulo SEM autenticação",
			zap.String("module", moduleID))
		return false
	}

	group, ok := router.(*httpx.EchoGroup)
	if !ok {
		ctx.Log().Warn("Router não é EchoGroup - auth não aplicada",
			zap.String("module", moduleID))
		return false
	}
	group.UseEchoMiddleware(httpx.NewHybridAuthMiddleware(httpx.AuthConfig{JWTSecret: jwtSecret}, nil))
	// Guard de tenant: no-op p/ usuários sem tenant (global) e serviços; impõe
	// isolamento p/ usuários com tenant vinculado (multi-cliente).
	group.UseEchoMiddleware(httpx.NewTenantGuardMiddleware())
	ctx.Log().Info("Autenticação híbrida + guard de tenant aplicados às rotas do módulo",
		zap.String("module", moduleID))
	return true
}

func allowAnon(moduleID string) bool {
	raw := os.Getenv("HULK_ALLOW_ANON_MODULES")
	if raw == "" {
		return false
	}
	for _, entry := range strings.Split(raw, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "*" || entry == moduleID {
			return true
		}
	}
	return false
}
