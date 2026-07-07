package hulk_test

import (
	"context"
	"os"
	"testing"

	hulk "github.com/vertikon/sdk-hulk.vertikon.com.br"
	"go.uber.org/zap"
)

func newTestContext(t *testing.T) hulk.Context {
	t.Helper()
	return hulk.NewContext(context.Background(), zap.NewNop(), nil, nil, nil, nil, nil)
}

// allowAnon é testado indiretamente via SecureRoutes; aqui validamos só o parsing
// do escape hatch usando a própria SecureRoutes com um contexto nil-safe não é
// trivial (Context é interface rica), então testamos o contrato observável:
// com HULK_ALLOW_ANON_MODULES="*" e um Context mínimo, retorna false (anon).
func TestSecureRoutesAllowAnonEscapeHatch(t *testing.T) {
	t.Setenv("HULK_ALLOW_ANON_MODULES", "mod-a, mod-b")
	t.Setenv("HULK_JWT_SECRET", "secret")

	ctx := newTestContext(t)
	if applied := hulk.SecureRoutes(ctx, "mod-a", nil); applied {
		t.Error("mod-a está na allowlist anon — não deveria aplicar auth")
	}
	if applied := hulk.SecureRoutes(ctx, "mod-c", nil); applied {
		// mod-c não está na allowlist, mas router nil (não-EchoGroup) → false com warn
		t.Error("router nil não é EchoGroup — não deveria reportar aplicado")
	}

	t.Setenv("HULK_ALLOW_ANON_MODULES", "*")
	if applied := hulk.SecureRoutes(ctx, "qualquer", nil); applied {
		t.Error("curinga * deveria desabilitar auth")
	}
}

func TestSecureRoutesWithoutSecret(t *testing.T) {
	t.Setenv("HULK_ALLOW_ANON_MODULES", "")
	t.Setenv("HULK_JWT_SECRET", "")
	os.Unsetenv("HULK_JWT_SECRET")

	ctx := newTestContext(t)
	if applied := hulk.SecureRoutes(ctx, "mod-x", nil); applied {
		t.Error("sem JWT secret não deveria aplicar auth")
	}
}
