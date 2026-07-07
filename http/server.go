package http

import (
	"context"
	"fmt"
	"mime/multipart"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type EchoServer struct {
	echo *echo.Echo
}

// Echo retorna a instância do Echo para uso em testes
func (s *EchoServer) Echo() *echo.Echo {
	return s.echo
}

func NewEchoServer() *EchoServer {
	e := echo.New()
	e.HideBanner = true

	// [BLOCO-P] Middleware de Observabilidade (deve vir ANTES de outros middlewares)
	// Para capturar tudo, incluindo erros e latência
	e.Use(OTelMiddleware("vertikon-monolith"))

        // Middlewares Padrão
        e.Use(middleware.Recover())
        e.Use(RequestContextMiddleware())
        e.Use(middleware.CORS())
        e.Use(middleware.RequestID())

        return &EchoServer{echo: e}
}

func (s *EchoServer) Start(port int) error {
	return s.echo.Start(fmt.Sprintf(":%d", port))
}

func (s *EchoServer) Stop(ctx context.Context) error {
	return s.echo.Shutdown(ctx)
}

// Implementação da Interface Router

func (s *EchoServer) GET(path string, handler HandlerFunc) {
	s.echo.GET(path, wrap(handler))
}

func (s *EchoServer) POST(path string, handler HandlerFunc) {
	s.echo.POST(path, wrap(handler))
}

func (s *EchoServer) PUT(path string, handler HandlerFunc) {
	s.echo.PUT(path, wrap(handler))
}

func (s *EchoServer) PATCH(path string, handler HandlerFunc) {
	s.echo.PATCH(path, wrap(handler))
}

func (s *EchoServer) DELETE(path string, handler HandlerFunc) {
	s.echo.DELETE(path, wrap(handler))
}

func (s *EchoServer) Group(prefix string) Router {
	return &EchoGroup{group: s.echo.Group(prefix)}
}

func (s *EchoServer) Use(middleware ...MiddlewareFunc) {
	for _, m := range middleware {
		s.echo.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
			return wrap(m(wrapEcho(next)))
		})
	}
}

// EchoGroup Wrapper

type EchoGroup struct {
	group *echo.Group
}

// EchoGroup returns the underlying echo.Group for direct middleware application
func (g *EchoGroup) EchoGroup() *echo.Group {
	return g.group
}

func (g *EchoGroup) GET(path string, handler HandlerFunc) {
	g.group.GET(path, wrap(handler))
}

func (g *EchoGroup) POST(path string, handler HandlerFunc) {
	g.group.POST(path, wrap(handler))
}

func (g *EchoGroup) PUT(path string, handler HandlerFunc) {
	g.group.PUT(path, wrap(handler))
}

func (g *EchoGroup) PATCH(path string, handler HandlerFunc) {
	g.group.PATCH(path, wrap(handler))
}

func (g *EchoGroup) DELETE(path string, handler HandlerFunc) {
	g.group.DELETE(path, wrap(handler))
}

func (g *EchoGroup) Group(prefix string) Router {
	return &EchoGroup{group: g.group.Group(prefix)}
}

func (g *EchoGroup) Use(middleware ...MiddlewareFunc) {
	for _, m := range middleware {
		g.group.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
			return wrap(m(wrapEcho(next)))
		})
	}
}

// UseEchoMiddleware allows direct application of Echo middleware functions
func (g *EchoGroup) UseEchoMiddleware(middleware ...echo.MiddlewareFunc) {
	g.group.Use(middleware...)
}

// Helper para converter Echo HandlerFunc para SDK HandlerFunc
func wrapEcho(h echo.HandlerFunc) HandlerFunc {
	return func(c Context) error {
		if echoCtx, ok := c.(*EchoContext); ok {
			return h(echoCtx.ctx)
		}
		return h(nil)
	}
}

// Helper para converter HandlerFunc do SDK para Echo Handler
func wrap(h HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		return h(&EchoContext{ctx: c})
	}
}

// WrapHandler converte um HandlerFunc do SDK para echo.HandlerFunc (exportado para uso por módulos)
func WrapHandler(h HandlerFunc) echo.HandlerFunc {
	return wrap(h)
}

// Implementação do Context

type EchoContext struct {
	ctx echo.Context
}

func (c *EchoContext) Bind(i interface{}) error {
	return c.ctx.Bind(i)
}

func (c *EchoContext) JSON(code int, i interface{}) error {
	return c.ctx.JSON(code, i)
}

func (c *EchoContext) NoContent(code int) error {
	return c.ctx.NoContent(code)
}

func (c *EchoContext) Param(name string) string {
	return c.ctx.Param(name)
}

func (c *EchoContext) QueryParam(name string) string {
	return c.ctx.QueryParam(name)
}

func (c *EchoContext) FormValue(name string) string {
	return c.ctx.FormValue(name)
}

func (c *EchoContext) FormFile(name string) (*multipart.FileHeader, error) {
	return c.ctx.FormFile(name)
}

func (c *EchoContext) Request() *http.Request {
	return c.ctx.Request()
}

func (c *EchoContext) Response() http.ResponseWriter {
	return c.ctx.Response().Writer
}
