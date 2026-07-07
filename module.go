package hulk

import (
	"context"
)

// ModuleConfig define a identidade e metadados de um bloco Vertikon.
type ModuleConfig struct {
	ID           string   // Identificador único (ex: "bloco-1-inventory")
	Name         string   // Nome legível (ex: "Gestão de Estoque Core")
	Version      string   // Versão semântica (ex: "v1.0.0")
	Dependencies []string // Lista de IDs de módulos dependentes (opcional, para ordem de boot)
}

// Module é a interface fundamental. Todo serviço (Estoque, Vendas, IA) deve implementar isso.
// Isso transforma um código solto em um "Plugin" gerenciado pelo HULK.
type Module interface {
	// Config retorna a identidade do módulo para registro e logs.
	Config() ModuleConfig

	// Init é a fase de preparação.
	// Use para: Criar tabelas (Migrate), preparar Statements SQL, carregar templates.
	// O HULK chama Init() em todos os módulos antes de chamar Start().
	Init(ctx Context) error

	// Start é a fase de execução.
	// Use para: Iniciar Consumers NATS, Workers em background, Servidores HTTP/gRPC.
	// Deve ser não-bloqueante (usar goroutines para workers longos) ou gerenciar seu próprio ciclo.
	Start(ctx Context) error

	// Stop é a fase de encerramento (Graceful Shutdown).
	// Use para: Fechar canais, parar tickers, desconectar recursos locais.
	Stop(ctx context.Context) error
}
