# 🛡️ Feature Manager - Sistema de Licenciamento de Módulos

**Versão:** v1.0.0 (Stub Mode)  
**Status:** ✅ Implementado (Modo Desenvolvimento)  
**Data:** 2025-11-26

---

## 📋 Visão Geral

O **Feature Manager** é o sistema central de controle de acesso a módulos da plataforma Vertikon Endurance Fashion. Ele funciona como um "porteiro" que verifica se um tenant (cliente) tem permissão para usar um módulo específico antes de permitir o acesso.

### 🎯 Objetivo

**Blindar a arquitetura** para que, quando você decidir cobrar, seja apenas uma "virada de chave" no banco de dados, **sem precisar tocar em uma linha de código dos módulos**.

---

## 🏗️ Arquitetura

### Modo Atual: Stub (Desenvolvimento)

- ✅ **Tudo liberado**: `HasAccess()` sempre retorna `true`
- ✅ **Zero bloqueio**: Desenvolvimento e testes funcionam normalmente
- ✅ **Estrutura pronta**: Schema SQL criado, middleware implementado

### Modo Futuro: Produção

- 🔄 **Consulta Redis**: Cache de permissões (performance)
- 🔄 **Consulta Banco**: Verificação de planos e addons
- 🔄 **Fail-secure**: Em caso de erro, nega acesso

---

## 📦 Componentes

### 1. Service (`service.go`)

Interface e implementação do Feature Manager:

```go
type FeatureManager interface {
    HasAccess(ctx context.Context, tenantID uuid.UUID, moduleCode string) bool
}
```

**Modo Stub:**
```go
func (s *Service) HasAccess(ctx context.Context, tenantID uuid.UUID, moduleCode string) bool {
    return true // Tudo liberado em desenvolvimento
}
```

### 2. Middleware (`middleware.go`)

Middleware HTTP que verifica permissões antes de permitir acesso:

```go
RequireModule(manager FeatureManager, moduleCode string) echo.MiddlewareFunc
```

**Comportamento:**
- Se `TenantID == uuid.Nil` (desenvolvimento): Permite acesso
- Se `TenantID != uuid.Nil` (produção): Consulta Feature Manager
- Retorna `403 Forbidden` se não tiver permissão

### 3. Schema SQL (`subscription-hub/schema.sql`)

Estrutura de dados para gerenciamento de módulos e planos:

- `system_modules` - Catálogo de módulos
- `subscription_plans` - Planos de assinatura
- `plan_entitlements` - Módulos incluídos em cada plano
- `tenant_subscriptions` - Assinaturas dos tenants
- `tenant_addons` - Módulos adicionais comprados
- `tenant_entitlements` - View consolidada (plano + addons)
- `check_tenant_module_access()` - Função SQL para verificação

---

## 🚀 Como Usar

### Exemplo 1: Proteger Rotas de um Módulo

```go
// No module.go do fiscal-document-issuer
func (m *Module) Start(ctx hulk.Context) error {
    // 1. Inicializa o Feature Manager
    fm := featuremanager.NewService()
    
    // 2. Cria grupo de rotas protegido
    fiscalGroup := ctx.HTTP().Group("/api/v1/fiscal")
    
    // 3. Aplica middlewares (ordem importa!)
    fiscalGroup.Use(authMiddleware) // Primeiro: Autenticação
    fiscalGroup.Use(featuremanager.RequireModule(fm, "mod.fiscal.issuer")) // Depois: Permissão
    
    // 4. Registra rotas
    fiscalGroup.POST("/nfe/emit", m.Handler.EmitNFe)
    fiscalGroup.GET("/nfe/:id", m.Handler.GetNFe)
    
    return nil
}
```

### Exemplo 2: Múltiplos Módulos no Mesmo Grupo

```go
// Grupo de rotas que requer múltiplos módulos
advancedGroup := ctx.HTTP().Group("/api/v1/advanced")
advancedGroup.Use(authMiddleware)
advancedGroup.Use(featuremanager.RequireModule(fm, "mod.fiscal.issuer"))
advancedGroup.Use(featuremanager.RequireModule(fm, "mod.ai.vision"))

// Só passa se tiver AMBOS os módulos
advancedGroup.POST("/analyze-invoice", handler.AnalyzeInvoice)
```

### Exemplo 3: Verificação Programática

```go
// Em um handler, verificar acesso antes de executar lógica
func (h *Handler) ProcessPayment(c echo.Context) error {
    user := sdkhttp.GetUserFromContext(c)
    if user == nil {
        return c.JSON(401, map[string]string{"error": "unauthorized"})
    }
    
    fm := featuremanager.NewService()
    if !fm.HasAccess(c.Request().Context(), user.TenantID, "mod.payments.gateway") {
        return c.JSON(403, map[string]string{
            "error": "module_not_enabled",
            "module": "mod.payments.gateway",
        })
    }
    
    // Lógica do handler...
    return nil
}
```

---

## 📊 Códigos de Módulos (Padrão)

### Convenção de Nomenclatura

```
mod.{categoria}.{nome}
```

### Exemplos

| Código | Módulo | Categoria |
|--------|--------|-----------|
| `mod.platform.iam` | Identity & SSO | PLATFORM |
| `mod.platform.audit` | Audit Log | PLATFORM |
| `mod.inventory.core` | Core Inventory | INVENTORY |
| `mod.checkout.engine` | Checkout Engine | CHECKOUT |
| `mod.checkout.oms` | OMS Core | CHECKOUT |
| `mod.payments.gateway` | Payment Gateway | PAYMENTS |
| `mod.payments.antifraud` | Antifraud Engine | PAYMENTS |
| `mod.finance.core` | Financial Core | FINANCE |
| `mod.fiscal.tax` | Tax Intelligence | FISCAL |
| `mod.fiscal.issuer` | Fiscal Document Issuer | FISCAL |
| `mod.logistics.wms` | WMS Core | LOGISTICS |
| `mod.procurement.core` | Procurement Core | PROCUREMENT |
| `mod.sales.unified` | Unified Commerce API | SALES |
| `mod.mdm.pim` | PIM Product Master | MDM |

---

## 🔄 Migração para Produção

### Passo 1: Atualizar `HasAccess()` no Service

```go
func (s *Service) HasAccess(ctx context.Context, tenantID uuid.UUID, moduleCode string) bool {
    // 1. Verificar cache Redis primeiro
    key := fmt.Sprintf("tenant:%s:entitlements", tenantID.String())
    allowed, err := s.redisClient.SIsMember(ctx, key, moduleCode).Result()
    if err == nil && allowed {
        return true
    }
    
    // 2. Consultar banco de dados
    var hasAccess bool
    query := `
        SELECT check_tenant_module_access($1, $2)
    `
    err = s.store.QueryRow(ctx, query, tenantID, moduleCode).Scan(&hasAccess)
    if err != nil {
        return false // Fail-secure
    }
    
    // 3. Cachear resultado (TTL: 5 minutos)
    if hasAccess {
        s.redisClient.SAdd(ctx, key, moduleCode)
        s.redisClient.Expire(ctx, key, 5*time.Minute)
    }
    
    return hasAccess
}
```

### Passo 2: Popular Banco de Dados

```sql
-- 1. Inserir planos
INSERT INTO subscription_plans (name, code, price, currency) VALUES
    ('Basic', 'plan.basic.v1', 99.90, 'BRL'),
    ('Pro', 'plan.pro.v1', 299.90, 'BRL'),
    ('Enterprise', 'plan.enterprise.v1', 999.90, 'BRL');

-- 2. Associar módulos aos planos
INSERT INTO plan_entitlements (plan_id, module_code)
SELECT sp.id, sm.code
FROM subscription_plans sp
CROSS JOIN system_modules sm
WHERE sp.code = 'plan.basic.v1'
    AND sm.code IN ('mod.platform.iam', 'mod.inventory.core', 'mod.sales.unified');

-- 3. Criar assinatura para um tenant
INSERT INTO tenant_subscriptions (tenant_id, plan_id, status, start_date, auto_renew)
VALUES (
    '550e8400-e29b-41d4-a716-446655440000'::UUID,
    (SELECT id FROM subscription_plans WHERE code = 'plan.basic.v1'),
    'ACTIVE',
    NOW(),
    true
);
```

### Passo 3: Configurar Redis (Opcional)

```go
// No app.go, injetar Redis no Feature Manager
redisClient := pgStore.RedisClient()
fm := featuremanager.NewServiceWithRedis(redisClient)
```

---

## 🧪 Testes

### Teste do Service (Stub Mode)

```go
func TestService_HasAccess_StubMode(t *testing.T) {
    service := featuremanager.NewService()
    ctx := context.Background()
    tenantID := uuid.New()
    
    // Em stub mode, sempre retorna true
    assert.True(t, service.HasAccess(ctx, tenantID, "mod.fiscal.issuer"))
    assert.True(t, service.HasAccess(ctx, tenantID, "mod.any.module"))
}
```

### Teste do Middleware

```go
func TestRequireModule_Middleware(t *testing.T) {
    // Setup
    e := echo.New()
    fm := featuremanager.NewService()
    
    // Rota protegida
    e.GET("/test", func(c echo.Context) error {
        return c.JSON(200, map[string]string{"status": "ok"})
    }, featuremanager.RequireModule(fm, "mod.test"))
    
    // Teste: Sem autenticação (deve retornar 401)
    // Teste: Com autenticação mas sem módulo (deve retornar 403)
    // Teste: Com autenticação e módulo (deve retornar 200)
}
```

---

## 📝 Checklist de Implementação

### ✅ Implementado

- [x] Interface `FeatureManager`
- [x] Service com modo stub
- [x] Middleware `RequireModule`
- [x] Schema SQL completo
- [x] View `tenant_entitlements`
- [x] Função SQL `check_tenant_module_access()`
- [x] TenantID adicionado ao `User` e `HulkClaims`
- [x] Documentação

### 🔄 Pendente (Futuro)

- [ ] Implementar lógica real no `HasAccess()` (Redis + Banco)
- [ ] Adicionar FeatureManager ao Context do HULK
- [ ] Criar testes de integração
- [ ] Implementar cache de permissões no Redis
- [ ] Criar API para gerenciar planos e assinaturas
- [ ] Implementar webhooks para atualização de assinaturas

---

## 🔐 Segurança

### Fail-Secure

- Em caso de erro na verificação, **nega acesso** por padrão
- Logs de tentativas de acesso negado
- Rate limiting para evitar abuso

### Performance

- Cache Redis com TTL de 5 minutos
- Consulta SQL otimizada (índices)
- View materializada para consultas frequentes

---

## 📚 Referências

- [Schema SQL](../internal/modules/checkout/subscription-hub/schema.sql)
- [Service Implementation](./service.go)
- [Middleware Implementation](./middleware.go)

---

**Última Atualização:** 2025-11-26  
**Versão:** v1.0.0 (Stub Mode)

