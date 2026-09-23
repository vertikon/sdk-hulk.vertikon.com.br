package state

// tenant.go — encanamento de tenant para RLS (blueprint F1.2, estágio 1).
//
// Problema raiz: as políticas RLS do padrão A dependem de
// current_setting('app.tenant_id'), mas os métodos não-transacionais do
// PostgresStore (Query/QueryRow/Exec) iam direto ao pool SEM setar a variável —
// com enforcement ligado, todo read fora de transação voltaria vazio.
//
// Solução em duas partes:
//  1. WithTenant — helper explícito (promove o withTenantTx que cada módulo
//     reimplementava): transação + SET LOCAL app.tenant_id.
//  2. Propagação AUTOMÁTICA do tenant do request context nos métodos do pool,
//     GATED por HULK_RLS_CTX=1 (default OFF = comportamento atual, deploy sem
//     risco). Com a env ligada: acquire da conexão → set_config → query →
//     reset → release (o reset impede vazar tenant entre requests do pool).
//
// O tenant é lido do contexto via o hook TenantFromContext, injetado pelo
// pacote sdk-hulk/http (evita dependência state→http/echo).

import (
	"context"
	"os"
	"strings"
)

// TenantFromContext é injetado pelo sdk-hulk/http (init) — devolve o tenant do
// request context, quando presente. Nil-safe: sem hook, nada é propagado.
var TenantFromContext func(ctx context.Context) (string, bool)

// rlsCtxEnabled — HULK_RLS_CTX=1|true liga a propagação automática por query.
func rlsCtxEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("HULK_RLS_CTX")))
	return v == "1" || v == "true"
}

func tenantOf(ctx context.Context) (string, bool) {
	if !rlsCtxEnabled() || TenantFromContext == nil {
		return "", false
	}
	return TenantFromContext(ctx)
}

// jobsCtxKey marca um contexto como caminho de WORKER cross-tenant/sem-tenant.
type jobsCtxKey struct{}

// WithJobs marca o contexto para rotear as queries ao pool de JOBS (conexão do
// owner, que bypassa RLS). Use nos schedulers/consumers que varrem TODOS os
// tenants (ex.: wa_broadcasts, cobranças vencidas, geradores de relatório) e na
// auth pré-tenant (iam_users/iam_api_keys). Sem JOBS_DATABASE_URL, é no-op (cai
// no pool normal — comportamento atual).
func WithJobs(ctx context.Context) context.Context {
	return context.WithValue(ctx, jobsCtxKey{}, true)
}

func jobsOf(ctx context.Context) bool {
	v, _ := ctx.Value(jobsCtxKey{}).(bool)
	return v
}

// WithTenant executa fn numa transação com app.tenant_id setado (escopo da
// transação — some no COMMIT/ROLLBACK). Use para fluxos de escrita ou quando o
// tenant não está no request context (workers, consumers NATS).
func WithTenant(ctx context.Context, s Store, tenantID string, fn func(tx Tx) error) error {
	tx, err := s.BeginTx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Exec(ctx, "SELECT set_config('app.tenant_id', $1, true)", tenantID); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}
