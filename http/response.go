package http

import (
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

func requestID(c echo.Context) string {
	if id := c.Response().Header().Get(echo.HeaderXRequestID); id != "" {
		return id
	}
	return c.Request().Header.Get(echo.HeaderXRequestID)
}
