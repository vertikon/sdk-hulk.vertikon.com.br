//go:build integration

// Smoke de isolamento RLS (o "verde/vermelho" do flip). Prova, contra Postgres
// real, que sob um role NÃO-owner (app_login) a RLS Padrão A vale de verdade:
//   1. tenant A vê só as linhas de A;
//   2. tenant B vê só as de B;
//   3. SEM tenant no contexto -> ZERO linhas (o risco dos caminhos sem-tenant);
//   4. o OWNER ainda vê tudo (bypassa por ownership -> é o caminho dos workers).
//
// Roda num BRANCH DESCARTÁVEL do Neon (cria role/tabela; a tabela é dropada; o role
// app_login recebe uma senha de teste). NÃO rodar contra o branch de serviço/prod.
//   DATABASE_URL=postgres://<owner>...  go test -tags integration \
//     -run TestRLSSmoke ./pkg/sdk-hulk/state/...
package state

import (
	"context"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func withUser(dsn, user, pw string) (string, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return "", err
	}
	u.User = url.UserPassword(user, pw)
	return u.String(), nil
}

func TestRLSSmoke_Isolation(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL não setado — pulando smoke RLS (rodar em BRANCH descartável)")
	}
	// RLS-GUARD-01(a): este teste CRIA roles, troca senha do app_login e DROPa tabela.
	// Gate anti-produção (a auditoria registrou um falso-verde por rodar no lugar errado):
	// exige ack explícito + recusa o endpoint de serviço/prod do eduue.
	if os.Getenv("RLS_SMOKE_ALLOW") != "1" {
		t.Skip("guard: exporte RLS_SMOKE_ALLOW=1 e aponte DATABASE_URL para um BRANCH DESCARTÁVEL do Neon")
	}
	if strings.Contains(dsn, "ep-green-morning") {
		t.Fatal("guard: DATABASE_URL aponta o endpoint de SERVIÇO/PROD do eduue — use um branch descartável do Neon")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	owner, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("owner connect: %v", err)
	}
	defer owner.Close(context.Background())

	const pw = "rls_smoke_pw_2607"
	exec := func(sql string, args ...any) {
		if _, e := owner.Exec(ctx, sql, args...); e != nil {
			t.Fatalf("owner exec %q: %v", sql, e)
		}
	}
	// role app_user (das migrations) + app_login não-owner com senha de teste
	exec(`DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='app_user') THEN CREATE ROLE app_user NOINHERIT; END IF; END $$`)
	exec(`DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='app_login') THEN CREATE ROLE app_login LOGIN; END IF; END $$`)
	exec(`ALTER ROLE app_login LOGIN PASSWORD '` + pw + `'`)
	exec(`GRANT app_user TO app_login`)

	// tabela RLS de teste (mesmo padrão das 457 tabelas de prod)
	exec(`DROP TABLE IF EXISTS rls_smoke`)
	exec(`CREATE TABLE rls_smoke (id serial PRIMARY KEY, tenant_id text NOT NULL, val text)`)
	exec(`ALTER TABLE rls_smoke ENABLE ROW LEVEL SECURITY`)
	exec(`CREATE POLICY rls_smoke_iso ON rls_smoke USING (tenant_id::text = current_setting('app.tenant_id', true))`)
	exec(`GRANT SELECT, INSERT, UPDATE, DELETE ON rls_smoke TO app_user`)
	exec(`GRANT USAGE, SELECT ON SEQUENCE rls_smoke_id_seq TO app_user`)
	exec(`INSERT INTO rls_smoke (tenant_id, val) VALUES ('tnA','a1'),('tnA','a2'),('tnB','b1')`)
	defer func() { _, _ = owner.Exec(context.Background(), `DROP TABLE IF EXISTS rls_smoke`) }()

	// conecta como app_login (não-owner) — aqui a RLS passa a valer
	loginDSN, err := withUser(dsn, "app_login", pw)
	if err != nil {
		t.Fatalf("build login dsn: %v", err)
	}
	login, err := pgx.Connect(ctx, loginDSN)
	if err != nil {
		t.Fatalf("app_login connect (role tem senha/permite login neste endpoint?): %v", err)
	}
	defer login.Close(context.Background())

	count := func(tenant string) int {
		if _, e := login.Exec(ctx, `SELECT set_config('app.tenant_id',$1,false)`, tenant); e != nil {
			t.Fatalf("set_config(%q): %v", tenant, e)
		}
		var n int
		if e := login.QueryRow(ctx, `SELECT count(*) FROM rls_smoke`).Scan(&n); e != nil {
			t.Fatalf("count (tenant=%q): %v", tenant, e)
		}
		return n
	}

	// 1+2: cada tenant vê só o seu
	if got := count("tnA"); got != 2 {
		t.Fatalf("tenant A deveria ver 2, viu %d — RLS não isolou (app_login é owner/BYPASSRLS?)", got)
	}
	if got := count("tnB"); got != 1 {
		t.Fatalf("tenant B deveria ver 1, viu %d", got)
	}
	// 3: SEM tenant -> 0 (o modo de falha dos caminhos sem-tenant sob o flip)
	if got := count(""); got != 0 {
		t.Fatalf("sem tenant deveria ver 0, viu %d — enforcement não estava ativo", got)
	}
	// 4: owner bypassa por ownership -> vê as 3 (é o caminho dos workers no flip)
	var all int
	if e := owner.QueryRow(ctx, `SELECT count(*) FROM rls_smoke`).Scan(&all); e != nil {
		t.Fatalf("owner count: %v", e)
	}
	if all != 3 {
		t.Fatalf("owner deveria ver as 3 (bypassa RLS), viu %d", all)
	}
}
