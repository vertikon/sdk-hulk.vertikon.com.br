package ai

import "context"

// AnalysisResult é a resposta padronizada da IA.
type AnalysisResult struct {
	Content string                 // Resposta textual da IA
	Data    map[string]interface{} // Dados estruturados extraídos (JSON)
}

// Client define o acesso às capacidades cognitivas.
type Client interface {
	// Chat envia um prompt simples para o LLM padrão (ex: GPT-4 ou Claude).
	Chat(ctx context.Context, prompt string) (string, error)

	// ChatWithContext envia um prompt com contexto de conversação.
	ChatWithContext(ctx context.Context, messages []ChatMessage) (string, error)

	// AnalyzeError usa IA para diagnosticar erros técnicos.
	AnalyzeError(ctx context.Context, context string, err error) (string, error)

	// Vision processa imagens (ex: fotos de roupas) e retorna atributos.
	Vision(ctx context.Context, imageURL string, prompt string) (*AnalysisResult, error)

	// Embeddings gera vetores para busca semântica.
	Embeddings(ctx context.Context, text string) ([]float32, error)

	// BatchEmbeddings gera embeddings para múltiplos textos de uma vez.
	BatchEmbeddings(ctx context.Context, texts []string) ([][]float32, error)
}

// ChatMessage representa uma mensagem na conversa com o LLM.
type ChatMessage struct {
	Role    string // "system", "user", "assistant"
	Content string
}
