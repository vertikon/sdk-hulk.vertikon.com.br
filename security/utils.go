package security

import "os"

// ResolveJWTSecret retrieves the JWT secret from environment or returns a default for dev
func ResolveJWTSecret() (string, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return "endurance-dev-secret-key-change-in-prod", nil
	}
	return secret, nil
}
