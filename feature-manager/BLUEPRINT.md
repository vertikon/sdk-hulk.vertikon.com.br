# 🛡️ Blueprint Técnico v1.0 - Feature Manager (Stub Mode)

**Data:** 2025-11-26  
**Status:** ✅ Implementado  
**Versão:** v1.0.0

---

## 📋 Objetivo

**Blindar a arquitetura** para que, quando você decidir cobrar, seja apenas uma "virada de chave" no banco de dados, **sem precisar tocar em uma linha de código dos módulos**.

---

## 🏗️ Arquitetura Implementada

### 1. Service (`service.go`)

**Interface:**
```go
type FeatureManager interface {
    HasAccess(ctx context.Context, tenantID uuid.UUID, moduleCode string) bool
}
```

**Implementação Stub:**
- ✅ Retorna sempre `true` (modo desenvolvimento)
- ✅ Código comentado mostra lógica futura (Redis + Banco)
- ✅ Zero impacto no desenvolvimento atual

### 2. Middleware (`middleware.go`)

**Funcionalidade:**
- ✅ Verifica autenticação (deve vir antes)
- ✅ Extrai `TenantID` do usuário
- ✅ Se `TenantID == uuid.Nil` → Permite (desenvolvimento)
- ✅ Se `TenantID != uuid.Nil` → Consulta Feature Manager
- ✅ Retorna `403 Forbidden` se não tiver permissão

### 3. Schema SQL (`subscription-hub/schema.sql`)

**Tabelas Criadas:**
- ✅ `system_modules` - Catálogo de módulos
- ✅ `subscription_plans` - Planos de assinatura
- ✅ `plan_entitlements` - Módulos por plano
- ✅ `tenant_subscriptions` - Assinaturas dos tenants
- ✅ `tenant_addons` - Módulos adicionais
- ✅ `tenant_entitlements` - View consolidada
- ✅ `check_tenant_module_access()` - Função SQL

### 4. Integração com Security

**Alterações:**
- ✅ `HulkClaims` agora inclui `TenantID uuid.UUID`
- ✅ `User` agora inclui `TenantID uuid.UUID`
- ✅ `AuthMiddleware` injeta `TenantID` no contexto

---

## 🔄 Fluxo de Ativação (Futuro)

### Passo 1: Atualizar `HasAccess()`

```go
func (s *Service) HasAccess(ctx context.Context, tenantID uuid.UUID, moduleCode string) bool {
    // 1. Cache Redis (performance)
    key := fmt.Sprintf("tenant:%s:entitlements", tenantID.String())
    allowed, err := s.redisClient.SIsMember(ctx, key, moduleCode).Result()
    if err == nil && allowed {
        return true
    }
    
    // 2. Consulta Banco
    var hasAccess bool
    err = s.store.QueryRow(ctx, 
        "SELECT check_tenant_module_access($1, $2)", 
        tenantID, moduleCode,
    ).Scan(&hasAccess)
    
    if err != nil {
        return false // Fail-secure
    }
    
    // 3. Cacheia resultado
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
INSERT INTO subscription_plans (name, code, price) VALUES
    ('Basic', 'plan.basic.v1', 99.90),
    ('Pro', 'plan.pro.v1', 299.90);

-- 2. Associar módulos aos planos
INSERT INTO plan_entitlements (plan_id, module_code)
SELECT sp.id, sm.code
FROM subscription_plans sp
CROSS JOIN system_modules sm
WHERE sp.code = 'plan.basic.v1'
    AND sm.code IN ('mod.platform.iam', 'mod.inventory.core');

-- 3. Criar assinatura para tenant
INSERT INTO tenant_subscriptions (tenant_id, plan_id, status, start_date)
VALUES (
    '550e8400-e29b-41d4-a716-446655440000'::UUID,
    (SELECT id FROM subscription_plans WHERE code = 'plan.basic.v1'),
    'ACTIVE',
    NOW()
);
```

### Passo 3: Configurar Redis (Opcional)

```go
// No app.go
redisClient := pgStore.RedisClient()
fm := featuremanager.NewServiceWithRedis(redisClient, store)
```

---

## ✅ O que ganhamos hoje?

1. **Segurança Arquitetural:** Nenhum código novo entra no sistema sem passar pelo "porteiro" de licenciamento.

2. **Zero Bloqueio:** Como o `HasAccess` retorna `true`, você continua desenvolvendo e testando sem precisar cadastrar planos no banco.

3. **Monetização Futura:** No dia que você quiser cobrar, basta:
   - Mudar o `return true` para a lógica do Redis/Banco.
   - Popular as tabelas SQL.
   - **Pronto:** O sistema inteiro passa a respeitar as regras de negócio instantaneamente.

---

## 📊 Status de Implementação

- [x] Interface `FeatureManager`
- [x] Service com modo stub
- [x] Middleware `RequireModule`
- [x] Schema SQL completo
- [x] View `tenant_entitlements`
- [x] Função SQL `check_tenant_module_access()`
- [x] TenantID adicionado ao `User` e `HulkClaims`
- [x] Testes unitários
- [x] Documentação completa
- [x] Exemplos de uso

---

## 🚀 Próximos Passos (Futuro)

- [ ] Implementar lógica real no `HasAccess()` (Redis + Banco)
- [ ] Adicionar FeatureManager ao Context do HULK (opcional)
- [ ] Criar testes de integração
- [ ] Implementar API para gerenciar planos e assinaturas
- [ ] Implementar webhooks para atualização de assinaturas
- [ ] Dashboard de gerenciamento de planos

---

**Última Atualização:** 2025-11-26  
**Versão:** v1.0.0 (Stub Mode)

