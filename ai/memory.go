package ai

import (
	"context"
)

// MemoryClient define operações de memória episódica e semântica.
type MemoryClient interface {
	// StoreEpisodic armazena uma memória episódica (evento com contexto temporal).
	StoreEpisodic(ctx context.Context, moduleID string, event string, metadata map[string]interface{}) error

	// RetrieveEpisodic recupera memórias episódicas baseadas em critérios.
	RetrieveEpisodic(ctx context.Context, moduleID string, filters *EpisodicFilters) ([]EpisodicMemory, error)

	// StoreSemantic armazena uma memória semântica (conhecimento estruturado).
	StoreSemantic(ctx context.Context, moduleID string, knowledge string, embedding []float32) error

	// SearchSemantic busca memórias semânticas usando similaridade vetorial.
	SearchSemantic(ctx context.Context, moduleID string, queryEmbedding []float32, topK int) ([]SemanticMemory, error)
}

// EpisodicMemory representa uma memória episódica.
type EpisodicMemory struct {
	ID        string
	ModuleID  string
	Event     string
	Metadata  map[string]interface{}
	Timestamp int64
}

// SemanticMemory representa uma memória semântica.
type SemanticMemory struct {
	ID        string
	ModuleID  string
	Knowledge string
	Embedding []float32
	Score     float64 // Similaridade com a query
}

// EpisodicFilters define filtros para busca de memórias episódicas.
type EpisodicFilters struct {
	EventPattern string            // Padrão de busca no evento
	Metadata     map[string]string // Filtros por metadados
	Since        int64             // Timestamp mínimo
	Until        int64             // Timestamp máximo
	Limit        int               // Limite de resultados
}
