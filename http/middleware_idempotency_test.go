package http

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
)

func newIdempotentEcho(handlerCalls *atomic.Int32, store IdempotencyStore) *echo.Echo {
	e := echo.New()
	mw := NewIdempotencyMiddleware(IdempotencyConfig{Store: store, TTL: time.Minute})
	e.POST("/orders", func(c echo.Context) error {
		n := handlerCalls.Add(1)
		return c.JSON(http.StatusCreated, map[string]interface{}{"order": fmt.Sprintf("order-%d", n)})
	}, mw)
	e.GET("/orders", func(c echo.Context) error {
		handlerCalls.Add(1)
		return c.JSON(http.StatusOK, map[string]string{"ok": "true"})
	}, mw)
	return e
}

func postWithKey(e *echo.Echo, key string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/orders", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func TestIdempotencyReplaysResponse(t *testing.T) {
	var calls atomic.Int32
	e := newIdempotentEcho(&calls, nil)

	first := postWithKey(e, "key-1")
	if first.Code != http.StatusCreated {
		t.Fatalf("primeira: status = %d", first.Code)
	}
	second := postWithKey(e, "key-1")
	if second.Code != http.StatusCreated {
		t.Fatalf("replay: status = %d", second.Code)
	}
	if second.Header().Get("Idempotency-Replayed") != "true" {
		t.Error("replay sem header Idempotency-Replayed")
	}
	if first.Body.String() != second.Body.String() {
		t.Errorf("respostas diferem: %q vs %q", first.Body.String(), second.Body.String())
	}
	if calls.Load() != 1 {
		t.Errorf("handler executado %d vezes, esperado 1", calls.Load())
	}
}

func TestIdempotencyDifferentKeysExecute(t *testing.T) {
	var calls atomic.Int32
	e := newIdempotentEcho(&calls, nil)

	postWithKey(e, "key-a")
	postWithKey(e, "key-b")
	if calls.Load() != 2 {
		t.Errorf("handler executado %d vezes, esperado 2", calls.Load())
	}
}

func TestIdempotencyWithoutKeyOrOnGetIsBypassed(t *testing.T) {
	var calls atomic.Int32
	e := newIdempotentEcho(&calls, nil)

	postWithKey(e, "")
	postWithKey(e, "")
	if calls.Load() != 2 {
		t.Errorf("sem Idempotency-Key: handler executado %d vezes, esperado 2", calls.Load())
	}

	calls.Store(0)
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/orders", nil)
		req.Header.Set("Idempotency-Key", "key-get")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
	}
	if calls.Load() != 2 {
		t.Errorf("GET: handler executado %d vezes, esperado 2 (idempotência só em métodos mutantes)", calls.Load())
	}
}

func TestIdempotencyInProgressConflict(t *testing.T) {
	store := NewMemoryIdempotencyStore()
	var calls atomic.Int32
	e := newIdempotentEcho(&calls, store)

	// Simular requisição em andamento reservando a chave manualmente
	req := httptest.NewRequest(http.MethodPost, "/orders", strings.NewReader(`{}`))
	req.Header.Set("Idempotency-Key", "key-busy")
	c := e.NewContext(req, httptest.NewRecorder())
	store.Set(cacheKey(c, "key-busy"), &IdempotencyRecord{InProgress: true, StoredAt: time.Now()}, time.Minute)

	rec := postWithKey(e, "key-busy")
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, esperado 409; body: %s", rec.Code, rec.Body.String())
	}
	if calls.Load() != 0 {
		t.Error("handler não deveria executar com chave em andamento")
	}
}

func TestIdempotencyServerErrorNotCached(t *testing.T) {
	store := NewMemoryIdempotencyStore()
	var calls atomic.Int32
	e := echo.New()
	mw := NewIdempotencyMiddleware(IdempotencyConfig{Store: store, TTL: time.Minute})
	e.POST("/flaky", func(c echo.Context) error {
		if calls.Add(1) == 1 {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "boom"})
		}
		return c.JSON(http.StatusCreated, map[string]string{"ok": "true"})
	}, mw)

	send := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/flaky", strings.NewReader(`{}`))
		req.Header.Set("Idempotency-Key", "key-retry")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		return rec
	}

	if rec := send(); rec.Code != http.StatusInternalServerError {
		t.Fatalf("primeira: status = %d", rec.Code)
	}
	// 5xx não é cacheado: retry executa de novo e pode suceder
	if rec := send(); rec.Code != http.StatusCreated {
		t.Fatalf("retry: status = %d, esperado 201", rec.Code)
	}
	if calls.Load() != 2 {
		t.Errorf("handler executado %d vezes, esperado 2", calls.Load())
	}
}
