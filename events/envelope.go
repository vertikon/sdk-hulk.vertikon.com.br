package events

// envelope.go — envelope padrão de eventos de domínio (blueprint F1.4).
// Opt-in: NÃO muda a assinatura de Publish (os ~250 subjects existentes seguem
// publicando payload cru). Eventos NOVOS devem publicar Envelope(ctx, data):
//
//	bus.Publish(TopicX, events.Envelope(ctx, payload))
//
// O envelope carrega os metadados de correlação que hoje não existem em nenhum
// payload: event_id (idempotência), correlation_id (trace OTel do request),
// tenant_id (do request context) e occurred_at. Consumidores detectam o
// envelope pela presença de "schema":"vtk.event.envelope.v1".

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"go.opentelemetry.io/otel/trace"

	sdk_http "github.com/vertikon/sdk-hulk.vertikon.com.br/http"
)

// EnvelopeSchema identifica o envelope no payload publicado.
const EnvelopeSchema = "vtk.event.envelope.v1"

// EventEnvelope é o wire-format do envelope (JSON via Publish).
type EventEnvelope struct {
	Schema        string      `json:"schema"`
	EventID       string      `json:"event_id"`
	CorrelationID string      `json:"correlation_id,omitempty"` // trace-id OTel do request
	CausationID   string      `json:"causation_id,omitempty"`   // event_id do evento que causou este
	TenantID      string      `json:"tenant_id,omitempty"`
	OccurredAt    time.Time   `json:"occurred_at"`
	Data          interface{} `json:"data"`
}

// Envelope embrulha um payload com os metadados de correlação do contexto.
func Envelope(ctx context.Context, data interface{}) EventEnvelope {
	var idb [16]byte
	_, _ = rand.Read(idb[:])
	ev := EventEnvelope{
		Schema:     EnvelopeSchema,
		EventID:    "evt-" + hex.EncodeToString(idb[:]),
		OccurredAt: time.Now().UTC(),
		Data:       data,
	}
	if sc := trace.SpanContextFromContext(ctx); sc.HasTraceID() {
		ev.CorrelationID = sc.TraceID().String()
	}
	if tid, err := sdk_http.TenantIDFromContext(ctx); err == nil {
		ev.TenantID = tid.String()
	}
	return ev
}

// Caused marca a cadeia de causalidade (evento gerado ao consumir outro).
func (e EventEnvelope) Caused(byEventID string) EventEnvelope {
	e.CausationID = byEventID
	return e
}
