package http

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/vertikon/sdk-hulk.vertikon.com.br/state"
)

// O hook state.TenantFromContext é o funil do RLS por query: precisa achar o
// tenant tanto pela chave tipada (middleware HTTP) quanto pela chave string
// "tenant_id" (consumers NATS) — e recusar valores que não são uuid.
func TestTenantFromContextFallback(t *testing.T) {
	tid := uuid.New()

	if got, ok := state.TenantFromContext(WithTenantID(context.Background(), tid)); !ok || got != tid.String() {
		t.Fatalf("chave tipada: esperava %s, veio %q/%v", tid, got, ok)
	}
	//lint:ignore SA1029 exercita de propósito o fallback pela chave string dos consumers NATS
	if got, ok := state.TenantFromContext(context.WithValue(context.Background(), "tenant_id", tid.String())); !ok || got != tid.String() {
		t.Fatalf("chave string (consumer NATS): esperava %s, veio %q/%v", tid, got, ok)
	}
	for _, bad := range []string{"default", "global", "", "unknown"} {
		//lint:ignore SA1029 idem: fallback pela chave string
		if _, ok := state.TenantFromContext(context.WithValue(context.Background(), "tenant_id", bad)); ok {
			t.Fatalf("valor não-uuid %q não pode virar contexto RLS", bad)
		}
	}
	if _, ok := state.TenantFromContext(context.Background()); ok {
		t.Fatal("contexto vazio não pode ter tenant")
	}
}
