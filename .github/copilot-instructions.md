<!--
This file provides compact, repo-specific guidance for AI coding agents (Copilot-style).
Keep it short and actionable — examples and concrete file references help the agent act fast.
-->

# Copilot instructions — SDK-HULK

Be concise. Focus on code in `pkg/sdk-hulk` and its examples. This repository is an SDK of interfaces (not the runtime): keep changes limited to the SDK surface unless explicitly asked to update Core implementations elsewhere.

Key facts for immediate productivity
- The SDK exposes the `Module` interface in `module.go` — every module must implement Config(), Init(ctx), Start(ctx), Stop(ctx).
- The `Context` (see `context.go`) carries Log(), EventBus(), Store(), AI(), and HTTP(). Use those instead of importing drivers directly.
- Examples live under `examples/` (notably `examples/inventory_module/module.go` and `examples/simple_module/module.go`) — copy their shape when scaffolding new modules.
- Messaging conventions are in `docs/NATS_SUBJECTS.md` — follow `{module}.{domain}.{action}.{version}` and prefer queue groups for work distribution.

Common developer workflows (concrete commands)
- Run unit tests for the SDK package:
  - `go test ./pkg/sdk-hulk/...`
- Run the example HTTP server (quick local run):
  - `go run ./pkg/sdk-hulk/cmd`
- Check the examples for usage patterns (important for lifecycle and event wiring):
  - `pkg/sdk-hulk/examples/*`

Important conventions & gotchas
- Init() is for preparation (migrations, prepare statements). Start() must not block — start goroutines for long-running workers and return quickly.
- Modules should only depend on SDK interfaces (avoid pulling in NATS, Postgres or AI drivers directly). Core provides implementations.
- Use `ctx.Log()` (Zap logger) for structured logs and prefer TraceIDs injected by the SDK's telemetry helpers.
- Use `ctx.EventBus()` to Publish/Subscribe/QueueSubscribe (see `events/*` and `docs/NATS_SUBJECTS.md`). Always ack messages in handlers.

Files to consult when in doubt
- `module.go` — Module interface and ModuleConfig naming convention (bloco-{N}-{name})
- `context.go` — Context API (Log, EventBus, Store, AI, HTTP)
- `examples/` — canonical module patterns
- `docs/NATS_SUBJECTS.md` — subject naming and message patterns
- `telemetry/logger.go` — how tracing/logging are expected to be used

When making changes
- Prefer small, focused PRs. If you add exports to the SDK, update examples to showcase the new API.
- Add or update unit tests under `pkg/sdk-hulk/*_test.go` and use `go test` to validate behavior.
- Keep public API stable — the SDK is an interface boundary used by many modules.

If you need to modify Core implementations (NATS, DB, AI), ask for permission and point the change at the Core repo (not this SDK package).

Troubleshooting hints
- Look at `docs/validation/raw/` for previous staticcheck/govet results when linting failures appear.
- If a module's Start() blocks tests or local runs, check for missing goroutines or unclosed resources.

Questions to ask a human reviewer when unsure
- Is this change API-level (SDK surface) or an implementation detail? If API-level, we must update all consuming examples.
- Will this change require Core-level wiring (NATS/DB/AI)? If yes, coordinate with the Core owners.

Short samples (copyable)
Use the Module skeleton when scaffolding new modules:
```go
func (m *MyModule) Init(ctx hulk.Context) error { /* migrate or prepare */ }
func (m *MyModule) Start(ctx hulk.Context) error { go m.runWorkers(); return nil }
func (m *MyModule) Stop(ctx context.Context) error { /* cleanup */ }
```

Event handling example (from examples/inventory_module):
```go
err := ctx.EventBus().Subscribe("sales.order.created.v1", func(msg events.Message) error {
    // deserialize, process, ack
    return msg.Ack()
})
```

— End of file —
