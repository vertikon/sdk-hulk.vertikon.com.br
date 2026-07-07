package security

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

func TestHulkClaims_Structure(t *testing.T) {
	claims := &HulkClaims{
		UserID: "user-123",
		Email:  "test@example.com",
		Role:   "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "vertikon-endurance",
			Subject:   "user-123",
		},
	}

	assert.Equal(t, "user-123", claims.UserID)
	assert.Equal(t, "test@example.com", claims.Email)
	assert.Equal(t, "admin", claims.Role)
	assert.NotNil(t, claims.ExpiresAt)
}

func TestUser_Structure(t *testing.T) {
	user := User{
		ID:    "user-456",
		Email: "user@example.com",
		Role:  "user",
	}

	assert.Equal(t, "user-456", user.ID)
	assert.Equal(t, "user@example.com", user.Email)
	assert.Equal(t, "user", user.Role)
}

func TestUserContextKey_Constant(t *testing.T) {
	assert.Equal(t, "hulk_user", UserContextKey)
}
