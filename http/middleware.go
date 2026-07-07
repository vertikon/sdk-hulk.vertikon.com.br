package http

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/vertikon/sdk-hulk.vertikon.com.br/security"
)

// AuthConfig configura o middleware
type AuthConfig struct {
	JWTSecret string
}

// NewAuthMiddleware cria o middleware de proteção para o Echo
func NewAuthMiddleware(cfg AuthConfig) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// 1. Extrair o Token do Header (Authorization: Bearer <token>)
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Missing Authorization header"})
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid Authorization format"})
			}
			tokenString := parts[1]

			// 2. Parse e Validação do Token
			claims := &security.HulkClaims{}
			token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
				// Valida o algoritmo de assinatura
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, echo.NewHTTPError(http.StatusUnauthorized, "Unexpected signing method")
				}
				return []byte(cfg.JWTSecret), nil
			})

			if err != nil || !token.Valid {
				fmt.Printf("❌ Authentication Failed: %v\n", err)
				if token != nil {
					fmt.Printf("   Token Valid: %v\n", token.Valid)
				}
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid or expired token"})
			}

			// 3. Injeção no Contexto
			// Disponibiliza o usuário para o Handler do Módulo
			user := security.User{
				ID:       claims.UserID,
				Email:    claims.Email,
				Role:     claims.Role,
				TenantID: claims.TenantID, // Pode ser uuid.Nil em desenvolvimento
			}
			c.Set(security.UserContextKey, user)

			// 4. Segue para o próximo handler (o Módulo)
			return next(c)
		}
	}
}
