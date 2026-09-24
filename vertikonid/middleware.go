package vertikonid

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

type ctxKey struct{}

// FromContext devolve as claims validadas pelo Middleware.
func FromContext(ctx context.Context) (*Claims, bool) {
	c, ok := ctx.Value(ctxKey{}).(*Claims)
	return c, ok
}

// Middleware exige `Authorization: Bearer <token do Vertikon ID>` válido e
// bloqueia tenant suspenso pelo billing. Serve para net/http direto e para
// echo/chi via adaptador (ex.: echo.WrapMiddleware(vertikonid.Middleware(v))).
//
//	401 missing_token / invalid_token — sem token, assinatura/exp/issuer inválidos
//	403 tenant_suspended              — plan=suspended (contrato §3d do DEVKIT do ID)
func Middleware(v *Verifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
			if !ok || strings.TrimSpace(token) == "" {
				deny(w, http.StatusUnauthorized, "missing_token")
				return
			}
			c, err := v.Verify(r.Context(), strings.TrimSpace(token))
			if err != nil {
				deny(w, http.StatusUnauthorized, "invalid_token")
				return
			}
			if c.Suspended() {
				deny(w, http.StatusForbidden, "tenant_suspended")
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, c)))
		})
	}
}

func deny(w http.ResponseWriter, status int, code string) {
	if status == http.StatusUnauthorized {
		w.Header().Set("WWW-Authenticate", `Bearer realm="vertikon-id"`)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": code})
}
