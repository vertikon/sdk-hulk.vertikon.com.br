// Package metering publica consumo no token.vertikon (contrato
// vtk.metering.usage.v1; dono: token.vertikon.com.br internal/domain/usage).
// Espelha o contrato de mcp-shared/pkg/events/metering.go num módulo importável
// por require — o SDK do token (github.com/vertikon/token) não é.
//
// Regra do token: todo meter_key (vertical.operation[.model]) precisa estar no
// catálogo (docs/meters-catalog.md) e no price book ANTES do primeiro evento em
// produção; sem preço, o evento vai para a quarentena no fechamento.
// `vertical-kit provision` confere isso a partir de billing.meters do manifest.
package metering

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// SchemaV1 identifica a versão vigente do contrato de evento.
const SchemaV1 = "vtk.metering.usage.v1"

// Subject devolve vtk.metering.usage.v1.<vertical>.<mcp> (stream VTK_METERING).
func Subject(vertical, mcp string) string { return fmt.Sprintf("%s.%s.%s", SchemaV1, vertical, mcp) }

// Quantity: TokensIn/TokensOut para LLM; Units para medidor não-token.
type Quantity struct {
	TokensIn  int64 `json:"tokens_in"`
	TokensOut int64 `json:"tokens_out"`
	Units     int64 `json:"units"`
}

// UsageEvent é o registro imutável de consumo. Nunca carrega valor monetário
// (ADR-002 do token) nem PII (RNF-05).
type UsageEvent struct {
	EventID    string            `json:"event_id"`
	Schema     string            `json:"schema"`
	TenantID   string            `json:"tenant_id"`
	Vertical   string            `json:"vertical"`
	MCP        string            `json:"mcp"`
	SubjectID  string            `json:"subject_id,omitempty"`
	Operation  string            `json:"operation"`
	Provider   string            `json:"provider,omitempty"`
	Model      string            `json:"model,omitempty"`
	Quantity   Quantity          `json:"quantity"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	OccurredAt time.Time         `json:"occurred_at"`
}

// Erros de validação (mesmas invariantes do agregado canônico do token).
var (
	ErrMissingEventID  = errors.New("metering: event_id obrigatório")
	ErrMissingTenant   = errors.New("metering: tenant_id obrigatório")
	ErrMissingMeter    = errors.New("metering: vertical, mcp e operation obrigatórios")
	ErrEmptyQuantity   = errors.New("metering: quantity deve ser positiva em ao menos uma dimensão")
	ErrZeroOccurredAt  = errors.New("metering: occurred_at obrigatório")
	ErrPIIMetadataKey  = errors.New("metering: metadata contém chave PII proibida")
	ErrNegativeAmounts = errors.New("metering: quantidades não podem ser negativas")
)

var piiDenylist = map[string]struct{}{
	"email": {}, "cpf": {}, "cnpj_titular": {}, "phone": {}, "telefone": {},
	"name": {}, "nome": {}, "address": {}, "endereco": {}, "rg": {}, "cnh": {},
}

// Validate aplica as invariantes do contrato — evento inválido não publica.
func (e UsageEvent) Validate() error {
	switch {
	case e.EventID == "":
		return ErrMissingEventID
	case e.TenantID == "":
		return ErrMissingTenant
	case e.Vertical == "" || e.MCP == "" || e.Operation == "":
		return ErrMissingMeter
	case e.OccurredAt.IsZero():
		return ErrZeroOccurredAt
	}
	if e.Quantity.TokensIn < 0 || e.Quantity.TokensOut < 0 || e.Quantity.Units < 0 {
		return ErrNegativeAmounts
	}
	if e.Quantity.TokensIn == 0 && e.Quantity.TokensOut == 0 && e.Quantity.Units == 0 {
		return ErrEmptyQuantity
	}
	for k := range e.Metadata {
		if _, bad := piiDenylist[k]; bad {
			return ErrPIIMetadataKey
		}
	}
	return nil
}

// Conn é o que o Publisher precisa do NATS (*nats.Conn satisfaz).
type Conn interface {
	Publish(subject string, data []byte) error
}

// Publisher emite o consumo de um serviço (vertical + mcp fixos).
type Publisher struct {
	conn     Conn
	vertical string
	mcp      string
}

// New cria o Publisher do serviço. mcp = nome do serviço (ex.: mcp-radar-service).
func New(conn Conn, vertical, mcp string) *Publisher {
	return &Publisher{conn: conn, vertical: vertical, mcp: mcp}
}

// Record completa o evento (schema, vertical, mcp; event_id e occurred_at se
// vazios), valida e publica. Para reprocessamento idempotente, passe o seu
// próprio EventID — o token deduplica por ele.
func (p *Publisher) Record(e UsageEvent) error {
	e.Schema, e.Vertical, e.MCP = SchemaV1, p.vertical, p.mcp
	if e.EventID == "" {
		e.EventID = uuid.NewString()
	}
	if e.OccurredAt.IsZero() {
		e.OccurredAt = time.Now().UTC()
	}
	if err := e.Validate(); err != nil {
		return err
	}
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	return p.conn.Publish(Subject(p.vertical, p.mcp), b)
}
