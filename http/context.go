package http

import (
	"github.com/labstack/echo/v4"
	"github.com/vertikon/sdk-hulk.vertikon.com.br/security"
)

// GetUserFromContext extrai o usuário autenticado de forma segura do contexto Echo
func GetUserFromContext(c echo.Context) *security.User {
	user, ok := c.Get(security.UserContextKey).(security.User)
	if !ok {
		return nil
	}
	return &user
}

// GetUserFromSDKContext extrai o usuário autenticado do contexto SDK
func GetUserFromSDKContext(c Context) *security.User {
	// Se o contexto for EchoContext, extrai o echo.Context interno
	if echoCtx, ok := c.(*EchoContext); ok {
		return GetUserFromContext(echoCtx.ctx)
	}
	return nil
}
