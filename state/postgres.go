package state

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
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
	jobsPool *pgxpool.Pool // pool OWNER p/ workers sem-tenant (bypassa RLS); nil se JOBS_DATABASE_URL ausente
	migPool  *pgxpool.Pool // pool OWNER da FASE DE BOOT (self-migrations/seeds); nil se MIGRATIONS_DATABASE_URL ausente
	booting  atomic.Bool   // true da criação até EndBoot(): roteia tudo pelo migPool (semântica pré-flip)
	db       *gorm.DB      // GORM instance
	redis    *RedisClient  // Cliente Redis para cache
	useCache bool          // Flag para habilitar/desabilitar cache
}

// bootPool devolve o pool de boot enquanto a fase de boot durar (e ele existir).
// Sob o flip RLS o runtime conecta como app_login, que NÃO pode CREATE/ALTER —
// as self-migrations dos módulos (Init) e seeds precisam do owner, exatamente
// como era antes do flip. Gated por MIGRATIONS_DATABASE_URL: ausente = nil =
// comportamento antigo (tudo no pool principal).
func (s *PostgresStore) bootPool() *pgxpool.Pool {
	if s.migPool != nil && s.booting.Load() {
		return s.migPool
	}
	return nil
}

// EndBoot encerra a fase de boot: o store volta ao pool principal (app_login,
// RLS enforçada) e o pool de migrations é fechado. Chamado pelo app.Run após o
// Start de todos os módulos, ANTES de o HTTP começar a servir. Idempotente.
// Nota: "restart de módulo" via /system re-roda Init depois do boot — a
// migration idempotente falha com WARN (objetos já existem), como antes.
func (s *PostgresStore) EndBoot() {
	if s.booting.Swap(false) && s.migPool != nil {
		s.migPool.Close()
	}
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

	// Pool de JOBS (flip RLS): quando o serviço roda como app_login (não-owner, RLS
	// enforçada), os workers cross-tenant/sem-tenant precisam da conexão do OWNER
	// (bypassa RLS por ownership). GATED por JOBS_DATABASE_URL — ausente = nil = os
	// métodos usam s.pool (comportamento atual, deploy sem risco).
	var jobsPool *pgxpool.Pool
	if jobsDSN := strings.TrimSpace(os.Getenv("JOBS_DATABASE_URL")); jobsDSN != "" {
		jc, err := pgxpool.ParseConfig(jobsDSN)
		if err != nil {
			return nil, fmt.Errorf("erro ao parsear JOBS_DATABASE_URL: %w", err)
		}
		jc.MaxConns = 10
		// MinConns 0 e idle ABAIXO do auto-suspend do Neon (medido em 300 s, 2026-09-12),
		// que antes eram 1 e o padrão do pgxpool (30 min). Os dois têm de andar juntos:
		//
		//  - idle 30 min > suspend 5 min fazia o pool entregar conexão que o servidor já
		//    tinha fechado; quem fosse usar descobria na hora e estourava. É a causa das
		//    falhas de `dial tcp :5432` do poller desta fila (F-13).
		//  - mas baixar o idle mantendo MinConns 1 seria pior: o pool reconectaria a cada
		//    4 min para repor o mínimo, o compute NUNCA suspenderia e a fatura voltaria ao
		//    salto de US$ 14 → US$ 150 documentado no runQueueWorker do wa-dispatch-consumer.
		//
		// Com 0 o pool drena até zero quando ocioso, o compute suspende e a economia fica de
		// pé. O custo é o wake (medido: ~2,7 s) na primeira consulta depois da ociosidade —
		// que este worker absorve, porque chama com context.Background(), sem deadline.
		jc.MinConns = 0
		jc.MaxConnIdleTime = 4 * time.Minute
		jc.MaxConnLifetime = time.Hour
		jobsPool, err = pgxpool.NewWithConfig(ctx, jc)
		if err != nil {
			return nil, fmt.Errorf("erro ao criar pool de jobs: %w", err)
		}
		if err := jobsPool.Ping(ctx); err != nil {
			return nil, fmt.Errorf("erro ao conectar no pool de jobs: %w", err)
		}
	}

	// Pool de BOOT (flip RLS): self-migrations/seeds dos módulos rodam no Init
	// pelo store principal — sob app_login o CREATE/ALTER falharia. Com
	// MIGRATIONS_DATABASE_URL (owner) setado, a fase de boot inteira roda no
	// owner (igual ao pré-flip) e EndBoot() devolve o runtime ao app_login.
	var migPool *pgxpool.Pool
	if migDSN := strings.TrimSpace(os.Getenv("MIGRATIONS_DATABASE_URL")); migDSN != "" {
		mc, err := pgxpool.ParseConfig(migDSN)
		if err != nil {
			return nil, fmt.Errorf("erro ao parsear MIGRATIONS_DATABASE_URL: %w", err)
		}
		mc.MaxConns = 4
		mc.MinConns = 0
		mc.MaxConnLifetime = time.Hour
		migPool, err = pgxpool.NewWithConfig(ctx, mc)
		if err != nil {
			return nil, fmt.Errorf("erro ao criar pool de migrations: %w", err)
		}
		if err := migPool.Ping(ctx); err != nil {
			return nil, fmt.Errorf("erro ao conectar no pool de migrations: %w", err)
		}
	}

	s := &PostgresStore{
		pool:     pool,
		jobsPool: jobsPool,
		migPool:  migPool,
		db:       gormDB,
		redis:    nil,
		useCache: false,
	}
	if migPool != nil {
		s.booting.Store(true)
	}
	return s, nil
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
	if s.jobsPool != nil {
		s.jobsPool.Close()
	}
	if s.booting.Swap(false) && s.migPool != nil { // EndBoot nunca chamado → fecha aqui
		s.migPool.Close()
	}
	if s.redis != nil {
		s.redis.Close()
	}
}

// Exec executa comandos de escrita (Insert/Update/Delete).
// Com HULK_RLS_CTX=1 e tenant no contexto, roda com app.tenant_id setado (RLS).
func (s *PostgresStore) Exec(ctx context.Context, query string, args ...interface{}) error {
	if p := s.bootPool(); p != nil { // fase de boot → owner (migrations/seeds; pré-flip)
		_, err := p.Exec(ctx, query, args...)
		return err
	}
	if s.jobsPool != nil && jobsOf(ctx) { // worker sem-tenant → pool do owner (bypassa RLS)
		_, err := s.jobsPool.Exec(ctx, query, args...)
		return err
	}
	if tid, ok := tenantOf(ctx); ok {
		conn, cleanup, err := s.acquireWithTenant(ctx, tid)
		if err != nil {
			return err
		}
		_, err = conn.Exec(ctx, query, args...)
		cleanup()
		return err
	}
	_, err := s.pool.Exec(ctx, query, args...)
	return err
}

// QueryRow busca um único registro.
func (s *PostgresStore) QueryRow(ctx context.Context, query string, args ...interface{}) RowScanner {
	if p := s.bootPool(); p != nil {
		return p.QueryRow(ctx, query, args...)
	}
	if s.jobsPool != nil && jobsOf(ctx) {
		return s.jobsPool.QueryRow(ctx, query, args...)
	}
	if tid, ok := tenantOf(ctx); ok {
		conn, cleanup, err := s.acquireWithTenant(ctx, tid)
		if err != nil {
			return errRow{err}
		}
		return tenantRow{row: conn.QueryRow(ctx, query, args...), cleanup: cleanup}
	}
	return s.pool.QueryRow(ctx, query, args...)
}

// Query busca múltiplos registros.
func (s *PostgresStore) Query(ctx context.Context, query string, args ...interface{}) (Rows, error) {
	if p := s.bootPool(); p != nil {
		rows, err := p.Query(ctx, query, args...)
		if err != nil {
			return nil, err
		}
		return &PostgresRows{rows: rows}, nil
	}
	if s.jobsPool != nil && jobsOf(ctx) {
		rows, err := s.jobsPool.Query(ctx, query, args...)
		if err != nil {
			return nil, err
		}
		return &PostgresRows{rows: rows}, nil
	}
	if tid, ok := tenantOf(ctx); ok {
		conn, cleanup, err := s.acquireWithTenant(ctx, tid)
		if err != nil {
			return nil, err
		}
		rows, qerr := conn.Query(ctx, query, args...)
		if qerr != nil {
			cleanup()
			return nil, qerr
		}
		return &PostgresRows{rows: rows, cleanup: cleanup}, nil
	}
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &PostgresRows{rows: rows}, nil
}

// acquireWithTenant reserva uma conexão do pool com app.tenant_id setado.
// O cleanup RESETA a variável antes de devolver ao pool (nunca vazar tenant
// entre requests); se o reset falhar, a conexão é destruída, não reciclada.
func (s *PostgresStore) acquireWithTenant(ctx context.Context, tenantID string) (*pgxpool.Conn, func(), error) {
	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		return nil, nil, err
	}
	if _, err := conn.Exec(ctx, "SELECT set_config('app.tenant_id', $1, false)", tenantID); err != nil {
		conn.Release()
		return nil, nil, fmt.Errorf("set tenant context: %w", err)
	}
	cleanup := func() {
		if _, rerr := conn.Exec(context.Background(), "SELECT set_config('app.tenant_id', '', false)"); rerr != nil {
			_ = conn.Conn().Close(context.Background()) // não recicla conexão contaminada
		}
		conn.Release()
	}
	return conn, cleanup, nil
}

// tenantRow embrulha o Row para resetar/devolver a conexão após o Scan.
type tenantRow struct {
	row     pgx.Row
	cleanup func()
}

func (r tenantRow) Scan(dest ...interface{}) error {
	err := r.row.Scan(dest...)
	r.cleanup()
	return err
}

// errRow propaga erro de aquisição pela interface RowScanner.
type errRow struct{ err error }

func (r errRow) Scan(...interface{}) error { return r.err }

// BeginTx inicia uma transação.
func (s *PostgresStore) BeginTx(ctx context.Context) (Tx, error) {
	if p := s.bootPool(); p != nil {
		tx, err := p.Begin(ctx)
		if err != nil {
			return nil, err
		}
		return &PostgresTx{tx: tx}, nil
	}
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
	rows    pgx.Rows
	cleanup func() // devolve/limpa a conexão de tenant (nil no caminho comum)
}

func (r *PostgresRows) Next() bool {
	return r.rows.Next()
}

func (r *PostgresRows) Scan(dest ...interface{}) error {
	return r.rows.Scan(dest...)
}

func (r *PostgresRows) Close() error {
	r.rows.Close()
	if r.cleanup != nil {
		r.cleanup()
		r.cleanup = nil
	}
	return nil
}

func (r *PostgresRows) Err() error {
	return r.rows.Err()
}
