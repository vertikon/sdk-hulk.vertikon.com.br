package http

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"golang.org/x/time/rate"
)

// RateLimitConfig configura o middleware de rate limiting (Fase 0.3)
type RateLimitConfig struct {
	// RPS é a taxa sustentada de requisições por segundo por chave
	RPS float64
	// Burst é o pico tolerado por chave
	Burst int
	// KeyFunc deriva a chave de limitação; default: credencial M2M → usuário → IP
	KeyFunc func(c echo.Context) string
	// TTL de entradas ociosas no mapa de limiters (default 10 min)
	IdleTTL time.Duration
}

type limiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// NewRateLimitMiddleware cria um middleware token-bucket por chave (in-memory).
// Responde 429 com Retry-After quando o limite é excedido e expõe X-RateLimit-*.
// Para HA multi-instância, evoluir o backend para Redis mantendo esta interface.
func NewRateLimitMiddleware(cfg RateLimitConfig) echo.MiddlewareFunc {
	if cfg.RPS <= 0 {
		cfg.RPS = 10
	}
	if cfg.Burst <= 0 {
		cfg.Burst = int(cfg.RPS) * 2
	}
	if cfg.KeyFunc == nil {
		cfg.KeyFunc = defaultRateLimitKey
	}
	if cfg.IdleTTL <= 0 {
		cfg.IdleTTL = 10 * time.Minute
	}

	var mu sync.Mutex
	limiters := make(map[string]*limiterEntry)
	lastSweep := time.Now()

	getLimiter := func(key string) *rate.Limiter {
		mu.Lock()
		defer mu.Unlock()

		now := time.Now()
		// Limpeza periódica de chaves ociosas (evita crescimento sem limite)
		if now.Sub(lastSweep) > cfg.IdleTTL {
			for k, entry := range limiters {
				if now.Sub(entry.lastSeen) > cfg.IdleTTL {
					delete(limiters, k)
				}
			}
			lastSweep = now
		}

		entry, ok := limiters[key]
		if !ok {
			entry = &limiterEntry{limiter: rate.NewLimiter(rate.Limit(cfg.RPS), cfg.Burst)}
			limiters[key] = entry
		}
		entry.lastSeen = now
		return entry.limiter
	}

	limitHeader := strconv.Itoa(cfg.Burst)
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			limiter := getLimiter(cfg.KeyFunc(c))

			c.Response().Header().Set("X-RateLimit-Limit", limitHeader)
			tokens := int(limiter.Tokens())
			if tokens < 0 {
				tokens = 0
			}
			c.Response().Header().Set("X-RateLimit-Remaining", strconv.Itoa(tokens))

			if !limiter.Allow() {
				// Tempo aproximado até liberar 1 token
				retryAfter := int(1 / cfg.RPS)
				if retryAfter < 1 {
					retryAfter = 1
				}
				c.Response().Header().Set("Retry-After", strconv.Itoa(retryAfter))
				return c.JSON(http.StatusTooManyRequests, map[string]string{"error": "Rate limit exceeded"})
			}
			return next(c)
		}
	}
}

// defaultRateLimitKey deriva a chave: sistema M2M → usuário JWT → IP de origem
func defaultRateLimitKey(c echo.Context) string {
	if svc := GetServiceFromContext(c); svc != nil {
		return "svc:" + svc.ID
	}
	if user := GetUserFromContext(c); user != nil {
		return "user:" + user.ID
	}
	return "ip:" + c.RealIP()
}
