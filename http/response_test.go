package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestErrorEnvelope(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(echo.HeaderXRequestID, "req-123")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := Error(c, http.StatusNotFound, "OMS-404", "order not found"); err != nil {
		t.Fatalf("Error: %v", err)
	}
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rec.Code)
	}
	var body ErrorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json: %v", err)
	}
	if body.Error != "order not found" || body.ErrorCode != "OMS-404" || body.TraceID != "req-123" {
		t.Errorf("envelope incorreto: %+v", body)
	}

	// Compatibilidade com consumidores legados que leem só {"error": string}
	var legacy struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &legacy); err != nil || legacy.Error == "" {
		t.Errorf("envelope não é retrocompatível: %s", rec.Body.String())
	}
}

func TestErrorEnvelopeOmitsEmptyFields(t *testing.T) {
	e := echo.New()
	c := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), httptest.NewRecorder())
	rec := c.Response().Writer.(*httptest.ResponseRecorder)

	_ = Error(c, http.StatusBadRequest, "", "bad input")
	var raw map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &raw)
	if _, has := raw["error_code"]; has {
		t.Error("error_code vazio deveria ser omitido")
	}
	if _, has := raw["trace_id"]; has {
		t.Error("trace_id vazio deveria ser omitido")
	}
}

func TestNewPage(t *testing.T) {
	page := NewPage([]string{"a", "b"}, 2, 10, 25)
	if page.Pagination.TotalPages != 3 {
		t.Errorf("total_pages = %d, esperado 3", page.Pagination.TotalPages)
	}
	if page.Pagination.Page != 2 || page.Pagination.PageSize != 10 || page.Pagination.Total != 25 {
		t.Errorf("pagination incorreta: %+v", page.Pagination)
	}

	// Defaults defensivos
	page = NewPage(nil, 0, 0, 0)
	if page.Pagination.Page != 1 || page.Pagination.PageSize != 20 || page.Pagination.TotalPages != 0 {
		t.Errorf("defaults incorretos: %+v", page.Pagination)
	}
}
