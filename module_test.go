package hulk_test

import (
	"gorm.io/gorm"
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vertikon/sdk-hulk.vertikon.com.br"
	"github.com/vertikon/sdk-hulk.vertikon.com.br/ai"
	"github.com/vertikon/sdk-hulk.vertikon.com.br/events"
	"github.com/vertikon/sdk-hulk.vertikon.com.br/state"
	"go.uber.org/zap"
)

// mockModule é uma implementação de teste do Module.
type mockModule struct {
	config      hulk.ModuleConfig
	initCalled  bool
	startCalled bool
	stopCalled  bool
	initError   error
	startError  error
	stopError   error
}

func (m *mockModule) Config() hulk.ModuleConfig {
	return m.config
}

func (m *mockModule) Init(ctx hulk.Context) error {
	m.initCalled = true
	return m.initError
}

func (m *mockModule) Start(ctx hulk.Context) error {
	m.startCalled = true
	return m.startError
}

func (m *mockModule) Stop(ctx context.Context) error {
	m.stopCalled = true
	return m.stopError
}

// Mock implementations for Context dependencies
type mockEventBus struct{}

func (m *mockEventBus) Publish(topic string, payload interface{}) error                  { return nil }
func (m *mockEventBus) Subscribe(topic string, handler events.Handler) error             { return nil }
func (m *mockEventBus) QueueSubscribe(topic, queue string, handler events.Handler) error { return nil }

type mockStore struct{}

func (m *mockStore) DB() *gorm.DB { return nil }

func (m *mockStore) CacheSet(ctx context.Context, key string, value interface{}, ttl int) error {
	return nil
}
func (m *mockStore) CacheGet(ctx context.Context, key string, target interface{}) error { return nil }
func (m *mockStore) CacheDelete(ctx context.Context, key string) error                  { return nil }
func (m *mockStore) QueryRow(ctx context.Context, query string, args ...interface{}) state.RowScanner {
	return nil
}
func (m *mockStore) Query(ctx context.Context, query string, args ...interface{}) (state.Rows, error) {
	return nil, nil
}
func (m *mockStore) Exec(ctx context.Context, query string, args ...interface{}) error { return nil }
func (m *mockStore) BeginTx(ctx context.Context) (state.Tx, error)                     { return nil, nil }

type mockAIClient struct{}

func (m *mockAIClient) Chat(ctx context.Context, prompt string) (string, error) {
	return "response", nil
}
func (m *mockAIClient) ChatWithContext(ctx context.Context, messages []ai.ChatMessage) (string, error) {
	return "response", nil
}
func (m *mockAIClient) AnalyzeError(ctx context.Context, context string, err error) (string, error) {
	return "analysis", nil
}
func (m *mockAIClient) Vision(ctx context.Context, imageURL string, prompt string) (*ai.AnalysisResult, error) {
	return nil, nil
}
func (m *mockAIClient) Embeddings(ctx context.Context, text string) ([]float32, error) {
	return nil, nil
}
func (m *mockAIClient) BatchEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	return nil, nil
}

func TestModuleConfig(t *testing.T) {
	config := hulk.ModuleConfig{
		ID:           "test-module",
		Name:         "Test Module",
		Version:      "v1.0.0",
		Dependencies: []string{"dep-1", "dep-2"},
	}

	assert.Equal(t, "test-module", config.ID)
	assert.Equal(t, "Test Module", config.Name)
	assert.Equal(t, "v1.0.0", config.Version)
	assert.Len(t, config.Dependencies, 2)
}

func TestModuleConfigNoDependencies(t *testing.T) {
	config := hulk.ModuleConfig{
		ID:      "simple-module",
		Name:    "Simple Module",
		Version: "v2.0.0",
	}

	assert.Equal(t, "simple-module", config.ID)
	assert.Nil(t, config.Dependencies)
}

func TestModuleInterface(t *testing.T) {
	module := &mockModule{
		config: hulk.ModuleConfig{
			ID:   "test",
			Name: "Test",
		},
	}

	// Verifica que implementa a interface
	var _ hulk.Module = module

	config := module.Config()
	assert.Equal(t, "test", config.ID)
}

func TestModuleLifecycle(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	ctx := hulk.NewContext(
		context.Background(),
		logger,
		&mockEventBus{},
		&mockStore{},
		&mockAIClient{},
		nil,
		nil,
	)

	module := &mockModule{
		config: hulk.ModuleConfig{
			ID:      "lifecycle-test",
			Name:    "Lifecycle Test Module",
			Version: "v1.0.0",
		},
	}

	// Test Init
	err := module.Init(ctx)
	assert.NoError(t, err)
	assert.True(t, module.initCalled)

	// Test Start
	err = module.Start(ctx)
	assert.NoError(t, err)
	assert.True(t, module.startCalled)

	// Test Stop
	err = module.Stop(context.Background())
	assert.NoError(t, err)
	assert.True(t, module.stopCalled)
}

func TestModuleInitError(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	ctx := hulk.NewContext(
		context.Background(),
		logger,
		&mockEventBus{},
		&mockStore{},
		&mockAIClient{},
		nil,
		nil,
	)

	expectedErr := errors.New("init failed")
	module := &mockModule{
		config: hulk.ModuleConfig{
			ID: "error-test",
		},
		initError: expectedErr,
	}

	err := module.Init(ctx)
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
}

func TestModuleStartError(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	ctx := hulk.NewContext(
		context.Background(),
		logger,
		&mockEventBus{},
		&mockStore{},
		&mockAIClient{},
		nil,
		nil,
	)

	expectedErr := errors.New("start failed")
	module := &mockModule{
		config: hulk.ModuleConfig{
			ID: "error-test",
		},
		startError: expectedErr,
	}

	err := module.Start(ctx)
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
}

func TestModuleStopError(t *testing.T) {
	expectedErr := errors.New("stop failed")
	module := &mockModule{
		config: hulk.ModuleConfig{
			ID: "error-test",
		},
		stopError: expectedErr,
	}

	err := module.Stop(context.Background())
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
}

func TestNewContext(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	bus := &mockEventBus{}
	store := &mockStore{}
	aiClient := &mockAIClient{}

	ctx := hulk.NewContext(
		context.Background(),
		logger,
		bus,
		store,
		aiClient,
		nil,
		nil,
	)

	assert.NotNil(t, ctx)
	assert.NotNil(t, ctx.Log())
	assert.NotNil(t, ctx.EventBus())
	assert.NotNil(t, ctx.Store())
	assert.NotNil(t, ctx.AI())
}

func TestContextLog(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	ctx := hulk.NewContext(
		context.Background(),
		logger,
		&mockEventBus{},
		&mockStore{},
		&mockAIClient{},
		nil,
		nil,
	)

	log := ctx.Log()
	assert.NotNil(t, log)
	assert.Equal(t, logger, log)
}

func TestContextEventBus(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	bus := &mockEventBus{}
	ctx := hulk.NewContext(
		context.Background(),
		logger,
		bus,
		&mockStore{},
		&mockAIClient{},
		nil,
		nil,
	)

	eventBus := ctx.EventBus()
	assert.NotNil(t, eventBus)
	assert.Equal(t, bus, eventBus)
}

func TestContextStore(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	store := &mockStore{}
	ctx := hulk.NewContext(
		context.Background(),
		logger,
		&mockEventBus{},
		store,
		&mockAIClient{},
		nil,
		nil,
	)

	s := ctx.Store()
	assert.NotNil(t, s)
	assert.Equal(t, store, s)
}

func TestContextAI(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	aiClient := &mockAIClient{}
	ctx := hulk.NewContext(
		context.Background(),
		logger,
		&mockEventBus{},
		&mockStore{},
		aiClient,
		nil,
		nil,
	)

	ai := ctx.AI()
	assert.NotNil(t, ai)
	assert.Equal(t, aiClient, ai)
}

func TestContextInheritsFromParent(t *testing.T) {
	type key string
	const testKey key = "test-key"

	parent := context.WithValue(context.Background(), testKey, "test-value")
	logger, _ := zap.NewDevelopment()

	ctx := hulk.NewContext(
		parent,
		logger,
		&mockEventBus{},
		&mockStore{},
		&mockAIClient{},
		nil,
		nil,
	)

	value := ctx.Value(testKey)
	assert.Equal(t, "test-value", value)
}

func TestContextCancellation(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	logger, _ := zap.NewDevelopment()

	ctx := hulk.NewContext(
		parent,
		logger,
		&mockEventBus{},
		&mockStore{},
		&mockAIClient{},
		nil,
		nil,
	)

	// Context should not be done initially
	select {
	case <-ctx.Done():
		t.Fatal("Context should not be done initially")
	default:
	}

	// Cancel parent
	cancel()

	// Context should be done after parent cancellation
	select {
	case <-ctx.Done():
		// Expected
	default:
		t.Fatal("Context should be done after parent cancellation")
	}
}
