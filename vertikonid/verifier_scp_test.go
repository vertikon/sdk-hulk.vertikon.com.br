package vertikonid

import "testing"

// TestHasScopeLeScpLista: o hub emite `scp` como lista; HasScope precisa casar por ela
// e continuar aceitando `scope` separado por espaço.
func TestHasScopeLeScpLista(t *testing.T) {
	c := Claims{Scp: []string{"recap.intake.public"}}
	if !c.HasScope("recap.intake.public") {
		t.Fatal("scp em lista não foi reconhecido")
	}
	if c.HasScope("idp:admin") {
		t.Fatal("escopo ausente não pode casar")
	}
	if !(Claims{Scope: "a b"}).HasScope("b") {
		t.Fatal("scope string deixou de funcionar")
	}
}
