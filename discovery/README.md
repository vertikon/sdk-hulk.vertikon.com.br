# discovery — adira ao inventário em 3 passos

Qualquer MCP ganha **auto-registro + heartbeat + deregister** e aparece no
**mcp-inventory** (`inventario.vertikon.com.br`) sem cadastro manual.

```go
import "github.com/vertikon/sdk-hulk.vertikon.com.br/discovery"

// 1) configurar (slug bate com o catálogo)
c, err := discovery.Connect(os.Getenv("NATS_URL"), discovery.Config{
    Slug: "meu-mcp", Environment: "prod", Host: "VM1", Version: "1.0.0", SDKVersion: "0.1.0",
    Checks: map[string]discovery.Check{                 // dependências reportadas
        "db":   func(ctx context.Context) bool { return pingDB(ctx) == nil },
        "nats": func(ctx context.Context) bool { return true },
    },
})
// 2) iniciar (register + heartbeat a cada 30s)
c.Start(ctx)
// 3) parar no shutdown gracioso (deregister)
defer c.Stop()
```

- **Contrato canônico (wire):** `pkg/contracts` do mcp-inventory (`vertikon.inventory.v1.*`).
  As structs aqui espelham esse JSON; publicamos por core NATS e a stream
  `INVENTORY_INGEST` do inventário captura.
- **Sem heartbeat → o inventário te marca `down`** em até 5min; um novo heartbeat volta pra `up`.
- Exemplo completo: `examples/discovery-piloto`.
