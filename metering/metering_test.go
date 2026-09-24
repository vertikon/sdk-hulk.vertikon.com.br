package metering

import (
	"encoding/json"
	"errors"
	"testing"
)

type fakeConn struct {
	subject string
	data    []byte
}

func (f *fakeConn) Publish(s string, d []byte) error { f.subject, f.data = s, d; return nil }

func TestRecord(t *testing.T) {
	c := &fakeConn{}
	p := New(c, "radar", "mcp-radar-service")
	if err := p.Record(UsageEvent{TenantID: "t1", Operation: "consulta", Quantity: Quantity{Units: 1}}); err != nil {
		t.Fatal(err)
	}
	if c.subject != "vtk.metering.usage.v1.radar.mcp-radar-service" {
		t.Fatalf("subject: %q", c.subject)
	}
	var e UsageEvent
	_ = json.Unmarshal(c.data, &e)
	if e.Schema != SchemaV1 || e.Vertical != "radar" || e.EventID == "" || e.OccurredAt.IsZero() {
		t.Fatalf("evento incompleto: %+v", e)
	}
	// EventID do chamador é preservado (idempotência no token)
	_ = p.Record(UsageEvent{EventID: "radar-consulta-42", TenantID: "t1", Operation: "consulta", Quantity: Quantity{Units: 1}})
	_ = json.Unmarshal(c.data, &e)
	if e.EventID != "radar-consulta-42" {
		t.Fatalf("event_id do chamador sobrescrito: %q", e.EventID)
	}
}

func TestRecusa(t *testing.T) {
	p := New(&fakeConn{}, "radar", "svc")
	for nome, tc := range map[string]struct {
		e    UsageEvent
		quer error
	}{
		"sem tenant":      {UsageEvent{Operation: "x", Quantity: Quantity{Units: 1}}, ErrMissingTenant},
		"sem operação":    {UsageEvent{TenantID: "t", Quantity: Quantity{Units: 1}}, ErrMissingMeter},
		"quantidade 0":    {UsageEvent{TenantID: "t", Operation: "x"}, ErrEmptyQuantity},
		"negativa":        {UsageEvent{TenantID: "t", Operation: "x", Quantity: Quantity{Units: -1}}, ErrNegativeAmounts},
		"PII no metadata": {UsageEvent{TenantID: "t", Operation: "x", Quantity: Quantity{Units: 1}, Metadata: map[string]string{"cpf": "1"}}, ErrPIIMetadataKey},
	} {
		if err := p.Record(tc.e); !errors.Is(err, tc.quer) {
			t.Errorf("%s: esperava %v, veio %v", nome, tc.quer, err)
		}
	}
}
