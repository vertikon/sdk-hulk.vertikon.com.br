package state

import (
	"context"

	"gorm.io/gorm"
)

// Store abstrai o acesso a dados (SQL/NoSQL) e Cache.
type Store interface {
	// Exec executa comandos de escrita (Insert/Update/Delete).
	Exec(ctx context.Context, query string, args ...interface{}) error

	// QueryRow busca um único registro.
	QueryRow(ctx context.Context, query string, args ...interface{}) RowScanner

	// Query busca múltiplos registros.
	Query(ctx context.Context, query string, args ...interface{}) (Rows, error)

	// BeginTx inicia uma transação.
	BeginTx(ctx context.Context) (Tx, error)

	// CacheSet salva algo no Redis/Memória.
	CacheSet(ctx context.Context, key string, value interface{}, ttlSeconds int) error

	// CacheGet recupera algo do Redis/Memória.
	CacheGet(ctx context.Context, key string, target interface{}) error

	// CacheDelete remove uma chave do cache.
	CacheDelete(ctx context.Context, key string) error

	// DB retorna a instância do GORM (use com cuidado, quebra a abstração).
	DB() *gorm.DB
}

// RowScanner abstrai o sql.Row do Go.
type RowScanner interface {
	Scan(dest ...interface{}) error
}

// Rows abstrai o sql.Rows do Go.
type Rows interface {
	Next() bool
	Scan(dest ...interface{}) error
	Close() error
	Err() error
}

// Tx representa uma transação de banco de dados.
type Tx interface {
	Exec(ctx context.Context, query string, args ...interface{}) error
	QueryRow(ctx context.Context, query string, args ...interface{}) RowScanner
	Query(ctx context.Context, query string, args ...interface{}) (Rows, error)
	Commit() error
	Rollback() error
}
