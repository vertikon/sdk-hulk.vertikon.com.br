package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// O gravador de resposta do middleware de idempotência precisa ser um http.Flusher: o reverse
// proxy do gateway chama Flush em respostas lentas e o echo entra em PANIC se o writer não
// suportar (derrubou o monólito em produção em 17/09/2026).
func TestResponseRecorder_Flush(t *testing.T) {
	rec := httptest.NewRecorder()
	var w http.ResponseWriter = &responseRecorder{ResponseWriter: rec, limit: 1024}
	f, ok := w.(http.Flusher)
	if !ok {
		t.Fatal("responseRecorder deveria implementar http.Flusher")
	}
	_, _ = w.Write([]byte("ok"))
	f.Flush()
	if !rec.Flushed {
		t.Fatal("Flush não foi delegado ao writer original")
	}
}
