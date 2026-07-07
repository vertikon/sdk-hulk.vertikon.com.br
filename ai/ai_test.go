package ai

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
)

// MockClient implementa Client para testes
type MockClient struct {
	responses map[string]string
	calls     []string
	mu        sync.RWMutex
}

func NewMockClient() *MockClient {
	return &MockClient{
		responses: make(map[string]string),
		calls:     make([]string, 0),
	}
}

func (m *MockClient) SetResponse(prompt string, response string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.responses[prompt] = response
}

func (m *MockClient) Chat(ctx context.Context, prompt string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.calls = append(m.calls, prompt)

	if response, ok := m.responses[prompt]; ok {
		return response, nil
	}

	return "Mock response for: " + prompt, nil
}

func (m *MockClient) ChatWithContext(ctx context.Context, messages []ChatMessage) (string, error) {
	return "Mock context response", nil
}

func (m *MockClient) AnalyzeError(ctx context.Context, context string, err error) (string, error) {
	if err == nil {
		return "", errors.New("no error to analyze")
	}

	prompt := "Analyze error: " + context + " - " + err.Error()
	return m.Chat(ctx, prompt)
}

func (m *MockClient) Vision(ctx context.Context, imageURL string, prompt string) (*AnalysisResult, error) {
	return &AnalysisResult{
		Content: "Mock vision analysis",
		Data:    map[string]interface{}{"detected": "object"},
	}, nil
}

func (m *MockClient) Embeddings(ctx context.Context, text string) ([]float32, error) {
	return []float32{0.1, 0.2, 0.3}, nil
}

func (m *MockClient) BatchEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	result := make([][]float32, len(texts))
	for i := range texts {
		result[i] = []float32{0.1, 0.2, 0.3}
	}
	return result, nil
}

func (m *MockClient) GetCalls() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.calls
}

// Testes

func TestMockClient_Chat(t *testing.T) {
	client := NewMockClient()
	ctx := context.Background()

	prompt := "What is 2+2?"
	expectedResponse := "4"

	client.SetResponse(prompt, expectedResponse)

	response, err := client.Chat(ctx, prompt)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	if response != expectedResponse {
		t.Errorf("Expected '%s', got '%s'", expectedResponse, response)
	}
}

func TestMockClient_ChatWithContext(t *testing.T) {
	client := NewMockClient()
	ctx := context.Background()

	messages := []ChatMessage{
		{Role: "user", Content: "Hello"},
	}

	response, err := client.ChatWithContext(ctx, messages)
	if err != nil {
		t.Fatalf("ChatWithContext failed: %v", err)
	}

	if response == "" {
		t.Error("Expected non-empty response")
	}
}

func TestMockClient_AnalyzeError(t *testing.T) {
	client := NewMockClient()
	ctx := context.Background()

	testErr := errors.New("database connection failed")

	analysis, err := client.AnalyzeError(ctx, "Database operation", testErr)
	if err != nil {
		t.Fatalf("AnalyzeError failed: %v", err)
	}

	if !strings.Contains(analysis, "Mock response") {
		t.Errorf("Expected mock response, got '%s'", analysis)
	}
}

func TestMockClient_Vision(t *testing.T) {
	client := NewMockClient()
	ctx := context.Background()

	result, err := client.Vision(ctx, "http://example.com/image.jpg", "Describe")
	if err != nil {
		t.Fatalf("Vision failed: %v", err)
	}

	if result == nil || result.Content == "" {
		t.Error("Expected valid result")
	}
}

func TestMockClient_Embeddings(t *testing.T) {
	client := NewMockClient()
	ctx := context.Background()

	embedding, err := client.Embeddings(ctx, "test text")
	if err != nil {
		t.Fatalf("Embeddings failed: %v", err)
	}

	if len(embedding) == 0 {
		t.Error("Expected non-empty embedding")
	}
}

func TestMockClient_BatchEmbeddings(t *testing.T) {
	client := NewMockClient()
	ctx := context.Background()

	texts := []string{"text1", "text2", "text3"}

	embeddings, err := client.BatchEmbeddings(ctx, texts)
	if err != nil {
		t.Fatalf("BatchEmbeddings failed: %v", err)
	}

	if len(embeddings) != len(texts) {
		t.Errorf("Expected %d embeddings, got %d", len(texts), len(embeddings))
	}
}

func TestClient_Interface(t *testing.T) {
	var _ Client = (*MockClient)(nil)
}

func TestDefaultCompletionOptions(t *testing.T) {
	opts := DefaultCompletionOptions()

	if opts == nil {
		t.Fatal("DefaultCompletionOptions returned nil")
	}

	if opts.Temperature != 1.0 {
		t.Errorf("Expected Temperature 1.0, got %f", opts.Temperature)
	}

	if opts.MaxTokens != 1000 {
		t.Errorf("Expected MaxTokens 1000, got %d", opts.MaxTokens)
	}

	if opts.TopP != 1.0 {
		t.Errorf("Expected TopP 1.0, got %f", opts.TopP)
	}

	if opts.StopSequences != nil {
		t.Error("Expected StopSequences to be nil")
	}
}

func TestCompletionOptions_CustomValues(t *testing.T) {
	opts := &CompletionOptions{
		Temperature:   0.7,
		MaxTokens:     500,
		TopP:          0.9,
		StopSequences: []string{"\n", "END"},
	}

	if opts.Temperature != 0.7 {
		t.Errorf("Expected Temperature 0.7, got %f", opts.Temperature)
	}

	if opts.MaxTokens != 500 {
		t.Errorf("Expected MaxTokens 500, got %d", opts.MaxTokens)
	}

	if len(opts.StopSequences) != 2 {
		t.Errorf("Expected 2 StopSequences, got %d", len(opts.StopSequences))
	}
}
