# 📖 Exemplos de Uso - Feature Manager

Este documento mostra exemplos práticos de como usar o Feature Manager em diferentes módulos.

---

## 🎯 Exemplo 1: Fiscal Document Issuer

### Arquivo: `internal/modules/fiscal/fiscal-document-issuer/module.go`

```go
package fiscal_document_issuer

import (
	"context"

	"github.com/vertikon/endurance/pkg/sdk-hulk"
	sdkhttp "github.com/vertikon/endurance/pkg/sdk-hulk/http"
	featuremanager "github.com/vertikon/endurance/pkg/sdk-hulk/feature-manager"
	"go.uber.org/zap"
)

func (m *Module) Start(ctx hulk.Context) error {
	ctx.Log().Info("Iniciando módulo Fiscal Document Issuer...")

	// 1. Inicializa o Feature Manager
	fm := featuremanager.NewService()

	// 2. Configura middleware de autenticação
	jwtSecret := "change_me_in_production" // TODO: Carregar do config
	authMiddleware := sdkhttp.NewAuthMiddleware(sdkhttp.AuthConfig{
		JWTSecret: jwtSecret,
	})

	// 3. Cria grupo de rotas protegido
	fiscalGroup := ctx.HTTP().Group("/api/v1/fiscal")

	// 4. Aplica middlewares (ORDEM IMPORTA!)
	// Primeiro: Autenticação (valida JWT)
	if echoGroup, ok := fiscalGroup.(*sdkhttp.EchoGroup); ok {
		echoGroup.UseEchoMiddleware(authMiddleware)
		
		// Depois: Permissão de Módulo (verifica se tem acesso)
		echoGroup.UseEchoMiddleware(featuremanager.RequireModule(fm, "mod.fiscal.issuer"))
	}

	// 5. Registra rotas (só chegam aqui se passaram pelos middlewares)
	handler := NewHTTPHandler(m.service)
	handler.RegisterRoutes(fiscalGroup, ctx)

	return nil
}
```

---

## 🎯 Exemplo 2: WMS Core (Múltiplos Módulos)

### Arquivo: `internal/modules/logistics/wms-core/module.go`

```go
func (m *Module) Start(ctx hulk.Context) error {
	ctx.Log().Info("Iniciando módulo WMS Core...")

	fm := featuremanager.NewService()
	authMiddleware := sdkhttp.NewAuthMiddleware(sdkhttp.AuthConfig{
		JWTSecret: "change_me_in_production",
	})

	// Grupo base de logística
	logisticsGroup := ctx.HTTP().Group("/api/v1/logistics")
	
	if echoGroup, ok := logisticsGroup.(*sdkhttp.EchoGroup); ok {
		echoGroup.UseEchoMiddleware(authMiddleware)
		
		// Requer módulo WMS
		echoGroup.UseEchoMiddleware(featuremanager.RequireModule(fm, "mod.logistics.wms"))
	}

	// Sub-grupo para funcionalidades avançadas (requer módulo adicional)
	advancedGroup := logisticsGroup.Group("/advanced")
	if echoGroup, ok := advancedGroup.(*sdkhttp.EchoGroup); ok {
		// Requer módulo adicional de AI para análise de rotas
		echoGroup.UseEchoMiddleware(featuremanager.RequireModule(fm, "mod.ai.vision"))
	}

	handler := NewHTTPHandler(m.service)
	handler.RegisterRoutes(logisticsGroup, ctx)

	return nil
}
```

---

## 🎯 Exemplo 3: Verificação Programática (Dentro de Handler)

### Arquivo: `internal/modules/payments/payment-gateway/handler.go`

```go
package payment_gateway

import (
	"github.com/labstack/echo/v4"
	featuremanager "github.com/vertikon/endurance/pkg/sdk-hulk/feature-manager"
	sdkhttp "github.com/vertikon/endurance/pkg/sdk-hulk/http"
)

func (h *HTTPHandler) ProcessPayment(c echo.Context) error {
	// 1. Verifica autenticação
	user := sdkhttp.GetUserFromContext(c)
	if user == nil {
		return c.JSON(401, map[string]string{"error": "unauthorized"})
	}

	// 2. Verifica permissão de módulo (programaticamente)
	fm := featuremanager.NewService()
	if user.TenantID != uuid.Nil { // Só verifica se não for desenvolvimento
		if !fm.HasAccess(c.Request().Context(), user.TenantID, "mod.payments.gateway") {
			return c.JSON(403, map[string]interface{}{
				"error":       "module_not_enabled",
				"message":     "Seu plano não inclui processamento de pagamentos.",
				"module_code": "mod.payments.gateway",
				"action":      "UPGRADE_PLAN",
			})
		}
	}

	// 3. Processa pagamento normalmente
	// ... lógica do handler ...
	
	return c.JSON(200, map[string]string{"status": "processed"})
}
```

---

## 🎯 Exemplo 4: Módulo com Múltiplas Funcionalidades

### Arquivo: `internal/modules/checkout/checkout-engine/module.go`

```go
func (m *Module) Start(ctx hulk.Context) error {
	ctx.Log().Info("Iniciando módulo Checkout Engine...")

	fm := featuremanager.NewService()
	authMiddleware := sdkhttp.NewAuthMiddleware(sdkhttp.AuthConfig{
		JWTSecret: "change_me_in_production",
	})

	checkoutGroup := ctx.HTTP().Group("/api/v1/checkout")
	
	if echoGroup, ok := checkoutGroup.(*sdkhttp.EchoGroup); ok {
		echoGroup.UseEchoMiddleware(authMiddleware)
		echoGroup.UseEchoMiddleware(featuremanager.RequireModule(fm, "mod.checkout.engine"))
	}

	// Sub-rota que requer módulo adicional (ex: cálculo fiscal em tempo real)
	fiscalSubGroup := checkoutGroup.Group("/tax-calculation")
	if echoGroup, ok := fiscalSubGroup.(*sdkhttp.EchoGroup); ok {
		// Requer módulo fiscal adicional
		echoGroup.UseEchoMiddleware(featuremanager.RequireModule(fm, "mod.fiscal.tax"))
	}

	handler := NewHTTPHandler(m.service)
	handler.RegisterRoutes(checkoutGroup, ctx)

	return nil
}
```

---

## 🎯 Exemplo 5: Rotas Públicas vs Protegidas

### Arquivo: `internal/modules/mdm/pim-product-master/module.go`

```go
func (m *Module) Start(ctx hulk.Context) error {
	ctx.Log().Info("Iniciando módulo PIM Product Master...")

	fm := featuremanager.NewService()
	authMiddleware := sdkhttp.NewAuthMiddleware(sdkhttp.AuthConfig{
		JWTSecret: "change_me_in_production",
	})

	pimGroup := ctx.HTTP().Group("/api/v1/mdm/products")

	// Rotas públicas (sem autenticação, sem verificação de módulo)
	pimGroup.GET("/public/:id", m.Handler.GetProductPublic) // Catálogo público

	// Rotas protegidas (com autenticação e verificação de módulo)
	protectedGroup := pimGroup.Group("/admin")
	if echoGroup, ok := protectedGroup.(*sdkhttp.EchoGroup); ok {
		echoGroup.UseEchoMiddleware(authMiddleware)
		echoGroup.UseEchoMiddleware(featuremanager.RequireModule(fm, "mod.mdm.pim"))
	}

	protectedGroup.POST("", m.Handler.CreateProduct)
	protectedGroup.PUT("/:id", m.Handler.UpdateProduct)
	protectedGroup.DELETE("/:id", m.Handler.DeleteProduct)

	return nil
}
```

---

## 🔄 Fluxo de Verificação

```
Cliente → HTTP Request
    ↓
1. AuthMiddleware (valida JWT)
    ↓ (se válido)
2. Injeta User no contexto (com TenantID)
    ↓
3. RequireModule (verifica permissão)
    ↓ (se TenantID == uuid.Nil → permite)
    ↓ (se TenantID != uuid.Nil → consulta FeatureManager)
    ↓ (se HasAccess() == true → permite)
    ↓
4. Handler do Módulo (executa lógica)
    ↓
5. Response
```

---

## ⚠️ Ordem dos Middlewares

**IMPORTANTE:** A ordem dos middlewares importa!

```go
// ✅ CORRETO: Auth primeiro, depois Feature Manager
group.UseEchoMiddleware(authMiddleware)
group.UseEchoMiddleware(featuremanager.RequireModule(fm, "mod.test"))

// ❌ ERRADO: Feature Manager antes do Auth
group.UseEchoMiddleware(featuremanager.RequireModule(fm, "mod.test"))
group.UseEchoMiddleware(authMiddleware) // User ainda não está no contexto!
```

---

## 📝 Códigos de Módulos Recomendados

| Módulo | Código Sugerido |
|--------|-----------------|
| Fiscal Document Issuer | `mod.fiscal.issuer` |
| Tax Intelligence | `mod.fiscal.tax` |
| Payment Gateway | `mod.payments.gateway` |
| Antifraud Engine | `mod.payments.antifraud` |
| Checkout Engine | `mod.checkout.engine` |
| OMS Core | `mod.checkout.oms` |
| WMS Core | `mod.logistics.wms` |
| Procurement Core | `mod.procurement.core` |
| Financial Core | `mod.finance.core` |
| PIM Product Master | `mod.mdm.pim` |

---

**Última Atualização:** 2025-11-26

