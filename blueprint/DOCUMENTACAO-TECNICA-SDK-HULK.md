# DOCUMENTAÇÃO TÉCNICA — SDK-HULK

**Repositório:** `github.com/vertikon/sdk-hulk`  
**Local:** `E:/vertikon/.templates/sdk-hulk`  
**Data:** 2025-11-24

---

## 1. Visão geral
O SDK-HULK é a camada de abstração usada para construir os 54 blocos Vertikon dentro de um monolito modular. Ele entrega contratos Go puros que isolam os módulos de infraestrutura (NATS, bancos, IA, logs). O repositório contém:
- Interfaces e tipos que definem o contrato oficial (`module.go`, `context.go`, subpacotes `ai`, `events`, `state`, `telemetry`).
- Exemplos completos para acelerar o onboarding (`examples/`).
- Um binário exemplo (`cmd/main.go`) que expõe endpoints HTTP e health-check.
- Utilidades de health (`internal/health`) e material de documentação/testes (`docs/`, `coverage/`, `state_coverage/`).

## 2. Arquitetura dos pacotes
### 2.1 Contrato do módulo (`module.go`)
```go
// ModuleConfig e Module definem a unidade plugável do HULK
func (m *MyModule) Config() ModuleConfig
func (m *MyModule) Init(ctx Context) error
func (m *MyModule) Start(ctx Context) error
func (m *MyModule) Stop(ctx context.Context) error
```
- `Stop` usa `context.Context` para permitir integração com sinais e deadlines durante shutdown.
- `ModuleConfig.Dependencies` suporta ordenação e verificação de boot.

### 2.2 Contexto enriquecido (`context.go`)
`Context` embute `context.Context` e expõe `Log()`, `EventBus()`, `Store()` e `AI()`. A implementação privada recebe dependências pelo core via `NewContext(parent, logger, bus, store, ai)` e distribui a todos os módulos.

### 2.3 Pacote `ai/`
- `client.go`: define `Client` com operações síncronas (`Chat`, `ChatWithContext`, `AnalyzeError`, `Vision`, `Embeddings`, `BatchEmbeddings`). Todas recebem `context.Context` para cancelamento/timeouts.
- `llm.go`: contratos de baixo nível (`Completion`, `StreamCompletion`) com `CompletionOptions` padrão.
- `memory.go`: contratos de memória episódica/semântica (`MemoryClient`, `EpisodicMemory`, `SemanticMemory`).
- `ai_test.go`: mocks `MockClient` e testes cobrindo fluxos básicos.

### 2.4 Pacote `events/`
- `bus.go`: Interface `Bus` com `Publish`, `Subscribe`, `QueueSubscribe` e tipos `Message`, `Handler`.
- `publisher.go`/`subscriber.go`: helpers leves que encapsulam o `Bus` para padrões JSON/Queue.
- `events_test.go`: implementa `MockEventBus` e `MockMessage` para teste de módulos sem NATS real.

### 2.5 Pacote `state/`
- `store.go`: interface `Store` cobre `Exec`, `QueryRow`, `Query`, `BeginTx` e APIs de cache (`CacheSet/Get/Delete`). Define também `Rows`, `RowScanner` e `Tx`.
- `cache.go`: cache L1/L2 com helpers (`SetWithDefaultTTL`, `GetOrSet`).
- `repository.go`: CRUD genérico + execução transacional via `ExecuteInTx`.
- `state_test.go`: mocks, testes de concorrência, benchmarks.

### 2.6 Telemetria (`telemetry/logger.go`)
Encapsula `zap.Logger` com awareness de OpenTelemetry. Oferece `WithTraceID`, `WithContext`, `WithModule` e `WithFields` para adicionar metadados consistentes.

### 2.7 Infra de health (`internal/health`)
Define `Service`, `Checker`, `Health` e `Check`. Um `HTTPHandler` de exemplo retorna o estado agregado e pode ser conectado a frameworks HTTP (o `cmd/main.go` usa apenas uma função inline para expor `/health`).

## 3. Fluxo de ciclo de vida
1. O núcleo (`hulk-core`, fora deste repo) inicializa o runtime, cria `zap.Logger`, `events.Bus`, `state.Store` e `ai.Client` reais.
2. Para cada módulo registrado, chama `Init(ctx)` para preparação (migrations, caches).
3. Em seguida, invoca `Start(ctx)` para acionar consumidores/eventos/rotinas.
4. Durante shutdown, passa um `context.Context` para `Stop` para liberar recursos.
5. `Context` é propagado entre módulos para manter traceability (logging, AI, store, event bus).

## 4. Criando um módulo
Referência: `examples/inventory_module/module.go`.

```go
func (m *InventoryModule) Start(ctx hulk.Context) error {
    return ctx.EventBus().Subscribe("sales.order.created.v1", func(msg events.Message) error {
        // logs estruturados
        ctx.Log().Info("Processando", zap.String("msg_id", msg.ID()))
        // payload -> domínio
        var order OrderCreated
        json.Unmarshal(msg.Payload(), &order)
        // persistência
        err := m.ledgerService.ReserveStock(ctx, order.SKU, order.Qty)
        // IA para diagnóstico
        if err != nil {
            analysis, _ := ctx.AI().AnalyzeError(ctx, "Erro de reserva", err)
            ctx.Log().Error("Falha", zap.String("ai_analysis", analysis))
        }
        return err
    })
}
```
Passos recomendados:
1. Criar struct do módulo e dependências internas.
2. Implementar `Config()` retornando IDs padronizados (`bloco-{N}-{dominio}`).
3. Em `Init`, construir serviços a partir de `ctx.Store()` ou caches.
4. Em `Start`, registrar assinaturas ou workers; usar goroutines para loops bloqueantes.
5. Em `Stop`, fechar canais, aguardar goroutines e usar `ctx` apenas para deadlines/sinais.

## 5. Aplicativo exemplo (`cmd/main.go`)
- Inicia `zap.Logger` e mostra banner informativo.
- Instancia `internal/health.Service` e expõe `GET /health` com JSON contendo `status`, `checks` e `version`.
- Expõe `GET /` com metadados do SDK.
- Controla ciclo de vida com canais (`serverErrors`, `shutdown`) e `context.WithTimeout` durante o shutdown.
Esse binário serve como referência para wiring do SDK em processos reais.

## 6. Observabilidade e logs
- `telemetry.Logger.WithContext` extrai TraceID via OpenTelemetry (`trace.SpanFromContext`). Se não existir span, busca `ctx.Value("trace_id")`.
- Recomenda-se encadear `logger := telemetry.NewLogger(base).WithModule(moduleID, moduleName)` antes de injetar no `Context` para garantir consistência de campos.

## 7. Health checks
Para expor health:
```go
service := health.NewService(version)
service.Register(dbChecker)
service.Register(natsChecker)
http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
    status := service.Check(r.Context())
    json.NewEncoder(w).Encode(status)
})
```
Cada `Checker` deve medir tempo, status (healthy/degraded/unhealthy) e mensagem.

## 8. Testes e validação
- `go test ./...` cobre pacotes `ai`, `events`, `state`, `internal/health` e o módulo principal (mock context).
- Testes utilizam `github.com/stretchr/testify` para asserts.
- Há relatórios históricos em `docs/validation/raw/` (lint, staticcheck, govulncheck). Use-os como referência para pipelines de CI.

## 9. Convenções e melhores práticas
1. **Imports**: sempre use `github.com/vertikon/sdk-hulk/...` dentro dos blocos.
2. **Contextos**: propague `ctx context.Context` para qualquer operação longa. Os clientes expostos (`AI`, `Store`, `EventBus`) já aceitam contextos.
3. **Dependências**: mantenha `go.mod` minimalista (zap, otel trace, testify). Drivers concretos vivem no `hulk-core`.
4. **Nomenclatura de tópicos**: padronize eventos como `{dominio}.{entidade}.{acao}.v{n}` (vide `inventory_module`).
5. **Cache**: utilize `state.Cache.GetOrSet` para evitar carimbo manual de TTL.
6. **Docs**: atualize `README.md` e esta documentação ao estender contratos.

## 10. Referências
- Blueprint oficial: `blueprint/templates_sdk-hulk-v1.md`.
- Exemplos: `examples/simple_module`, `examples/inventory_module`.
- Health: `internal/health/health.go`.
- Telemetria: `telemetry/logger.go`.
- Relatórios anteriores: `docs/*`, `coverage/`, `state_coverage/`.
