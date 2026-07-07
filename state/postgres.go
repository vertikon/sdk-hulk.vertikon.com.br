package state

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// IsNoRows informa se o erro é "nenhuma linha encontrada". PEGADINHA: QueryRow
// devolve o pgxpool direto, e o Scan retorna pgx.ErrNoRows (NÃO sql.ErrNoRows) —
// comparar só com sql.ErrNoRows faz o not-found virar 500. Use este helper.
func IsNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sql.ErrNoRows)
}

type PostgresStore struct {
	pool     *pgxpool.Pool
	db       *gorm.DB     // GORM instance
	redis    *RedisClient // Cliente Redis para cache
	useCache bool         // Flag para habilitar/desabilitar cache
}

// NewPostgresStore cria um novo PostgresStore sem Redis (cache desabilitado).
func NewPostgresStore(ctx context.Context, dsn string) (*PostgresStore, error) {
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("erro ao parsear DSN: %w", err)
	}

	// Configurações de pool (podem vir do config no futuro)
	config.MaxConns = 25
	config.MinConns = 2
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar pool de conexões: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("erro ao conectar no banco: %w", err)
	}

	// Initialize GORM
	gormDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("erro ao inicializar GORM: %w", err)
	}

	return &PostgresStore{
		pool:     pool,
		db:       gormDB,
		redis:    nil,
		useCache: false,
	}, nil
}

// NewPostgresStoreWithRedis cria um novo PostgresStore com Redis habilitado.
func NewPostgresStoreWithRedis(ctx context.Context, dsn string, redisAddr string) (*PostgresStore, error) {
	store, err := NewPostgresStore(ctx, dsn)
	if err != nil {
		return nil, err
	}

	redisClient, err := NewRedisClient(ctx, redisAddr)
	if err != nil {
		// Se Redis falhar, continua sem cache (graceful degradation)
		return store, nil
	}

	store.redis = redisClient
	store.useCache = true
	return store, nil
}

func (s *PostgresStore) Close() {
	s.pool.Close()
	if s.redis != nil {
		s.redis.Close()
	}
}

// Exec executa comandos de escrita (Insert/Update/Delete).
func (s *PostgresStore) Exec(ctx context.Context, query string, args ...interface{}) error {
	_, err := s.pool.Exec(ctx, query, args...)
	return err
}

// QueryRow busca um único registro.
func (s *PostgresStore) QueryRow(ctx context.Context, query string, args ...interface{}) RowScanner {
	return s.pool.QueryRow(ctx, query, args...)
}

// Query busca múltiplos registros.
func (s *PostgresStore) Query(ctx context.Context, query string, args ...interface{}) (Rows, error) {
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &PostgresRows{rows: rows}, nil
}

// BeginTx inicia uma transação.
func (s *PostgresStore) BeginTx(ctx context.Context) (Tx, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return &PostgresTx{tx: tx}, nil
}

// --- Cache Implementation (Redis) ---

func (s *PostgresStore) CacheSet(ctx context.Context, key string, value interface{}, ttlSeconds int) error {
	if !s.useCache || s.redis == nil {
		return nil // Cache desabilitado ou Redis não disponível
	}

	ttl := time.Duration(ttlSeconds) * time.Second
	if ttlSeconds <= 0 {
		ttl = 0 // Sem expiração
	}

	return s.redis.Set(ctx, key, value, ttl)
}

func (s *PostgresStore) CacheGet(ctx context.Context, key string, target interface{}) error {
	if !s.useCache || s.redis == nil {
		return fmt.Errorf("cache miss") // Cache desabilitado
	}

	return s.redis.Get(ctx, key, target)
}

func (s *PostgresStore) CacheDelete(ctx context.Context, key string) error {
	if !s.useCache || s.redis == nil {
		return nil // Cache desabilitado
	}

	return s.redis.Delete(ctx, key)
}

// RedisClient retorna o cliente Redis (para uso direto em módulos que precisam de operações avançadas).
func (s *PostgresStore) RedisClient() *RedisClient {
	return s.redis
}

// DB retorna a instância do GORM.
func (s *PostgresStore) DB() *gorm.DB {
	return s.db
}

// --- Transaction Wrapper ---

type PostgresTx struct {
	tx pgx.Tx
}

func (t *PostgresTx) Exec(ctx context.Context, query string, args ...interface{}) error {
	_, err := t.tx.Exec(ctx, query, args...)
	return err
}

func (t *PostgresTx) QueryRow(ctx context.Context, query string, args ...interface{}) RowScanner {
	return t.tx.QueryRow(ctx, query, args...)
}

func (t *PostgresTx) Query(ctx context.Context, query string, args ...interface{}) (Rows, error) {
	rows, err := t.tx.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &PostgresRows{rows: rows}, nil
}

func (t *PostgresTx) Commit() error {
	return t.tx.Commit(context.Background())
}

func (t *PostgresTx) Rollback() error {
	return t.tx.Rollback(context.Background())
}

// --- Rows Wrapper ---

type PostgresRows struct {
	rows pgx.Rows
}

func (r *PostgresRows) Next() bool {
	return r.rows.Next()
}

func (r *PostgresRows) Scan(dest ...interface{}) error {
	return r.rows.Scan(dest...)
}

func (r *PostgresRows) Close() error {
	r.rows.Close()
	return nil
}

func (r *PostgresRows) Err() error {
	return r.rows.Err()
}
