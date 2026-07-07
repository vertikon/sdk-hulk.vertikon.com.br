package featuremanager

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestService_HasAccess_StubMode(t *testing.T) {
	service := NewService()
	ctx := context.Background()
	tenantID := uuid.New()

	// Em stub mode, sempre retorna true (desenvolvimento)
	assert.True(t, service.HasAccess(ctx, tenantID, "mod.fiscal.issuer"))
	assert.True(t, service.HasAccess(ctx, tenantID, "mod.any.module"))
	assert.True(t, service.HasAccess(ctx, uuid.Nil, "mod.test")) // Mesmo com tenant nil
}

func TestService_HasAccess_WithDifferentTenants(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	tenant1 := uuid.New()
	tenant2 := uuid.New()

	// Em stub mode, todos os tenants têm acesso
	assert.True(t, service.HasAccess(ctx, tenant1, "mod.fiscal.issuer"))
	assert.True(t, service.HasAccess(ctx, tenant2, "mod.fiscal.issuer"))
	assert.True(t, service.HasAccess(ctx, tenant1, "mod.checkout.engine"))
}

func TestService_InterfaceCompliance(t *testing.T) {
	// Garantir que Service implementa FeatureManager
	var _ FeatureManager = (*Service)(nil)

	service := NewService()
	assert.NotNil(t, service)
}
