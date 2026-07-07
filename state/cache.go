package state

import (
	"context"
	"time"
)

// Cache fornece operações de cache L1 (memória) e L2 (Redis).
type Cache struct {
	store Store
}

// NewCache cria um novo Cache.
func NewCache(store Store) *Cache {
	return &Cache{store: store}
}

// Set salva um valor no cache com TTL.
func (c *Cache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	ttlSeconds := int(ttl.Seconds())
	if ttlSeconds < 0 {
		ttlSeconds = 0 // Sem expiração
	}
	return c.store.CacheSet(ctx, key, value, ttlSeconds)
}

// Get recupera um valor do cache.
func (c *Cache) Get(ctx context.Context, key string, target interface{}) error {
	return c.store.CacheGet(ctx, key, target)
}

// Delete remove uma chave do cache.
func (c *Cache) Delete(ctx context.Context, key string) error {
	return c.store.CacheDelete(ctx, key)
}

// SetWithDefaultTTL salva um valor com TTL padrão (1 hora).
func (c *Cache) SetWithDefaultTTL(ctx context.Context, key string, value interface{}) error {
	return c.Set(ctx, key, value, time.Hour)
}

// GetOrSet recupera um valor do cache ou executa a função para gerar e armazenar.
func (c *Cache) GetOrSet(ctx context.Context, key string, target interface{}, ttl time.Duration, generator func() (interface{}, error)) error {
	// Tenta recuperar do cache
	err := c.Get(ctx, key, target)
	if err == nil {
		return nil // Cache hit
	}

	// Cache miss - gera o valor
	value, err := generator()
	if err != nil {
		return err
	}

	// Armazena no cache
	if err := c.Set(ctx, key, value, ttl); err != nil {
		return err
	}

	// Retorna o valor gerado
	return c.Get(ctx, key, target)
}
