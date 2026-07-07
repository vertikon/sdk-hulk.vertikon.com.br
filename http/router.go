package http

import (
	"mime/multipart"
	"net/http"
)

// HandlerFunc define o padrão de função que processa requisições (Compatible with Echo)
type HandlerFunc func(c Context) error

// Context abstrai o contexto HTTP (Request/Response)
type Context interface {
	Bind(i interface{}) error
	JSON(code int, i interface{}) error
	NoContent(code int) error
	Param(name string) string
	QueryParam(name string) string
	FormValue(name string) string
	FormFile(name string) (*multipart.FileHeader, error)
	Request() *http.Request
	Response() http.ResponseWriter
}

// MiddlewareFunc define uma função de middleware
type MiddlewareFunc func(next HandlerFunc) HandlerFunc

// Router define como os módulos registram suas rotas
type Router interface {
	GET(path string, handler HandlerFunc)
	POST(path string, handler HandlerFunc)
	PUT(path string, handler HandlerFunc)
	PATCH(path string, handler HandlerFunc)
	DELETE(path string, handler HandlerFunc)
	Group(prefix string) Router
	Use(middleware ...MiddlewareFunc)
}
