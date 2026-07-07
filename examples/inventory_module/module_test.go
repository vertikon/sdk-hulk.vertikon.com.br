package inventory_module

import (
	"gorm.io/gorm"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vertikon/sdk-hulk.vertikon.com.br"
	"github.com/vertikon/sdk-hulk.vertikon.com.br/ai"
	"github.com/vertikon/sdk-hulk.vertikon.com.br/events"
	"github.com/vertikon/sdk-hulk.vertikon.com.br/state"
	"go.uber.org/zap"
)

func TestNewModule(t *testing.T) {
	mod := New()
	assert.NotNil(t, mod)
}

func TestInventoryModule_Config(t *testing.T) {
	mod := New()
	config := mod.Config()

	assert.Equal(t, "bloco-1-inventory", config.ID)
	assert.Equal(t, "Core Inventory & Fulfillment", config.Name)
	assert.Equal(t, "v1.0.0", config.Version)
	assert.Contains(t, config.Dependencies, "bloco-15-mdm")
}

func TestInventoryModule_Init(t *testing.T) {
	mod := New()

	// Mock store for Init
	store := &MockStore{}
	logger, _ := zap.NewDevelopment()
	ctx := hulk.NewContext(context.Background(), logger, nil, store, nil, nil, nil)

	err := mod.Init(ctx)
	assert.NoError(t, err)
}

type MockBus struct{}

func (m *MockBus) Publish(topic string, payload interface{}) error                  { return nil }
func (m *MockBus) Subscribe(topic string, handler events.Handler) error             { return nil }
func (m *MockBus) QueueSubscribe(topic, queue string, handler events.Handler) error { return nil }

type MockStore struct{}

func (m *MockStore) DB() *gorm.DB { return nil }

func (m *MockStore) CacheSet(ctx context.Context, key string, value interface{}, ttl int) error {
	return nil
}
func (m *MockStore) CacheGet(ctx context.Context, key string, target interface{}) error { return nil }
func (m *MockStore) CacheDelete(ctx context.Context, key string) error                  { return nil }
func (m *MockStore) QueryRow(ctx context.Context, query string, args ...interface{}) state.RowScanner {
	return nil
}
func (m *MockStore) Query(ctx context.Context, query string, args ...interface{}) (state.Rows, error) {
	return nil, nil
}
func (m *MockStore) Exec(ctx context.Context, query string, args ...interface{}) error { return nil }
func (m *MockStore) BeginTx(ctx context.Context) (state.Tx, error)                     { return nil, nil }

type MockAI struct{}

func (m *MockAI) Chat(ctx context.Context, prompt string) (string, error) { return "4", nil }
func (m *MockAI) ChatWithContext(ctx context.Context, messages []ai.ChatMessage) (string, error) {
	return "", nil
}
func (m *MockAI) AnalyzeError(ctx context.Context, context string, err error) (string, error) {
	return "", nil
}
func (m *MockAI) Vision(ctx context.Context, prompt string, imageURL string) (*ai.AnalysisResult, error) {
	return nil, nil
}
func (m *MockAI) Embeddings(ctx context.Context, text string) ([]float32, error) { return nil, nil }
func (m *MockAI) BatchEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	return nil, nil
}

func TestInventoryModule_Start(t *testing.T) {
	mod := New()

	store := &MockStore{}
	logger, _ := zap.NewDevelopment()
	ctx := hulk.NewContext(context.Background(), logger, &MockBus{}, store, &MockAI{}, nil, nil)

	// Init first to setup ledgerService
	mod.Init(ctx)

	err := mod.Start(ctx)
	assert.NoError(t, err)
}

func TestInventoryModule_Stop(t *testing.T) {
	mod := New()
	err := mod.Stop(context.Background())
	assert.NoError(t, err)
}
