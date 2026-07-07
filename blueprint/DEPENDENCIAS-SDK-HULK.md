# DEPENDÊNCIAS SDK-HULK

**Data:** 2025-11-24  
**Versão:** 1.0  
**Módulo:** github.com/vertikon/sdk-hulk

---

## 📋 ÍNDICE

1. [Dependências Go](#dependências-go)
2. [Filosofia de Dependências](#filosofia-de-dependências)
3. [Versões e Compatibilidade](#versões-e-compatibilidade)

---

## 📦 DEPENDÊNCIAS GO

### Core Dependencies

#### Observability

**Zap Logger**

- **Pacote:** `go.uber.org/zap`
- **Versão:** v1.26.0
- **Uso:** Logging estruturado (interface Context)
- **Link:** <https://github.com/uber-go/zap>
- **Licença:** MIT

**OpenTelemetry**

- **Pacote:** `go.opentelemetry.io/otel`
- **Versão:** v1.24.0
- **Uso:** Distributed tracing (telemetry/logger.go)
- **Link:** <https://opentelemetry.io>
- **Licença:** Apache 2.0

**OpenTelemetry Trace**

- **Pacote:** `go.opentelemetry.io/otel/trace`
- **Versão:** v1.24.0
- **Uso:** Trace context extraction
- **Licença:** Apache 2.0

**OpenTelemetry Metrics**

- **Pacote:** `go.opentelemetry.io/otel/metric`
- **Versão:** v1.24.0
- **Uso:** Metrics support (opcional)
- **Licença:** Apache 2.0

#### Utilities

**UUID**

- **Pacote:** `github.com/google/uuid`
- **Versão:** v1.5.0
- **Uso:** Geração de IDs únicos (eventos, mensagens)
- **Link:** <https://github.com/google/uuid>
- **Licença:** BSD-3-Clause

**JSON Iterator**

- **Pacote:** `github.com/json-iterator/go`
- **Versão:** v1.1.12
- **Uso:** JSON parsing otimizado (serialização de eventos)
- **Link:** <https://github.com/json-iterator/go>
- **Licença:** MIT

**Dependências JSON Iterator:**

- `github.com/modern-go/concurrent` v0.0.0-20180306012644-bacd9c7ef1dd
- `github.com/modern-go/reflect2` v1.0.2

#### Error Handling

**Uber Multierr**

- **Pacote:** `go.uber.org/multierr`
- **Versão:** v1.11.0
- **Uso:** Error aggregation e handling
- **Link:** <https://github.com/uber-go/multierr>
- **Licença:** MIT

#### Testing

**Testify**

- **Pacote:** `github.com/stretchr/testify`
- **Versão:** v1.8.4
- **Uso:** Testing framework (assertions, mocks)
- **Link:** <https://github.com/stretchr/testify>
- **Licença:** MIT

**Dependências Testify:**

- `github.com/davecgh/go-spew` v1.1.1
- `github.com/pmezard/go-difflib` v1.0.0
- `gopkg.in/yaml.v3` v3.0.1

---

## 🎯 FILOSOFIA DE DEPENDÊNCIAS

### Princípio: Dependências Mínimas

O SDK-HULK segue o princípio de **dependências mínimas**:

1. **Apenas Interfaces**: O SDK define apenas interfaces e contratos
2. **Sem Implementações**: Não inclui drivers de banco, clientes NATS, etc.
3. **Foco em Contratos**: Dependências apenas para tipos de interface necessários

### O que NÃO está incluído (e por quê)

#### ❌ Drivers de Banco de Dados

- **PostgreSQL (pgx)**: Implementação fica no Core
- **MongoDB Driver**: Implementação fica no Core
- **Redis Client**: Implementação fica no Core

**Razão:** O SDK define apenas a interface `state.Store`. A implementação real fica no `hulk-core`.

#### ❌ Clientes de Mensageria

- **NATS Go Client**: Implementação fica no Core
- **NATS Keys/NUID**: Implementação fica no Core

**Razão:** O SDK define apenas a interface `events.Bus`. A implementação real fica no `hulk-core`.

#### ❌ Frameworks Web

- **Echo**: Não necessário no SDK
- **gRPC**: Não necessário no SDK

**Razão:** O SDK não expõe servidores HTTP/gRPC diretamente. Isso fica no Core.

#### ❌ Clientes de IA

- **OpenAI SDK**: Implementação fica no Core
- **Anthropic SDK**: Implementação fica no Core

**Razão:** O SDK define apenas a interface `ai.Client`. A implementação real fica no `hulk-core`.

### O que ESTÁ incluído (e por quê)

#### ✅ Zap Logger

**Razão:** O `Context` interface retorna `*zap.Logger` diretamente. É parte do contrato.

#### ✅ OpenTelemetry

**Razão:** O `telemetry/logger.go` precisa extrair TraceIDs do context. É necessário para observabilidade.

#### ✅ UUID

**Razão:** Útil para geração de IDs de eventos e mensagens. Biblioteca leve e padrão.

#### ✅ JSON Iterator

**Razão:** Usado para serialização eficiente de eventos. Biblioteca leve e performática.

#### ✅ Testify

**Razão:** Essencial para testes unitários. Padrão da comunidade Go.

---

## 📊 VERSÕES E COMPATIBILIDADE

### Go Version

**Mínimo:** Go 1.22  
**Recomendado:** Go 1.22+  
**Testado:** Go 1.22.0

**Nota:** Alinhado com mcp-hulk (Go 1.22+)

### Compatibilidade de OS

| OS | Versão | Status |
|---|---|---|
| Linux | Ubuntu 22.04+ | ✅ Suportado |
| Linux | Debian 12+ | ✅ Suportado |
| Linux | RHEL 9+ | ✅ Suportado |
| macOS | 13+ (Ventura) | ✅ Suportado |
| Windows | 11 | ✅ Suportado |
| Windows | Server 2022 | ✅ Suportado |

### Arquiteturas

| Arquitetura | Status |
|---|---|
| amd64 (x86_64) | ✅ Suportado |
| arm64 (aarch64) | ✅ Suportado |
| arm (32-bit) | ⚠️ Não testado |

---

## 🔄 ATUALIZAÇÃO DE DEPENDÊNCIAS

### Comandos Úteis

```bash
# Atualizar todas as dependências
go get -u ./...

# Atualizar dependência específica
go get -u go.uber.org/zap@latest

# Limpar dependências não usadas
go mod tidy

# Verificar vulnerabilidades
govulncheck ./...

# Verificar dependências desatualizadas
go list -u -m all
```

### Política de Atualização

- **Patch versions:** Atualizar automaticamente
- **Minor versions:** Revisar changelog antes de atualizar
- **Major versions:** Testar extensivamente antes de atualizar

---

## 🔐 SEGURANÇA

### Verificação de Vulnerabilidades

```bash
# Usando govulncheck
govulncheck ./...

# Usando nancy
go list -json -m all | nancy sleuth

# Usando snyk
snyk test
```

### Dependências com Vulnerabilidades Conhecidas

**Status:** ✅ Nenhuma vulnerabilidade crítica conhecida (última verificação: 2025-11-24)

### Recomendações de Segurança

1. ✅ Manter dependências atualizadas
2. ✅ Usar ferramentas de scanning regularmente
3. ✅ Revisar dependências antes de adicionar
4. ✅ Preferir dependências bem mantidas
5. ✅ Monitorar security advisories

---

## 📝 LICENÇAS

### Resumo de Licenças

| Licença | Quantidade | Exemplos |
|---|---|---|
| MIT | ~4 | Zap, Testify, JSON Iterator |
| Apache 2.0 | ~2 | OpenTelemetry |
| BSD-3-Clause | ~1 | UUID |

### Compatibilidade

✅ Todas as licenças são compatíveis com uso comercial  
✅ Nenhuma licença copyleft (GPL) detectada  
✅ Atribuição requerida para algumas dependências

---

## 🔗 ALINHAMENTO COM MCP-HULK

### Dependências Compartilhadas

O SDK-HULK compartilha as seguintes dependências com o MCP-HULK:

1. ✅ `go.uber.org/zap` - Logging estruturado
2. ✅ `go.opentelemetry.io/otel` - Distributed tracing
3. ✅ `github.com/google/uuid` - UUID generation
4. ✅ `github.com/json-iterator/go` - JSON parsing
5. ✅ `go.uber.org/multierr` - Error handling
6. ✅ `github.com/stretchr/testify` - Testing

### Diferenças Intencionais

O SDK-HULK **não inclui** dependências que estão no MCP-HULK mas são de implementação:

- ❌ `github.com/nats-io/nats.go` - Fica no Core
- ❌ `go.mongodb.org/mongo-driver` - Fica no Core
- ❌ `github.com/jackc/pgx/v5` - Fica no Core
- ❌ `github.com/labstack/echo/v4` - Fica no Core
- ❌ `github.com/spf13/cobra` - Fica no Core

**Razão:** O SDK define apenas interfaces. As implementações ficam no `hulk-core`.

---

## 📈 GRÁFICO DE DEPENDÊNCIAS

### Dependências Diretas (6)

1. `go.uber.org/zap` - Logging estruturado
2. `go.opentelemetry.io/otel` - Distributed tracing
3. `github.com/google/uuid` - UUID generation
4. `github.com/json-iterator/go` - JSON parsing
5. `go.uber.org/multierr` - Error handling
6. `github.com/stretchr/testify` - Testing

### Dependências Indiretas

**Total:** ~10 dependências indiretas

Principais categorias:

- Logging e observabilidade
- Serialização JSON
- Testing utilities
- Error handling

---

## 🔗 LINKS ÚTEIS

### Documentação Oficial

- [Go Modules](https://go.dev/ref/mod)
- [Dependency Management](https://go.dev/doc/modules/managing-dependencies)
- [Go Module Reference](https://go.dev/ref/mod)

### Ferramentas

- [pkg.go.dev](https://pkg.go.dev) - Go package discovery
- [goreportcard.com](https://goreportcard.com) - Code quality
- [deps.dev](https://deps.dev) - Dependency insights

---

**Fim do Relatório de Dependências**

*Gerado automaticamente em 2025-11-24*  
*SDK-HULK v1.0*  
*Alinhado com MCP-HULK v1.0*

