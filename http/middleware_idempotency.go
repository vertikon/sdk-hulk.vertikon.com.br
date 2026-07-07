package http

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
)

// IdempotencyRecord é a resposta armazenada de uma requisição idempotente
type IdempotencyRecord struct {
	StatusCode  int       `json:"status_code"`
	ContentType string    `json:"content_type"`
	Body        []byte    `json:"body"`
	InProgress  bool      `json:"in_progress"`
	StoredAt    time.Time `json:"stored_at"`
}

// IdempotencyStore abstrai o armazenamento dos registros (memória, Redis...).
// SetIfAbsent deve ser atômico: retorna false se a chave já existia.
type IdempotencyStore interface {
	Get(key string) (*IdempotencyRecord, bool)
	SetIfAbsent(key string, record *IdempotencyRecord, ttl time.Duration) bool
	Set(key string, record *IdempotencyRecord, ttl time.Duration)
	Delete(key string)
}

// IdempotencyConfig configura o middleware de idempotência (Fase 0.4)
type IdempotencyConfig struct {
	Store IdempotencyStore // default: memória
	TTL   time.Duration    // default: 24h
	// MaxBodySize limita o corpo de resposta cacheado (default 1 MiB);
	// respostas maiores não são cacheadas (a requisição segue normal).
	MaxBodySize int
}

// NewIdempotencyMiddleware honra o header Idempotency-Key em métodos mutantes:
// a primeira requisição executa e tem a resposta cacheada; replays com a mesma
// chave recebem a MESMA resposta (header Idempotency-Replayed: true); duplicatas
// concorrentes recebem 409 enquanto a original está em andamento.
//
// A chave de cache inclui a identidade (sistema/usuário), método e path — a mesma
// Idempotency-Key usada por consumidores diferentes não colide.
func NewIdempotencyMiddleware(cfg IdempotencyConfig) echo.MiddlewareFunc {
	if cfg.Store == nil {
		cfg.Store = NewMemoryIdempotencyStore()
	}
	if cfg.TTL <= 0 {
		cfg.TTL = 24 * time.Hour
	}
	if cfg.MaxBodySize <= 0 {
		cfg.MaxBodySize = 1 << 20
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			method := c.Request().Method
			if method != http.MethodPost && method != http.MethodPut && method != http.MethodPatch {
				return next(c)
			}
			idemKey := c.Request().Header.Get("Idempotency-Key")
			if idemKey == "" {
				return next(c)
			}

			key := cacheKey(c, idemKey)

			// Tentar reservar a chave (primeira requisição)
			reserved := cfg.Store.SetIfAbsent(key, &IdempotencyRecord{InProgress: true, StoredAt: time.Now()}, cfg.TTL)
			if !reserved {
				record, ok := cfg.Store.Get(key)
				if !ok {
					// Expirou entre o SetIfAbsent e o Get; tratar como nova
					return next(c)
				}
				if record.InProgress {
					return c.JSON(http.StatusConflict, map[string]string{
						"error": "A request with this Idempotency-Key is still in progress",
					})
				}
				// Replay da resposta original
				c.Response().Header().Set("Idempotency-Replayed", "true")
				if record.ContentType != "" {
					c.Response().Header().Set("Content-Type", record.ContentType)
				}
				c.Response().WriteHeader(record.StatusCode)
				_, _ = c.Response().Write(record.Body)
				return nil
			}

			// Executar capturando a resposta
			recorder := &responseRecorder{ResponseWriter: c.Response().Writer, limit: cfg.MaxBodySize}
			c.Response().Writer = recorder

			err := next(c)
			if err != nil {
				// Deixar o error handler do Echo agir; não cachear falhas de handler
				cfg.Store.Delete(key)
				return err
			}

			status := c.Response().Status
			if status >= 500 || recorder.truncated {
				// Não cachear erros de servidor nem respostas grandes demais
				cfg.Store.Delete(key)
				return nil
			}

			cfg.Store.Set(key, &IdempotencyRecord{
				StatusCode:  status,
				ContentType: c.Response().Header().Get("Content-Type"),
				Body:        recorder.body.Bytes(),
				StoredAt:    time.Now(),
			}, cfg.TTL)
			return nil
		}
	}
}

func cacheKey(c echo.Context, idemKey string) string {
	identity := "anon"
	if svc := GetServiceFromContext(c); svc != nil {
		identity = "svc:" + svc.ID
	} else if user := GetUserFromContext(c); user != nil {
		identity = "user:" + user.ID
	}
	sum := sha256.Sum256([]byte(identity + "|" + c.Request().Method + "|" + c.Request().URL.Path + "|" + idemKey))
	return "idem:" + hex.EncodeToString(sum[:])
}

// responseRecorder duplica a escrita da resposta para um buffer limitado
type responseRecorder struct {
	http.ResponseWriter
	body      bytes.Buffer
	limit     int
	truncated bool
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	if !r.truncated {
		if r.body.Len()+len(b) <= r.limit {
			r.body.Write(b)
		} else {
			r.truncated = true
			r.body.Reset()
		}
	}
	return r.ResponseWriter.Write(b)
}

// MemoryIdempotencyStore é a implementação in-memory de IdempotencyStore
type MemoryIdempotencyStore struct {
	mu      sync.Mutex
	records map[string]memoryIdemEntry
}

type memoryIdemEntry struct {
	record    *IdempotencyRecord
	expiresAt time.Time
}

// NewMemoryIdempotencyStore cria o store em memória (single-instance; usar Redis em HA)
func NewMemoryIdempotencyStore() *MemoryIdempotencyStore {
	return &MemoryIdempotencyStore{records: make(map[string]memoryIdemEntry)}
}

func (s *MemoryIdempotencyStore) Get(key string) (*IdempotencyRecord, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.records[key]
	if !ok || time.Now().After(entry.expiresAt) {
		delete(s.records, key)
		return nil, false
	}
	return entry.record, true
}

func (s *MemoryIdempotencyStore) SetIfAbsent(key string, record *IdempotencyRecord, ttl time.Duration) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if entry, ok := s.records[key]; ok && time.Now().Before(entry.expiresAt) {
		return false
	}
	s.records[key] = memoryIdemEntry{record: record, expiresAt: time.Now().Add(ttl)}
	return true
}

func (s *MemoryIdempotencyStore) Set(key string, record *IdempotencyRecord, ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records[key] = memoryIdemEntry{record: record, expiresAt: time.Now().Add(ttl)}
}

func (s *MemoryIdempotencyStore) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.records, key)
}
