package state

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisClient encapsula o cliente Redis do go-redis.
type RedisClient struct {
	client *redis.Client
}

// RawClient exposes the underlying go-redis client for advanced integrations.
// Use with care; prefer the typed helpers when possible.
func (r *RedisClient) RawClient() *redis.Client {
	return r.client
}

// NewRedisClient cria um novo cliente Redis.
func NewRedisClient(ctx context.Context, addr string) (*RedisClient, error) {   
        rdb := redis.NewClient(&redis.Options{
                Addr:     addr,
                Password: "", // Sem senha por padrão (configurável no futuro)  
		DB:       0,  // Database padrão
	})

	// Testa a conexão
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("erro ao conectar no Redis: %w", err)
	}

	return &RedisClient{client: rdb}, nil
}

// Close fecha a conexão com Redis.
func (r *RedisClient) Close() error {
	return r.client.Close()
}

// Set salva um valor no Redis com TTL.
func (r *RedisClient) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("erro ao serializar valor: %w", err)
	}

	return r.client.Set(ctx, key, data, ttl).Err()
}

// Get recupera um valor do Redis.
func (r *RedisClient) Get(ctx context.Context, key string, target interface{}) error {
	data, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return fmt.Errorf("cache miss: chave '%s' não encontrada", key)
	}
	if err != nil {
		return fmt.Errorf("erro ao recuperar do Redis: %w", err)
	}

	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("erro ao deserializar valor: %w", err)
	}

	return nil
}

// Delete remove uma chave do Redis.
func (r *RedisClient) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}

// Exists verifica se uma chave existe.
func (r *RedisClient) Exists(ctx context.Context, key string) (bool, error) {
	count, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// Expire define o TTL de uma chave existente.
func (r *RedisClient) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return r.client.Expire(ctx, key, ttl).Err()
}

// --- Operações de Hash (HSET/HGET) ---

// HSet salva um campo em um hash.
func (r *RedisClient) HSet(ctx context.Context, key, field string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("erro ao serializar valor: %w", err)
	}
	return r.client.HSet(ctx, key, field, data).Err()
}

// HGet recupera um campo de um hash.
func (r *RedisClient) HGet(ctx context.Context, key, field string, target interface{}) error {
	data, err := r.client.HGet(ctx, key, field).Bytes()
	if err == redis.Nil {
		return fmt.Errorf("campo '%s' não encontrado no hash '%s'", field, key)
	}
	if err != nil {
		return fmt.Errorf("erro ao recuperar campo do hash: %w", err)
	}

	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("erro ao deserializar valor: %w", err)
	}

	return nil
}

// HGetAll recupera todos os campos de um hash.
func (r *RedisClient) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return r.client.HGetAll(ctx, key).Result()
}

// HDel remove um ou mais campos de um hash.
func (r *RedisClient) HDel(ctx context.Context, key string, fields ...string) error {
	return r.client.HDel(ctx, key, fields...).Err()
}

// --- Operações de Sorted Set (ZSET) ---

// ZAdd adiciona um membro a um sorted set com score.
func (r *RedisClient) ZAdd(ctx context.Context, key string, score float64, member interface{}) error {
	memberStr, err := json.Marshal(member)
	if err != nil {
		return fmt.Errorf("erro ao serializar membro: %w", err)
	}
	return r.client.ZAdd(ctx, key, redis.Z{
		Score:  score,
		Member: string(memberStr),
	}).Err()
}

// ZRange recupera membros de um sorted set por range (com scores).
func (r *RedisClient) ZRange(ctx context.Context, key string, start, stop int64) ([]redis.Z, error) {
	return r.client.ZRangeWithScores(ctx, key, start, stop).Result()
}

// ZRem remove um ou mais membros de um sorted set.
func (r *RedisClient) ZRem(ctx context.Context, key string, members ...interface{}) error {
	memberStrs := make([]interface{}, len(members))
	for i, m := range members {
		data, err := json.Marshal(m)
		if err != nil {
			return fmt.Errorf("erro ao serializar membro: %w", err)
		}
		memberStrs[i] = string(data)
	}
	return r.client.ZRem(ctx, key, memberStrs...).Err()
}

// ZCard retorna o número de membros em um sorted set.
func (r *RedisClient) ZCard(ctx context.Context, key string) (int64, error) {
	return r.client.ZCard(ctx, key).Result()
}

// --- Operações de Script Lua ---

// Eval executa um script Lua no Redis (atômico).
func (r *RedisClient) Eval(ctx context.Context, script string, keys []string, args ...interface{}) (interface{}, error) {
	return r.client.Eval(ctx, script, keys, args...).Result()
}

// EvalSha executa um script Lua pelo SHA1 (mais eficiente após SCRIPT LOAD).
func (r *RedisClient) EvalSha(ctx context.Context, sha1 string, keys []string, args ...interface{}) (interface{}, error) {
	return r.client.EvalSha(ctx, sha1, keys, args...).Result()
}

// ScriptLoad carrega um script Lua no Redis e retorna o SHA1.
func (r *RedisClient) ScriptLoad(ctx context.Context, script string) (string, error) {
	return r.client.ScriptLoad(ctx, script).Result()
}
