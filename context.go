package hulk

import (
	"context"

	"github.com/vertikon/sdk-hulk.vertikon.com.br/ai"
	"github.com/vertikon/sdk-hulk.vertikon.com.br/events"
	"github.com/vertikon/sdk-hulk.vertikon.com.br/http"
	"github.com/vertikon/sdk-hulk.vertikon.com.br/secrets"
	"github.com/vertikon/sdk-hulk.vertikon.com.br/state"
	"github.com/vertikon/sdk-hulk.vertikon.com.br/telemetry"
	"go.uber.org/zap"
)

// Context é a interface principal passada para os módulos.
// Ela agrupa Logger, EventBus, Store e AI.
type Context interface {
	context.Context

	// Log retorna o Logger estruturado (Zap) já com TraceIDs injetados.
	Log() *zap.Logger

	// EventBus acessa a camada de mensageria (NATS JetStream).
	EventBus() events.Bus

	// Store acessa a camada de persistência de dados e cache.
	Store() state.Store

	// AI acessa o cérebro cognitivo (LLM/Vision/Embeddings).
	AI() ai.Client

	// HTTP acessa o roteador web.
	HTTP() http.Router

	// Secrets acessa o gerenciador de segredos.
	Secrets() secrets.Store
}

// implementation é uma estrutura privada que implementa a interface Context.
// Ela será instanciada pelo Core do HULK e passada para os módulos.
type hulkContext struct {
	context.Context
	logger   *zap.Logger
	bus      events.Bus
	store    state.Store
	aiClient ai.Client
	router   http.Router
	secrets  secrets.Store
}

func NewContext(ctx context.Context, logger *zap.Logger, bus events.Bus, store state.Store, aiClient ai.Client, router http.Router, secretStore secrets.Store) Context {
	return &hulkContext{
		Context:  ctx,
		logger:   logger,
		bus:      bus,
		store:    store,
		aiClient: aiClient,
		router:   router,
		secrets:  secretStore,
	}
}

func (c *hulkContext) Log() *zap.Logger {
	// [BLOCO-P] Enriquecer logger com TraceID e SpanID se disponíveis
	return telemetry.LoggerWithTrace(c.Context, c.logger)
}

func (c *hulkContext) EventBus() events.Bus {
	return c.bus
}

func (c *hulkContext) Store() state.Store {
	return c.store
}

func (c *hulkContext) AI() ai.Client {
	return c.aiClient
}

func (c *hulkContext) HTTP() http.Router {
	return c.router
}

func (c *hulkContext) Secrets() secrets.Store {
	return c.secrets
}
