package state

import (
	"context"
	"errors"
)

// Repository fornece operações CRUD genéricas sobre o Store.
type Repository struct {
	store Store
}

// NewRepository cria um novo Repository.
func NewRepository(store Store) *Repository {
	return &Repository{store: store}
}

// Create executa um INSERT e retorna o ID gerado (se aplicável).
func (r *Repository) Create(ctx context.Context, query string, args ...interface{}) (interface{}, error) {
	if err := r.store.Exec(ctx, query, args...); err != nil {
		return nil, err
	}
	// Nota: A implementação real deve extrair o ID retornado do banco
	// Isso é uma simplificação - em produção, use RETURNING ou LastInsertId()
	return nil, nil
}

// Update executa um UPDATE.
func (r *Repository) Update(ctx context.Context, query string, args ...interface{}) error {
	return r.store.Exec(ctx, query, args...)
}

// Delete executa um DELETE.
func (r *Repository) Delete(ctx context.Context, query string, args ...interface{}) error {
	return r.store.Exec(ctx, query, args...)
}

// FindOne busca um único registro.
func (r *Repository) FindOne(ctx context.Context, query string, args ...interface{}) (RowScanner, error) {
	row := r.store.QueryRow(ctx, query, args...)
	if row == nil {
		return nil, errors.New("no rows found")
	}
	return row, nil
}

// FindMany busca múltiplos registros.
func (r *Repository) FindMany(ctx context.Context, query string, args ...interface{}) (Rows, error) {
	return r.store.Query(ctx, query, args...)
}

// ExecuteInTx executa uma função dentro de uma transação.
func (r *Repository) ExecuteInTx(ctx context.Context, fn func(Tx) error) error {
	tx, err := r.store.BeginTx(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}
