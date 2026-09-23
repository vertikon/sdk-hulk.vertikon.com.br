package state

import "testing"

// O roteamento em si (Exec/Query no pool owner durante o boot) é coberto pelo
// smoke de integração/deploy; aqui garantimos a máquina de estados do gate —
// se ela quebrar, ou o boot fica preso no owner (RLS burlada em runtime) ou
// as migrations voltam a rodar como app_login (permission denied).
func TestBootPoolGating(t *testing.T) {
	s := &PostgresStore{}

	// Sem MIGRATIONS_DATABASE_URL (migPool nil) o gate NUNCA ativa, mesmo em boot.
	s.booting.Store(true)
	if s.bootPool() != nil {
		t.Fatal("sem migPool, bootPool deve ser nil (comportamento pré-flip)")
	}

	// EndBoot encerra a fase e é idempotente (sem pool p/ fechar, sem panic).
	s.EndBoot()
	if s.booting.Load() {
		t.Fatal("EndBoot deve encerrar a fase de boot")
	}
	s.EndBoot()
	if s.bootPool() != nil {
		t.Fatal("após EndBoot, bootPool deve ser nil para sempre")
	}
}
