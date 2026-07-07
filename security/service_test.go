package security

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestServiceHasScope(t *testing.T) {
	tests := []struct {
		name     string
		scopes   []string
		required string
		want     bool
	}{
		{"exact match", []string{"finance:read"}, "finance:read", true},
		{"missing scope", []string{"finance:read"}, "finance:write", false},
		{"domain wildcard", []string{"oms:*"}, "oms:write", true},
		{"domain wildcard other domain", []string{"oms:*"}, "finance:read", false},
		{"global wildcard", []string{"*"}, "anything:at-all", true},
		{"empty scopes", nil, "finance:read", false},
		{"wildcard does not match bare domain prefix", []string{"oms:*"}, "omsx:read", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &Service{Scopes: tt.scopes}
			if got := svc.HasScope(tt.required); got != tt.want {
				t.Errorf("HasScope(%q) com scopes %v = %v, esperado %v", tt.required, tt.scopes, got, tt.want)
			}
		})
	}
}

func TestServiceCanAccessTenant(t *testing.T) {
	allowed := uuid.New()
	other := uuid.New()
	svc := &Service{TenantIDs: []uuid.UUID{allowed}}

	if !svc.CanAccessTenant(allowed) {
		t.Error("tenant permitido foi negado")
	}
	if svc.CanAccessTenant(other) {
		t.Error("tenant não autorizado foi permitido")
	}
}

func TestGenerateAPIKey(t *testing.T) {
	plainLive, hashLive, err := GenerateAPIKey(true)
	if err != nil {
		t.Fatalf("GenerateAPIKey(live): %v", err)
	}
	if !strings.HasPrefix(plainLive, APIKeyPrefixLive) {
		t.Errorf("chave live sem prefixo %s: %s", APIKeyPrefixLive, plainLive)
	}
	if hashLive != HashAPIKey(plainLive) {
		t.Error("hash retornado difere de HashAPIKey(plaintext)")
	}

	plainTest, _, err := GenerateAPIKey(false)
	if err != nil {
		t.Fatalf("GenerateAPIKey(test): %v", err)
	}
	if !strings.HasPrefix(plainTest, APIKeyPrefixTest) {
		t.Errorf("chave test sem prefixo %s: %s", APIKeyPrefixTest, plainTest)
	}

	if !IsAPIKey(plainLive) || !IsAPIKey(plainTest) {
		t.Error("IsAPIKey deveria reconhecer chaves geradas")
	}
	if IsAPIKey("eyJhbGciOiJIUzI1NiJ9.jwt.token") {
		t.Error("IsAPIKey não deveria reconhecer JWT")
	}

	// Unicidade básica
	plain2, _, _ := GenerateAPIKey(true)
	if plainLive == plain2 {
		t.Error("duas chaves geradas são idênticas")
	}
}
