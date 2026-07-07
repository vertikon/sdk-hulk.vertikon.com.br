# DEPENDÊNCIAS — SDK-HULK / MCP-HULK

**Módulo:** `github.com/vertikon/sdk-hulk`  
**Go:** 1.22  
**Data:** 2025-11-24  
**Fonte de verdade:** `go.mod` + `GOWORK=off go list -m all`

---

## 1. Visão geral
- **Dependências diretas:** 3
- **Dependências transitivas resolvidas:** 13
- **Foco:** manter superfície mínima, apenas contratos (drivers reais vivem no `hulk-core`).

## 2. Dependências diretas
| Pacote | Versão | Categoria | Uso no SDK | Link |
| --- | --- | --- | --- | --- |
| `go.uber.org/zap` | v1.26.0 | Observabilidade | Logger injetado via `Context` e usado em `context.go`, `telemetry/logger.go`, exemplos e `cmd/main.go`. | https://github.com/uber-go/zap |
| `go.opentelemetry.io/otel/trace` | v1.24.0 | Observabilidade/Tracing | Extração de `TraceID` em `telemetry/logger.go` para correlacionar logs. | https://go.opentelemetry.io |
| `github.com/stretchr/testify` | v1.8.4 | Testes | Assertions/mocks em `module_test.go`, `ai/ai_test.go`, `events/events_test.go`, `state/state_test.go`. | https://github.com/stretchr/testify |

## 3. Dependências transitivas relevantes
| Pacote | Versão | Origem | Uso/Observação | Link |
| --- | --- | --- | --- | --- |
| `go.uber.org/multierr` | v1.11.0 | `zap` | Aggrega múltiplos erros dentro do logger. | https://github.com/uber-go/multierr |
| `go.uber.org/goleak` | v1.2.0 | `zap` (dev/test) | Detecta goroutines vazando nos testes de logging. | https://github.com/uber-go/goleak |
| `go.opentelemetry.io/otel` | v1.24.0 | `otel/trace` | Tipos base do ecossistema OpenTelemetry. | https://go.opentelemetry.io |
| `go.opentelemetry.io/otel/metric` | v1.24.0 | `otel/trace` | Módulo irmão incluído pelo metapacote; não é consumido diretamente. | https://go.opentelemetry.io |
| `github.com/google/go-cmp` | v0.6.0 | `testify` | Comparações profundas usadas em assertions. | https://github.com/google/go-cmp |
| `github.com/pmezard/go-difflib` | v1.0.0 | `testify` | Geração de diffs human-friendly em falhas. | https://github.com/pmezard/go-difflib |
| `github.com/davecgh/go-spew` | v1.1.1 | `testify` | Dumps estruturados para debugging em testes. | https://github.com/davecgh/go-spew |
| `github.com/stretchr/objx` | v0.5.0 | `testify` | Helpers para manipulação dinâmica em asserts. | https://github.com/stretchr/objx |
| `github.com/go-logr/logr` | v1.4.1 | `otel` | Interface comum de logging requerida pelos exporters do OpenTelemetry. | https://github.com/go-logr/logr |
| `github.com/go-logr/stdr` | v1.2.2 | `otel` | Ponte `logr` → `log/std`. | https://github.com/go-logr/stdr |
| `github.com/kr/text` | v0.2.0 | `go-cmp` | Formatação de diffs de texto. | https://github.com/kr/text |
| `gopkg.in/yaml.v3` | v3.0.1 | `testify` | Serialização YAML em mensagens de erro. | https://gopkg.in/yaml.v3 |
| `gopkg.in/check.v1` | pseudo v0.0.0-20161208… | `testify` | Compat layer herdada dos asserts legados. | https://gopkg.in/check.v1 |

> Observação: executar `go list -m all` sem `GOWORK=off` coleta centenas de módulos externos devido ao `go.work` corporativo do ambiente. Sempre exporte `GOWORK=off` ao auditar apenas o SDK.

## 4. Validação dos dados
Comandos utilizados:
```bash
# listar dependências isolado do go.work global
GOWORK=off go list -m all

# baixar versões travadas
GOWORK=off go mod download

# checar grafos e vulnerabilidades
GOWORK=off go mod verify
GOWORK=off govulncheck ./...
```

## 5. Recomendações
1. **Congelar versões** — mantenha `zap`, `otel/trace` e `testify` fixos até validar major releases. Atualizações patch/minor podem ser aplicadas após `go test ./...` + `govulncheck`.
2. **Monitorar CVEs** — habilitar alertas para `zap` e `otel`, que são peças críticas do contexto. 
3. **Revisar relatórios antigos** — `DEPENDENCIAS-SDK-HULK.md` na raiz lista bibliotecas que não constam no `go.mod`. Atualize-o ou substitua-o por este relatório para evitar divergências.

## 6. Referências cruzadas
- `context.go` e `telemetry/logger.go` → justificam o uso de `zap` e `otel`.
- `module_test.go`, `ai/ai_test.go`, `events/events_test.go`, `state/state_test.go` → justificam `testify` e suas transitividades.

*(Relatório gerado automaticamente com base no estado do repositório em 2025-11-24.)*
