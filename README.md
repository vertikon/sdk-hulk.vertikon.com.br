# 🛠️ SDK-HULK

**SDK Oficial para Desenvolvimento de Módulos Vertikon**

O `sdk-hulk` é a "cola" que permite escrever os 54 módulos do sistema como se fossem serviços independentes, mas rodando acoplados à infraestrutura robusta do HULK. Isso transforma a arquitetura em um **Monolito Modular** - você programa modularmente, mas deploya unificado.

## 📋 Índice

- [Visão Geral](#visão-geral)
- [Estrutura](#estrutura)
- [Quick Start](#quick-start)
- [Interfaces Principais](#interfaces-principais)
- [Exemplos](#exemplos)
- [Arquitetura](#arquitetura)

## 🎯 Visão Geral

O SDK-HULK abstrai a complexidade do Core (NATS, Bancos, AI, Logs) para que o desenvolvedor do Bloco foque apenas na **Regra de Negócio**. 

### Família HULK

| Repo | Papel |
|---|---|
| **mcp-hulk** (`B:\mcp-hulk.vertikon.com.br`) | Base MCP do ecossistema: servidor MCP (stdio) + engine de geração de projetos (`pkg/hulkgen`) — ver `DEVKIT.md` de lá |
| **sdk-hulk** (este repo) | SDK + runtime do Monolito Modular: interfaces (`Module`, `Context`) e `App` que hospeda os módulos |

Gate de qualidade compartilhado: `go build && go vet && staticcheck (zero) && go test` — CI em `.github/workflows/ci.yml` nos dois repos.

### Benefícios

1. **Inversão de Dependência (DIP)**: Módulos não dependem de drivers específicos, apenas de interfaces
2. **Padronização**: Todos os 54 serviços seguem o mesmo ciclo de vida
3. **Contexto Enriquecido**: Um único `Context` carrega todas as ferramentas necessárias
4. **Zero Configuração**: NATS, DBs e AI já estão configurados pelo Core

## 📁 Estrutura

```
pkg/sdk-hulk/
│
├── 📄 module.go             # Interface que todo Bloco deve implementar
├── 📄 context.go            # Contexto enriquecido (HulkContext)
│
├── 📁 ai/                   # Acesso fácil à Inteligência
│   ├── 📄 client.go         # Chat, Vision, Embeddings
│   ├── 📄 llm.go            # Operações de LLM
│   └── 📄 memory.go          # Memória episódica/semântica
│
├── 📁 events/               # Mensageria (NATS JetStream abstraído)
│   ├── 📄 bus.go            # Interface Bus
│   ├── 📄 publisher.go      # Helpers para publicação
│   └── 📄 subscriber.go     # Helpers para inscrição
│
├── 📁 state/                # Persistência (Postgres/Mongo/Redis)
│   ├── 📄 store.go          # Interface Store
│   ├── 📄 repository.go     # CRUD genérico
│   └── 📄 cache.go          # Cache L1/L2
│
└── 📁 telemetry/            # Observabilidade automática
    └── 📄 logger.go         # Logs estruturados com TraceID
```

## 🚀 Quick Start

### 1. Implementar a Interface Module

```go
package inventory

import (
    "context"
    "github.com/vertikon/sdk-hulk"
)

type InventoryModule struct {
    // Suas dependências internas
}

func New() hulk.Module {
    return &InventoryModule{}
}

func (m *InventoryModule) Config() hulk.ModuleConfig {
    return hulk.ModuleConfig{
        ID:      "bloco-1-inventory",
        Name:    "Core Inventory & Fulfillment",
        Version: "v1.0.0",
    }
}

func (m *InventoryModule) Init(ctx hulk.Context) error {
    // Preparação: criar tabelas, carregar templates
    ctx.Log().Info("Inicializando módulo de estoque...")
    return nil
}

func (m *InventoryModule) Start(ctx hulk.Context) error {
    // Execução: iniciar consumers, workers
    return ctx.EventBus().Subscribe("sales.order.created", func(msg events.Message) error {
        // Processar evento
        return msg.Ack()
    })
}

func (m *InventoryModule) Stop(ctx context.Context) error {
    // Cleanup: fechar recursos
    return nil
}
```

### 2. Registrar no Core

```go
// cmd/main.go
func main() {
    app := hulk.NewApp()
    
    app.Register(inventory.New())
    app.Register(sales.New())
    // ... registra os 54 módulos
    
    app.Run()
}
```

## 🔌 Interfaces Principais

### Module

Todo módulo deve implementar:

```go
type Module interface {
    Config() ModuleConfig
    Init(ctx Context) error
    Start(ctx Context) error
    Stop(ctx context.Context) error
}
```

### Context

O `HulkContext` fornece acesso a todas as ferramentas:

```go
type Context interface {
    context.Context
    
    Log() *zap.Logger          // Logger estruturado
    EventBus() events.Bus      // Mensageria NATS
    Store() state.Store        // Persistência e Cache
    AI() ai.Client             // Inteligência Artificial
}
```

### Events.Bus

```go
// Publicar evento
ctx.EventBus().Publish("sales.order.created", order)

// Inscrever-se em evento
ctx.EventBus().Subscribe("sales.order.created", func(msg events.Message) error {
    // Processar
    return msg.Ack()
})

// Load balancing
ctx.EventBus().QueueSubscribe("sales.order.created", "inventory-queue", handler)
```

### State.Store

```go
// Executar query
ctx.Store().Exec(ctx, "INSERT INTO ...", args...)

// Buscar registro
row := ctx.Store().QueryRow(ctx, "SELECT ...", args...)
row.Scan(&id, &name)

// Cache
ctx.Store().CacheSet(ctx, "key", value, 3600)
ctx.Store().CacheGet(ctx, "key", &value)
```

### AI.Client

```go
// Chat simples
response, _ := ctx.AI().Chat(ctx, "What is 2+2?")

// Análise de erro
analysis, _ := ctx.AI().AnalyzeError(ctx, "Contexto do erro", err)

// Vision
result, _ := ctx.AI().Vision(ctx, "https://image.jpg", "Descreva esta imagem")

// Embeddings
vector, _ := ctx.AI().Embeddings(ctx, "texto para buscar")
```

## 📚 Exemplos

### Exemplo Completo: Módulo de Estoque

Veja `examples/inventory_module/module.go` para um exemplo completo de módulo que:

- Escuta eventos de vendas
- Reserva estoque
- Usa IA para análise de erros
- Publica eventos de resposta

### Exemplo Simples

Veja `examples/simple_module/module.go` para um exemplo mínimo.

## 🏗️ Arquitetura

### Fluxo de Dados

```
┌─────────────┐
│   Módulo    │
│  (Estoque)  │
└──────┬──────┘
       │
       │ ctx.EventBus().Publish()
       ▼
┌─────────────────┐
│   NATS JetStream│
└──────┬──────────┘
       │
       │ Subscribe
       ▼
┌─────────────┐
│   Módulo    │
│   (Vendas)  │
└─────────────┘
```

### Ciclo de Vida

```
1. HULK inicia
   ↓
2. Carrega configs (NATS, DBs, AI)
   ↓
3. Para cada módulo:
   - Init(ctx)  → Preparação
   - Start(ctx) → Execução
   ↓
4. Módulos rodam (eventos, workers)
   ↓
5. Graceful Shutdown:
   - Stop(ctx)  → Cleanup
```

## 🧪 Testes

```bash
go test ./pkg/sdk-hulk/...
```

## 📝 Notas de Implementação

### Implementação Real vs SDK

- **SDK (este repositório)**: Define apenas **interfaces** e contratos
- **Core (hulk-core)**: Implementa as interfaces usando NATS, Postgres, OpenAI, etc.

Isso permite:
- Trocar infraestrutura sem quebrar módulos
- Testar módulos com mocks
- Desenvolvimento paralelo

### Dependências

O SDK mantém dependências mínimas:
- `go.uber.org/zap` - Logging estruturado
- `context` - Context padrão do Go

Todas as outras dependências (NATS, DB drivers, etc.) ficam no Core.

## 🤝 Contribuindo

Ao criar um novo módulo:

1. Implemente a interface `Module`
2. Use apenas as interfaces do SDK (não importe drivers diretamente)
3. Siga o padrão de nomenclatura: `bloco-{N}-{nome}`
4. Documente dependências no `ModuleConfig`

## 📄 Licença

[Definir licença]

## 🔗 Links

- [Blueprint Original](./blueprint/templates_sdk-hulk-v1.md)
- [Documentação do Core](../hulk-core/README.md)

