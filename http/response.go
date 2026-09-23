package http

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// Fase 0.1/0.5 do PLANEJAMENTO-EXPOSICAO-MODULOS-API-MCP-CONTRATOS:
// envelope de erro e paginação padrão. O formato é ADITIVO em relação ao
// legado {"error": "..."} — consumidores existentes continuam funcionando e
// os contratos publicados (schema Error em api/openapi/_shared/common.yaml)
// já preveem error_code opcional.

// ErrorBody é o envelope de erro padrão do Endurance
type ErrorBody struct {
	// Error é a mensagem legível (sem detalhes de infraestrutura interna)
	Error string `json:"error"`
	// ErrorCode é um código estável para tratamento programático (ex: OMS-001)
	ErrorCode string `json:"error_code,omitempty"`
	// TraceID correlaciona com logs/traces (X-Request-ID)
	TraceID string `json:"trace_id,omitempty"`
}

// Error responde com o envelope de erro padrão, propagando o X-Request-ID
// como trace_id. code pode ser vazio (campo omitido).
//
//	return hulk_http.Error(c, http.StatusNotFound, "OMS-404", "order not found")
func Error(c echo.Context, status int, code, message string) error {
	return c.JSON(status, ErrorBody{
		Error:     message,
		ErrorCode: code,
		TraceID:   requestID(c),
	})
}

// ErrorFromSDK é a variante para handlers que usam o Context do SDK (hulk_http.Context)
func ErrorFromSDK(c Context, status int, code, message string) error {
	body := ErrorBody{Error: message, ErrorCode: code}
	if echoCtx, ok := c.(*EchoContext); ok {
		body.TraceID = requestID(echoCtx.ctx)
	}
	return c.JSON(status, body)
}

// Pagination são os metadados de página do padrão 0.5
type Pagination struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// Page é a resposta paginada padrão: {"data": [...], "pagination": {...}}
type Page struct {
	Data       interface{} `json:"data"`
	Pagination Pagination  `json:"pagination"`
}

// NewPage monta a resposta paginada padrão a partir de page/pageSize 1-based.
// Use em endpoints NOVOS; endpoints já contratados mantêm o shape do contrato
// (mudar shape publicado = breaking change = /v2).
func NewPage(data interface{}, page, pageSize int, total int64) Page {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))
	return Page{
		Data: data,
		Pagination: Pagination{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: totalPages,
		},
	}
}

// UUIDParam lê um parâmetro de rota que vai DIRETO para uma coluna UUID e responde
// 400 se não for um UUID, antes de tocar o banco. Sem isso, um path que caia no `/:id`
// por engano (ex.: /products/pipeline) vira 500 com o erro cru do Postgres no corpo —
// erro de cliente contado como falha nossa, e detalhe de banco na API pública.
//
// Uso: `id, err := httpx.UUIDParam(c, "id"); if err != nil { return err }`.
// O gateway /ext/v1 tem a rede de segurança que sanitiza o que escapar; esta guarda
// evita a ida ao banco e devolve a mensagem certa.
//
// CUIDADO ao mexer: o erro devolvido tem de ser NÃO-NULO. `c.JSON` devolve nil em
// caso de sucesso, então devolver o resultado dele direto faria o handler seguir
// adiante com id vazio — e o `middleware.Recover()` do EchoServer esconderia o
// pânico atrás do 400 já escrito, com o teste passando pelo motivo errado.
// A resposta já foi escrita aqui; o DefaultHTTPErrorHandler do Echo vê
// Response().Committed e não escreve de novo.
func UUIDParam(c Context, name string) (string, error) {
	id := strings.TrimSpace(c.Param(name))
	if _, err := uuid.Parse(id); err != nil {
		_ = ErrorFromSDK(c, http.StatusBadRequest, "",
			fmt.Sprintf("%s inválido (esperado UUID)", name))
		return "", ErrInvalidUUID
	}
	return id, nil
}

// ErrInvalidUUID sinaliza que UUIDParam já respondeu 400 — o handler só precisa
// devolvê-lo para encerrar.
var ErrInvalidUUID = errors.New("parâmetro de rota não é um UUID")

func requestID(c echo.Context) string {
	if id := c.Response().Header().Get(echo.HeaderXRequestID); id != "" {
		return id
	}
	return c.Request().Header.Get(echo.HeaderXRequestID)
}
