# 📡 NATS Subjects - SDK-HULK

**Versão:** 1.0  
**Data:** 2025-11-24

---

## 📋 Visão Geral

Este documento descreve os padrões de nomenclatura e convenções para NATS subjects usados pelos módulos que implementam o SDK-HULK.

## 🎯 Convenções de Nomenclatura

### Formato Padrão

```
{module}.{domain}.{action}.{version}
```

### Componentes

- **module**: ID do módulo (ex: `inventory`, `sales`, `cognition`)
- **domain**: Domínio do evento (ex: `order`, `stock`, `payment`)
- **action**: Ação realizada (ex: `created`, `updated`, `reserved`)
- **version**: Versão do schema (ex: `v1`, `v2`)

### Exemplos

```
inventory.stock.reserved.v1
sales.order.created.v1
payment.transaction.completed.v1
```

## 📚 Subjects por Módulo

### Inventory (Estoque)

| Subject | Descrição | Publisher | Subscriber |
|---------|-----------|-----------|------------|
| `inventory.stock.reserved.v1` | Estoque reservado | Inventory Module | Fulfillment, Sales |
| `inventory.stock.released.v1` | Estoque liberado | Inventory Module | Sales |
| `inventory.stock.updated.v1` | Atualização de estoque | Inventory Module | Analytics, MDM |
| `inventory.stock.low.v1` | Estoque baixo (alerta) | Inventory Module | Notifications, Procurement |

### Sales (Vendas)

| Subject | Descrição | Publisher | Subscriber |
|---------|-----------|-----------|------------|
| `sales.order.created.v1` | Ordem de venda criada | Sales Module | Inventory, Payment |
| `sales.order.cancelled.v1` | Ordem cancelada | Sales Module | Inventory, Fulfillment |
| `sales.order.completed.v1` | Ordem completada | Sales Module | Analytics, CRM |
| `sales.payment.processed.v1` | Pagamento processado | Sales Module | Fulfillment, Inventory |

### Payment (Pagamento)

| Subject | Descrição | Publisher | Subscriber |
|---------|-----------|-----------|------------|
| `payment.transaction.created.v1` | Transação criada | Payment Module | Sales, Fraud Detection |
| `payment.transaction.completed.v1` | Transação completada | Payment Module | Sales, Fulfillment |
| `payment.transaction.failed.v1` | Transação falhou | Payment Module | Sales, Notifications |

### Fulfillment (Fulfillment)

| Subject | Descrição | Publisher | Subscriber |
|---------|-----------|-----------|------------|
| `fulfillment.shipment.created.v1` | Envio criado | Fulfillment Module | Sales, Logistics |
| `fulfillment.shipment.dispatched.v1` | Envio despachado | Fulfillment Module | Sales, Notifications |
| `fulfillment.shipment.delivered.v1` | Envio entregue | Fulfillment Module | Sales, Analytics |

## 🔄 Padrões de Comunicação

### Request-Reply

Para comunicação síncrona, use o padrão request-reply do NATS:

```
Request:  {module}.{domain}.{action}.request.v1
Reply:    {module}.{domain}.{action}.reply.v1
```

**Exemplo:**
```
Request:  inventory.stock.check.request.v1
Reply:    inventory.stock.check.reply.v1
```

### Pub-Sub

Para eventos assíncronos, use pub-sub:

```
Topic:    {module}.{domain}.{action}.v1
```

### Queue Groups

Para load balancing entre instâncias do mesmo módulo:

```
Topic:    {module}.{domain}.{action}.v1
Queue:    {module}-{action}-queue
```

**Exemplo:**
```go
ctx.EventBus().QueueSubscribe(
    "sales.order.created.v1",
    "inventory-reserve-queue",
    handler,
)
```

## 📝 Versionamento

### Quando Versionar

- Mudança incompatível no schema do payload
- Adição de campos obrigatórios
- Remoção de campos

### Estratégia

1. **v1**: Versão inicial
2. **v2**: Nova versão com mudanças incompatíveis
3. Manter v1 ativo durante período de transição
4. Deprecar v1 após migração completa

## 🔐 Segurança

### Autenticação

- Todos os subjects devem usar autenticação NATS
- Use NKeys para autenticação segura
- Configure permissões por módulo

### Validação

- Valide sempre o payload antes de processar
- Use schemas (JSON Schema, AsyncAPI) para validação
- Rejeite mensagens com schema inválido

## 📊 Monitoramento

### Métricas Recomendadas

- Mensagens publicadas por subject
- Mensagens consumidas por subject
- Latência de processamento
- Taxa de erro por subject

### Logging

- Log todas as mensagens publicadas (nível DEBUG)
- Log erros de processamento (nível ERROR)
- Inclua TraceID em todos os logs

## 🛠️ Uso no SDK

### Publicar Evento

```go
err := ctx.EventBus().Publish("inventory.stock.reserved.v1", map[string]interface{}{
    "sku":      "PROD-001",
    "quantity": 10,
    "order_id": "ORD-123",
})
```

### Inscrever-se em Evento

```go
err := ctx.EventBus().Subscribe("sales.order.created.v1", func(msg events.Message) error {
    // Processar mensagem
    return msg.Ack()
})
```

### Queue Subscribe (Load Balancing)

```go
err := ctx.EventBus().QueueSubscribe(
    "sales.order.created.v1",
    "inventory-queue",
    func(msg events.Message) error {
        // Processar mensagem
        return msg.Ack()
    },
)
```

## 📚 Referências

- [NATS Documentation](https://docs.nats.io/)
- [NATS JetStream](https://docs.nats.io/nats-concepts/jetstream)
- [AsyncAPI Specification](https://www.asyncapi.com/)

---

**Última atualização:** 2025-11-24  
**Mantido por:** Equipe Vertikon

