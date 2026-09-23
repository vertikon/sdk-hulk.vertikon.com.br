package state

import (
	"context"
	"testing"
)

// WithJobs marca o ctx como caminho de worker; jobsOf lê. Sem marca = false
// (comportamento padrão: cai no pool normal). O roteamento ao jobsPool depende
// disto — se WithJobs não "pegar", os workers iriam ao pool app_login e sob o
// flip voltariam zero linhas.
func TestWithJobs_RoundTrip(t *testing.T) {
	base := context.Background()
	if jobsOf(base) {
		t.Fatal("contexto virgem não deveria ser de jobs")
	}
	if !jobsOf(WithJobs(base)) {
		t.Fatal("WithJobs deveria marcar o contexto")
	}
	// a marca sobrevive a um contexto derivado (ex.: com deadline/valores)
	type outraChave struct{}
	derived := context.WithValue(WithJobs(base), outraChave{}, 1)
	if !jobsOf(derived) {
		t.Fatal("a marca de jobs deveria sobreviver a contextos derivados")
	}
}
