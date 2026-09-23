package state_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	sdk_http "github.com/vertikon/sdk-hulk.vertikon.com.br/http"
	"github.com/vertikon/sdk-hulk.vertikon.com.br/state"
)

// fakes mínimos do Store/Tx para exercitar o WithTenant sem banco.
type fakeTx struct {
	execs     []string
	committed bool
	rolled    bool
}

func (t *fakeTx) Exec(_ context.Context, q string, _ ...interface{}) error {
	t.execs = append(t.execs, q)
	return nil
}
func (t *fakeTx) QueryRow(context.Context, string, ...interface{}) state.RowScanner { return nil }
func (t *fakeTx) Query(context.Context, string, ...interface{}) (state.Rows, error) {
	return nil, nil
}
func (t *fakeTx) Commit() error   { t.committed = true; return nil }
func (t *fakeTx) Rollback() error { t.rolled = true; return nil }

type fakeStore struct {
	state.Store
	tx *fakeTx
}

func (s *fakeStore) BeginTx(context.Context) (state.Tx, error) { return s.tx, nil }

// WithTenant: seta app.tenant_id na transação, roda fn e commita; erro → rollback.
func TestWithTenant(t *testing.T) {
	tx := &fakeTx{}
	ran := false
	err := state.WithTenant(context.Background(), &fakeStore{tx: tx}, "t-1", func(state.Tx) error {
		ran = true
		return nil
	})
	if err != nil || !ran || !tx.committed || tx.rolled {
		t.Fatalf("fluxo feliz: err=%v ran=%v committed=%v rolled=%v", err, ran, tx.committed, tx.rolled)
	}
	if len(tx.execs) == 0 || tx.execs[0] != "SELECT set_config('app.tenant_id', $1, true)" {
		t.Fatalf("primeira instrução deveria setar o tenant: %v", tx.execs)
	}
}

// O hook do http deve estar injetado (init) e ler o tenant do request context;
// e a propagação automática fica DESLIGADA sem HULK_RLS_CTX.
func TestTenantHook(t *testing.T) {
	if state.TenantFromContext == nil {
		t.Fatal("hook TenantFromContext não injetado pelo sdk-hulk/http")
	}
	tid := uuid.New()
	got, ok := state.TenantFromContext(sdk_http.WithTenantID(context.Background(), tid))
	if !ok || got != tid.String() {
		t.Fatalf("hook não leu o tenant do ctx: %v %v", got, ok)
	}
	if _, ok := state.TenantFromContext(context.Background()); ok {
		t.Fatal("sem tenant no ctx deveria ser ok=false")
	}
}
