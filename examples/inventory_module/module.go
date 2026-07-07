package inventory_module

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/vertikon/sdk-hulk.vertikon.com.br"
	"github.com/vertikon/sdk-hulk.vertikon.com.br/events"
	"github.com/vertikon/sdk-hulk.vertikon.com.br/state"
	"go.uber.org/zap"
)

// InventoryModule implementa o módulo de gestão de estoque.
type InventoryModule struct {
	ledgerService *LedgerService
}

// New cria uma nova instância do módulo de estoque.
func New() hulk.Module {
	return &InventoryModule{}
}

func (m *InventoryModule) Config() hulk.ModuleConfig {
	return hulk.ModuleConfig{
		ID:           "bloco-1-inventory",
		Name:         "Core Inventory & Fulfillment",
		Version:      "v1.0.0",
		Dependencies: []string{"bloco-15-mdm"}, // Depende do MDM
	}
}

func (m *InventoryModule) Init(ctx hulk.Context) error {
	ctx.Log().Info("Inicializando tabelas de estoque...")
	// O SDK já injeta a conexão do banco configurada no config.yaml global
	m.ledgerService = NewLedgerService(ctx.Store())
	return nil
}

func (m *InventoryModule) Start(ctx hulk.Context) error {
	// Inscrevendo-se em eventos usando o SDK
	// "Quando uma venda for criada (B8), reserve o estoque"
	err := ctx.EventBus().Subscribe("sales.order.created.v1", func(msg events.Message) error {
		// 1. Telemetria automática (Tracing)
		ctx.Log().Info("Processando ordem de venda",
			zap.String("msg_id", msg.ID()),
			zap.String("topic", msg.Topic()),
		)

		// 2. Lógica de Negócio
		var order OrderCreated
		if err := json.Unmarshal(msg.Payload(), &order); err != nil {
			return fmt.Errorf("erro ao deserializar ordem: %w", err)
		}

		err := m.ledgerService.ReserveStock(ctx, order.SKU, order.Qty)

		// 3. IA Nativa (Ex: Análise de anomalia)
		if err != nil {
			// Pede ajuda pro HULK analisar o erro
			analysis, aiErr := ctx.AI().AnalyzeError(ctx, "Erro de reserva incomum", err)
			if aiErr != nil {
				ctx.Log().Warn("Falha ao analisar erro com IA", zap.Error(aiErr))
			} else {
				ctx.Log().Error("Falha na reserva",
					zap.String("ai_analysis", analysis),
					zap.Error(err),
				)
			}
		} else {
			// Publica evento de sucesso
			_ = ctx.EventBus().Publish("inventory.stock.reserved.v1", map[string]interface{}{
				"sku":      order.SKU,
				"quantity": order.Qty,
			})
		}

		return err
	})

	return err
}

func (m *InventoryModule) Stop(ctx context.Context) error {
	// Cleanup de recursos
	return nil
}

// OrderCreated representa uma ordem de venda criada.
type OrderCreated struct {
	SKU string `json:"sku"`
	Qty int    `json:"qty"`
	ID  string `json:"id"`
}

// LedgerService gerencia o ledger de estoque.
type LedgerService struct {
	store state.Store
}

// NewLedgerService cria um novo LedgerService.
func NewLedgerService(store state.Store) *LedgerService {
	return &LedgerService{store: store}
}

// ReserveStock reserva estoque para um SKU.
func (s *LedgerService) ReserveStock(ctx context.Context, sku string, qty int) error {
	// Implementação simplificada - em produção, usar transações
	query := `UPDATE inventory SET reserved = reserved + $1 WHERE sku = $2`
	return s.store.Exec(ctx, query, qty, sku)
}
