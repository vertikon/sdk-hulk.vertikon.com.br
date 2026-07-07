package simple_module

import (
	"context"

	"github.com/vertikon/sdk-hulk.vertikon.com.br"
	"github.com/vertikon/sdk-hulk.vertikon.com.br/events"
	"go.uber.org/zap"
)

// SimpleModule é um exemplo mínimo de implementação de módulo.
type SimpleModule struct{}

// New cria uma nova instância do módulo simples.
func New() hulk.Module {
	return &SimpleModule{}
}

func (m *SimpleModule) Config() hulk.ModuleConfig {
	return hulk.ModuleConfig{
		ID:      "simple-module",
		Name:    "Simple Example Module",
		Version: "v1.0.0",
	}
}

func (m *SimpleModule) Init(ctx hulk.Context) error {
	ctx.Log().Info("Módulo simples inicializado")
	return nil
}

func (m *SimpleModule) Start(ctx hulk.Context) error {
	ctx.Log().Info("Módulo simples iniciado")

	// Exemplo: Publicar um evento
	_ = ctx.EventBus().Publish("simple.hello", map[string]string{
		"message": "Hello from SimpleModule!",
	})

	// Exemplo: Inscrever-se em eventos
	_ = ctx.EventBus().Subscribe("simple.echo", func(msg events.Message) error {
		ctx.Log().Info("Evento recebido",
			zap.String("topic", msg.Topic()),
			zap.String("id", msg.ID()),
		)
		return msg.Ack()
	})

	// Exemplo: Usar cache
	_ = ctx.Store().CacheSet(ctx, "simple:key", "value", 3600)

	// Exemplo: Usar IA
	response, _ := ctx.AI().Chat(ctx, "What is 2+2?")
	ctx.Log().Info("Resposta da IA", zap.String("response", response))

	return nil
}

func (m *SimpleModule) Stop(ctx context.Context) error {
	// Cleanup
	return nil
}
