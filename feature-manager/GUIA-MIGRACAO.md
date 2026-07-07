# 🔄 Guia de Migração - Aplicar Feature Manager nos Módulos

**Status:** ✅ Opcional (Modo Stub)  
**Recomendação:** Aplicar gradualmente para preparar arquitetura  
**Data:** 2025-11-26

---

## 📋 Resumo

**NÃO é obrigatório agora** porque estamos em **modo STUB** (tudo liberado). Porém, é **recomendado** aplicar o Feature Manager nos módulos existentes para:

1. ✅ Preparar a arquitetura para monetização futura
2. ✅ Garantir que novos módulos já sigam o padrão
3. ✅ Testar o fluxo completo (mesmo que permita tudo)

---

## 🎯 Módulos que Precisam de Migração

### ✅ Módulos com Rotas HTTP (Prioridade Alta)

| Módulo | Código do Módulo | Status | Prioridade |
|--------|------------------|--------|------------|
| **financial-core** | `mod.finance.core` | ⚠️ Pendente | Alta |
| **unified-commerce-api** | `mod.sales.unified` | ⚠️ Pendente | Alta |
| **checkout-engine** | `mod.checkout.engine` | ⚠️ Pendente | Alta |
| **wms-core** | `mod.logistics.wms` | ⚠️ Pendente | Alta |
| **pim-product-master** | `mod.mdm.pim` | ⚠️ Pendente | Média |
| **core-inventory** | `mod.inventory.core` | ⚠️ Pendente | Média |
| **procurement-core** | `mod.procurement.core` | ⚠️ Pendente | Média |
| **iam-sso** | `mod.platform.iam` | ✅ Não precisa | - (Público) |

---

## 🔧 Passo a Passo da Migração

### Template de Migração

```go
// ANTES (sem Feature Manager)
func (m *Module) Start(ctx hulk.Context) error {
    // 1. Auth Middleware
    authMiddleware := sdk_http.NewAuthMiddleware(sdk_http.AuthConfig{
        JWTSecret: "change_me_in_production",
    })
    
    // 2. Grupo de rotas
    api := ctx.HTTP().Group("/api/v1/module")
    
    // 3. Aplicar Auth
    if echoGroup, ok := api.(*sdk_http.EchoGroup); ok {
        echoGroup.UseEchoMiddleware(authMiddleware)
    }
    
    // 4. Registrar rotas
    handler.RegisterRoutes(api, ctx)
    return nil
}
```

```go
// DEPOIS (com Feature Manager)
func (m *Module) Start(ctx hulk.Context) error {
    // 1. Feature Manager
    fm := featuremanager.NewService()
    
    // 2. Auth Middleware
    authMiddleware := sdk_http.NewAuthMiddleware(sdk_http.AuthConfig{
        JWTSecret: "change_me_in_production",
    })
    
    // 3. Grupo de rotas
    api := ctx.HTTP().Group("/api/v1/module")
    
    // 4. Aplicar middlewares (ORDEM IMPORTA!)
    if echoGroup, ok := api.(*sdk_http.EchoGroup); ok {
        echoGroup.UseEchoMiddleware(authMiddleware) // Primeiro: Auth
        echoGroup.UseEchoMiddleware(featuremanager.RequireModule(fm, "mod.module.code")) // Depois: Permissão
    }
    
    // 5. Registrar rotas
    handler.RegisterRoutes(api, ctx)
    return nil
}
```

---

## 📝 Exemplos Práticos por Módulo

### 1. Financial Core

**Arquivo:** `internal/modules/finance/financial-core/module.go`

```go
import (
    featuremanager "github.com/vertikon/endurance/pkg/sdk-hulk/feature-manager"
    // ... outros imports
)

func (m *Module) Start(ctx hulk.Context) error {
    ctx.Log().Info("Iniciando serviços Financeiros...", zap.String("module", "financial-core"))

    m.ctx = ctx

    // 1. Feature Manager
    fm := featuremanager.NewService()

    // 2. Auth Middleware
    jwtSecret := "change_me_in_production_please"
    authMiddleware := sdk_http.NewAuthMiddleware(sdk_http.AuthConfig{JWTSecret: jwtSecret})

    // 3. Grupo de API
    api := ctx.HTTP().Group("/api/v1/finance")

    // 4. Aplicar middlewares
    if echoGroup, ok := api.(*sdk_http.EchoGroup); ok {
        echoGroup.UseEchoMiddleware(authMiddleware)
        echoGroup.UseEchoMiddleware(featuremanager.RequireModule(fm, "mod.finance.core"))
    }

    // 5. Registrar rotas
    handler := NewHTTPHandler(m.service)
    handler.RegisterRoutes(api, ctx)

    // 6. Consumers NATS
    if err := m.setupConsumers(ctx); err != nil {
        return err
    }

    return nil
}
```

### 2. Unified Commerce API

**Arquivo:** `internal/modules/sales/unified-commerce-api/module.go`

```go
import (
    featuremanager "github.com/vertikon/endurance/pkg/sdk-hulk/feature-manager"
    // ... outros imports
)

func (m *Module) Start(ctx hulk.Context) error {
    ctx.Log().Info("Iniciando serviços de Vendas...", zap.String("module", "unified-commerce-api"))

    // 1. Feature Manager
    fm := featuremanager.NewService()

    // 2. Auth Middleware
    jwtSecret := "change_me_in_production_please"
    authMiddleware := sdk_http.NewAuthMiddleware(sdk_http.AuthConfig{JWTSecret: jwtSecret})

    // 3. Grupo de API
    api := ctx.HTTP().Group("/api/v1/sales")

    // 4. Aplicar middlewares
    if echoGroup, ok := api.(*sdk_http.EchoGroup); ok {
        echoGroup.UseEchoMiddleware(authMiddleware)
        echoGroup.UseEchoMiddleware(featuremanager.RequireModule(fm, "mod.sales.unified"))
    }

    // 5. Registrar rotas
    handler := NewHTTPHandler(m.service)
    handler.RegisterRoutes(api, ctx)

    return nil
}
```

### 3. Checkout Engine

**Arquivo:** `internal/modules/checkout/checkout-engine/module.go`

```go
import (
    featuremanager "github.com/vertikon/endurance/pkg/sdk-hulk/feature-manager"
    // ... outros imports
)

func (m *Module) Start(ctx hulk.Context) error {
    ctx.Log().Info("Iniciando workers...", zap.String("module", "checkout-engine"))

    // 1. Feature Manager
    fm := featuremanager.NewService()

    // 2. Auth Middleware (se necessário)
    jwtSecret := "change_me_in_production_please"
    authMiddleware := sdk_http.NewAuthMiddleware(sdk_http.AuthConfig{JWTSecret: jwtSecret})

    // 3. Registrar rotas HTTP
    handler := NewHandler(m.service)
    
    // 4. Criar grupo protegido
    checkoutGroup := ctx.HTTP().Group("/api/v1/checkout")
    
    // 5. Aplicar middlewares
    if echoGroup, ok := checkoutGroup.(*sdk_http.EchoGroup); ok {
        echoGroup.UseEchoMiddleware(authMiddleware)
        echoGroup.UseEchoMiddleware(featuremanager.RequireModule(fm, "mod.checkout.engine"))
    }
    
    // 6. Registrar rotas (ajustar handler.RegisterRoutes para aceitar grupo)
    handler.RegisterRoutesWithGroup(checkoutGroup, ctx)

    // 7. Consumers NATS
    if err := m.setupConsumers(ctx); err != nil {
        return err
    }

    return nil
}
```

### 4. WMS Core

**Arquivo:** `internal/modules/logistics/wms-core/module.go`

```go
import (
    featuremanager "github.com/vertikon/endurance/pkg/sdk-hulk/feature-manager"
    // ... outros imports
)

func (m *Module) Start(ctx hulk.Context) error {
    ctx.Log().Info("Iniciando serviços de Logística...", zap.String("module", "wms-core"))

    m.ctx = ctx

    // 1. Feature Manager
    fm := featuremanager.NewService()

    // 2. Auth Middleware
    jwtSecret := "change_me_in_production_please"
    authMiddleware := sdk_http.NewAuthMiddleware(sdk_http.AuthConfig{JWTSecret: jwtSecret})

    // 3. Grupo de API
    api := ctx.HTTP().Group("/api/v1/logistics")

    // 4. Aplicar middlewares
    if echoGroup, ok := api.(*sdk_http.EchoGroup); ok {
        echoGroup.UseEchoMiddleware(authMiddleware)
        echoGroup.UseEchoMiddleware(featuremanager.RequireModule(fm, "mod.logistics.wms"))
    }

    // 5. Registrar rotas
    handler := NewHTTPHandler(m.service)
    handler.RegisterRoutes(api, ctx)

    // 6. Consumers NATS
    if err := m.setupConsumers(ctx); err != nil {
        return err
    }

    return nil
}
```

### 5. PIM Product Master

**Arquivo:** `internal/modules/mdm/pim-product-master/module.go`

```go
import (
    featuremanager "github.com/vertikon/endurance/pkg/sdk-hulk/feature-manager"
    // ... outros imports
)

func (m *Module) Start(ctx hulk.Context) error {
    ctx.Log().Info("Iniciando workers...", zap.String("module", "pim-product-master"))

    // 1. Feature Manager
    fm := featuremanager.NewService()

    // 2. Auth Middleware
    jwtSecret := "change_me_in_production_please"
    authMiddleware := sdk_http.NewAuthMiddleware(sdk_http.AuthConfig{JWTSecret: jwtSecret})

    // 3. Grupo de API
    api := ctx.HTTP().Group("/api/v1/mdm")

    // 4. Aplicar middlewares
    if echoGroup, ok := api.(*sdk_http.EchoGroup); ok {
        echoGroup.UseEchoMiddleware(authMiddleware)
        echoGroup.UseEchoMiddleware(featuremanager.RequireModule(fm, "mod.mdm.pim"))
    }

    // 5. Registrar rotas
    handler := NewHTTPHandler(m.service)
    handler.RegisterRoutes(api, ctx)

    return nil
}
```

### 6. Core Inventory

**Arquivo:** `internal/modules/inventory/core-inventory/module.go`

```go
import (
    featuremanager "github.com/vertikon/endurance/pkg/sdk-hulk/feature-manager"
    // ... outros imports
)

func (m *Module) Start(ctx hulk.Context) error {
    ctx.Log().Info("Iniciando workers...", zap.String("module", "core-inventory"))

    m.ctx = ctx

    // 1. Feature Manager
    fm := featuremanager.NewService()

    // 2. Auth Middleware
    jwtSecret := "change_me_in_production_please"
    authMiddleware := sdk_http.NewAuthMiddleware(sdk_http.AuthConfig{JWTSecret: jwtSecret})

    // 3. Grupo de API
    api := ctx.HTTP().Group("/api/v1/inventory")

    // 4. Aplicar middlewares
    if echoGroup, ok := api.(*sdk_http.EchoGroup); ok {
        echoGroup.UseEchoMiddleware(authMiddleware)
        echoGroup.UseEchoMiddleware(featuremanager.RequireModule(fm, "mod.inventory.core"))
    }

    // 5. Registrar rotas
    handler := NewHTTPHandler(m.service)
    handler.RegisterRoutes(api, ctx)

    // 6. Consumers NATS
    if err := m.setupConsumers(ctx); err != nil {
        return err
    }

    return nil
}
```

### 7. Procurement Core

**Arquivo:** `internal/modules/procurement/procurement-core/module.go`

```go
import (
    featuremanager "github.com/vertikon/endurance/pkg/sdk-hulk/feature-manager"
    // ... outros imports
)

func (m *Module) Start(ctx hulk.Context) error {
    ctx.Log().Info("Iniciando módulo Procurement Core...", zap.String("module", "procurement-core"))

    // 1. Feature Manager
    fm := featuremanager.NewService()

    // 2. Auth Middleware
    jwtSecret := "change_me_in_production_please"
    authMiddleware := sdk_http.NewAuthMiddleware(sdk_http.AuthConfig{JWTSecret: jwtSecret})

    // 3. Grupo de API
    procurementGroup := ctx.HTTP().Group("/api/v1/procurement")

    // 4. Aplicar middlewares
    if echoGroup, ok := procurementGroup.(*sdk_http.EchoGroup); ok {
        echoGroup.UseEchoMiddleware(authMiddleware)
        echoGroup.UseEchoMiddleware(featuremanager.RequireModule(fm, "mod.procurement.core"))
    }

    // 5. Registrar rotas
    handler := handler.NewHTTPHandler(m.service)
    handler.RegisterRoutes(procurementGroup, ctx)

    // 6. Consumers NATS
    if err := m.setupConsumers(ctx); err != nil {
        return err
    }

    return nil
}
```

---

## ⚠️ Módulos que NÃO Precisam

### iam-sso

**Motivo:** Rotas públicas (login, registro) não devem ter Feature Manager.

```go
// ✅ CORRETO: Sem Feature Manager
func (m *Module) Start(ctx hulk.Context) error {
    apiGroup := ctx.HTTP().Group("/api/v1/platform")
    handler := NewHandler(m.service)
    handler.RegisterRoutes(apiGroup, ctx) // Rotas públicas
    return nil
}
```

---

## 📊 Checklist de Migração

### Por Módulo

- [ ] **financial-core** - `mod.finance.core`
- [ ] **unified-commerce-api** - `mod.sales.unified`
- [ ] **checkout-engine** - `mod.checkout.engine`
- [ ] **wms-core** - `mod.logistics.wms`
- [ ] **pim-product-master** - `mod.mdm.pim`
- [ ] **core-inventory** - `mod.inventory.core`
- [ ] **procurement-core** - `mod.procurement.core`

### Verificações

- [ ] Import do `feature-manager` adicionado
- [ ] `featuremanager.NewService()` chamado
- [ ] `RequireModule()` aplicado após `authMiddleware`
- [ ] Código do módulo correto (ex: `mod.finance.core`)
- [ ] Testes passando
- [ ] Compilação sem erros

---

## 🧪 Testando a Migração

### Teste Manual

```bash
# 1. Compilar
go build ./internal/modules/finance/financial-core/...

# 2. Executar testes
go test ./internal/modules/finance/financial-core/... -v

# 3. Verificar que rotas ainda funcionam (stub mode permite tudo)
curl -H "Authorization: Bearer <token>" http://localhost:8080/api/v1/finance/receivables
```

### Teste Automatizado

```go
func TestModule_WithFeatureManager(t *testing.T) {
    // Setup
    ctx := setupTestContext()
    module := financial_core.New()
    
    // Init
    err := module.Init(ctx)
    assert.NoError(t, err)
    
    // Start (deve aplicar Feature Manager)
    err = module.Start(ctx)
    assert.NoError(t, err)
    
    // Verificar que rotas estão protegidas
    // (mesmo que stub mode permita tudo)
}
```

---

## 🎯 Estratégia de Migração Recomendada

### Fase 1: Módulos Críticos (Alta Prioridade)

1. ✅ **financial-core** - Core financeiro
2. ✅ **unified-commerce-api** - API de vendas
3. ✅ **checkout-engine** - Motor de checkout

### Fase 2: Módulos Operacionais (Média Prioridade)

4. ✅ **wms-core** - Logística
5. ✅ **procurement-core** - Compras

### Fase 3: Módulos de Suporte (Baixa Prioridade)

6. ✅ **pim-product-master** - Master de produtos
7. ✅ **core-inventory** - Estoque

---

## ⚡ Impacto da Migração

### Modo Stub (Atual)

- ✅ **Zero impacto**: Tudo continua funcionando normalmente
- ✅ **Zero bloqueio**: Desenvolvimento não é afetado
- ✅ **Preparação**: Arquitetura pronta para produção

### Modo Produção (Futuro)

- 🔄 **Ativação automática**: Basta mudar `HasAccess()` no service
- 🔄 **Controle total**: Permissões baseadas em planos
- 🔄 **Sem alterações**: Módulos já estão preparados

---

## 📝 Notas Importantes

1. **Ordem dos Middlewares:** Sempre aplicar `authMiddleware` antes de `RequireModule`
2. **Códigos de Módulos:** Usar padrão `mod.{categoria}.{nome}` (ex: `mod.finance.core`)
3. **Rotas Públicas:** Não aplicar Feature Manager em rotas públicas (ex: login, registro)
4. **Testes:** Verificar que testes continuam passando após migração

---

**Última Atualização:** 2025-11-26  
**Status:** ✅ Guia completo criado

