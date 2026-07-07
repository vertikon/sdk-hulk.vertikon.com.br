package ai

import (
	"context"
)

// LLMClient define operações específicas de Language Models.
type LLMClient interface {
	// Completion gera uma resposta completa baseada no prompt.
	Completion(ctx context.Context, prompt string, options *CompletionOptions) (string, error)

	// StreamCompletion retorna um canal para streaming de respostas.
	StreamCompletion(ctx context.Context, prompt string, options *CompletionOptions) (<-chan string, error)
}

// CompletionOptions configura opções para geração de texto.
type CompletionOptions struct {
	Temperature   float64  // Controle de criatividade (0.0-2.0)
	MaxTokens     int      // Número máximo de tokens na resposta
	TopP          float64  // Nucleus sampling
	StopSequences []string // Sequências que param a geração
}

// DefaultCompletionOptions retorna opções padrão.
func DefaultCompletionOptions() *CompletionOptions {
	return &CompletionOptions{
		Temperature:   1.0,
		MaxTokens:     1000,
		TopP:          1.0,
		StopSequences: nil,
	}
}
