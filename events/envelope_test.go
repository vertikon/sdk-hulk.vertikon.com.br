package events

import (
	"context"
	"testing"

	"github.com/google/uuid"

	sdk_http "github.com/vertikon/sdk-hulk.vertikon.com.br/http"
)

// Envelope: event_id sempre presente; tenant do request context quando houver;
// causation encadeável. (correlation depende de span OTel ativo — fora do unit.)
func TestEnvelope(t *testing.T) {
	e := Envelope(context.Background(), map[string]string{"k": "v"})
	if e.Schema != EnvelopeSchema || e.EventID == "" || e.OccurredAt.IsZero() {
		t.Fatalf("envelope incompleto: %+v", e)
	}
	if e.TenantID != "" {
		t.Fatal("sem tenant no ctx, TenantID deve ficar vazio")
	}

	tid := uuid.New()
	e2 := Envelope(sdk_http.WithTenantID(context.Background(), tid), nil)
	if e2.TenantID != tid.String() {
		t.Fatalf("tenant do ctx não propagado: %s", e2.TenantID)
	}
	if e2.EventID == e.EventID {
		t.Fatal("event_id deve ser único")
	}

	e3 := e2.Caused(e.EventID)
	if e3.CausationID != e.EventID {
		t.Fatal("Caused deve preencher causation_id")
	}
}
