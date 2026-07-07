package featuremanager

import (
	"context"

	"github.com/google/uuid"
	// "github.com/redis/go-redis/v9" // Futuro: Para cachear permissões e evitar SQL lento
)

// FeatureManager define o contrato para verificar permissões de módulos
type FeatureManager interface {
	// HasAccess verifica se um Tenant tem permissão para usar um módulo específico
	HasAccess(ctx context.Context, tenantID uuid.UUID, moduleCode string) bool
}

// Service implementa a lógica de verificação
type Service struct {
	// redisClient *redis.Client // Futuro: Para cachear permissões e evitar SQL lento
	// store       state.Store   // Futuro: Para consultar banco de dados
}

// NewService cria uma nova instância do gerenciador
func NewService() *Service {
	return &Service{}
}

// HasAccess é o "Stub" (Dublê).
// HOJE: Retorna sempre true para não bloquear o desenvolvimento.
// FUTURO: Vai consultar Redis/Banco para ver se o plano do cliente inclui o módulo.
func (s *Service) HasAccess(ctx context.Context, tenantID uuid.UUID, moduleCode string) bool {
	// ---------------------------------------------------------
	// MODO DESENVOLVIMENTO: TUDO LIBERADO
	// ---------------------------------------------------------
	return true

	// ---------------------------------------------------------
	// MODO PRODUÇÃO (Lógica Futura):
	// ---------------------------------------------------------
	// 1. Verificar cache Redis primeiro (performance)
	// key := fmt.Sprintf("tenant:%s:entitlements", tenantID.String())
	// allowed, err := s.redisClient.SIsMember(ctx, key, moduleCode).Result()
	// if err == nil && allowed {
	// 	return true
	// }
	//
	// 2. Se não estiver no cache, consultar banco de dados
	// var hasAccess bool
	// query := `
	// 	SELECT EXISTS(
	// 		SELECT 1
	// 		FROM tenant_subscriptions ts
	// 		JOIN plan_entitlements pe ON ts.plan_id = pe.plan_id
	// 		WHERE ts.tenant_id = $1
	// 			AND ts.status = 'ACTIVE'
	// 			AND pe.module_code = $2
	// 	) OR EXISTS(
	// 		SELECT 1
	// 		FROM tenant_addons ta
	// 		WHERE ta.tenant_id = $1
	// 			AND ta.module_code = $2
	// 	)
	// `
	// err := s.store.QueryRow(ctx, query, tenantID, moduleCode).Scan(&hasAccess)
	// if err != nil {
	// 	return false // Em caso de erro, negar acesso (fail-secure)
	// }
	//
	// 3. Cachear resultado no Redis (TTL: 5 minutos)
	// if hasAccess {
	// 	s.redisClient.SAdd(ctx, key, moduleCode)
	// 	s.redisClient.Expire(ctx, key, 5*time.Minute)
	// }
	//
	// return hasAccess
}
