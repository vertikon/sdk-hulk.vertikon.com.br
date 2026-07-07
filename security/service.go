package security

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// ServiceContextKey é a chave usada para guardar o sistema consumidor (M2M) dentro do contexto
const ServiceContextKey = "hulk_service"

// APIKeyPrefixLive e APIKeyPrefixTest identificam o ambiente da credencial M2M
const (
	APIKeyPrefixLive = "vtk_live_"
	APIKeyPrefixTest = "vtk_test_"
)

// Service representa um sistema consumidor autenticado via API key (máquina-a-máquina)
type Service struct {
	ID        string      // ID da API key
	Name      string      // Nome do sistema consumidor (ex: "erp-acme")
	Scopes    []string    // Scopes no formato "dominio:acao" (ex: "finance:read", "oms:*")
	TenantIDs []uuid.UUID // Tenants que a credencial pode acessar (vazio = nenhum tenant específico)
}

// HasScope verifica se o serviço possui o scope exigido.
// Suporta curinga por domínio ("finance:*") e curinga global ("*").
func (s *Service) HasScope(required string) bool {
	for _, scope := range s.Scopes {
		if scope == "*" || scope == required {
			return true
		}
		if domain, ok := strings.CutSuffix(scope, ":*"); ok {
			if strings.HasPrefix(required, domain+":") {
				return true
			}
		}
	}
	return false
}

// CanAccessTenant verifica se a credencial pode operar sobre o tenant informado.
func (s *Service) CanAccessTenant(tenantID uuid.UUID) bool {
	for _, t := range s.TenantIDs {
		if t == tenantID {
			return true
		}
	}
	return false
}

// IsAPIKey informa se um token de Authorization é uma API key Vertikon (vs JWT)
func IsAPIKey(token string) bool {
	return strings.HasPrefix(token, APIKeyPrefixLive) || strings.HasPrefix(token, APIKeyPrefixTest)
}

// GenerateAPIKey gera uma nova API key em claro e o hash a persistir.
// A chave em claro deve ser exibida UMA única vez ao criador; apenas o hash é armazenado.
func GenerateAPIKey(live bool) (plaintext string, hash string, err error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", "", fmt.Errorf("failed to generate api key: %w", err)
	}
	prefix := APIKeyPrefixTest
	if live {
		prefix = APIKeyPrefixLive
	}
	plaintext = prefix + hex.EncodeToString(raw)
	return plaintext, HashAPIKey(plaintext), nil
}

// HashAPIKey calcula o hash SHA-256 (hex) de uma API key para lookup/armazenamento.
func HashAPIKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

// MintServiceToken emite um JWT interno de curta duração representando um sistema
// consumidor autenticado por API key (token exchange na borda). Usado pelo
// external-gateway e pelo hulk-mcp para chamar o plano interno /api/v1 sem que
// os módulos JWT-only precisem mudar.
func MintServiceToken(secret string, svc *Service, tenantID uuid.UUID, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := &HulkClaims{
		UserID:   "system:" + svc.Name,
		Email:    svc.Name + "@systems.vertikon.internal",
		Role:     "service",
		TenantID: tenantID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "system:" + svc.Name,
			Issuer:    "edge-token-exchange",
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

// APIKeyValidator valida uma API key em claro e devolve a identidade do sistema consumidor.
// Implementado pelo iam-sso; consumido pelo middleware HTTP do SDK.
type APIKeyValidator interface {
	ValidateAPIKey(ctx context.Context, rawKey string) (*Service, error)
}

var (
	validatorMu     sync.RWMutex
	globalValidator APIKeyValidator
)

// RegisterAPIKeyValidator registra o validador global de API keys (chamado pelo iam-sso no Init).
func RegisterAPIKeyValidator(v APIKeyValidator) {
	validatorMu.Lock()
	defer validatorMu.Unlock()
	globalValidator = v
}

// GetAPIKeyValidator devolve o validador global registrado (nil se nenhum módulo registrou).
func GetAPIKeyValidator() APIKeyValidator {
	validatorMu.RLock()
	defer validatorMu.RUnlock()
	return globalValidator
}
