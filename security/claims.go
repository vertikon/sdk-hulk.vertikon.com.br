package security

import (
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// HulkClaims define o padrão de token para todo o ecossistema Vertikon
type HulkClaims struct {
	UserID   string    `json:"user_id"`
	Email    string    `json:"email"`
	Role     string    `json:"role"`
	TenantID uuid.UUID `json:"tenant_id,omitempty"` // UUID do tenant (opcional para desenvolvimento)
	jwt.RegisteredClaims
}

// UserContextKey é a chave usada para guardar o usuário dentro do contexto
const UserContextKey = "hulk_user"

// User representa o usuário extraído do token para uso nos módulos
type User struct {
	ID       string
	Email    string
	Role     string
	TenantID uuid.UUID // UUID do tenant (uuid.Nil em desenvolvimento)
}
