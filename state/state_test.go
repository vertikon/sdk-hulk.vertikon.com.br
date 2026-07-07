package state

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// MockStore implementa Store para testes
type MockStore struct {
	cache map[string]interface{}
	mu    sync.RWMutex
}

func (m *MockStore) DB() *gorm.DB { return nil }

func NewMockStore() *MockStore {
	return &MockStore{
		cache: make(map[string]interface{}),
	}
}

func (m *MockStore) CacheSet(ctx context.Context, key string, value interface{}, ttl int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cache[key] = value
	return nil
}

func (m *MockStore) CacheGet(ctx context.Context, key string, target interface{}) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	val, ok := m.cache[key]
	if !ok {
		return errors.New("key not found")
	}

	// Simular deserialização
	if ptr, ok := target.(*string); ok {
		if str, ok := val.(string); ok {
			*ptr = str
		}
	}

	return nil
}

func (m *MockStore) CacheDelete(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.cache, key)
	return nil
}

func (m *MockStore) QueryRow(ctx context.Context, query string, args ...interface{}) RowScanner {
	return &MockRowScanner{}
}

func (m *MockStore) Query(ctx context.Context, query string, args ...interface{}) (Rows, error) {
	return &MockRows{}, nil
}

func (m *MockStore) Exec(ctx context.Context, query string, args ...interface{}) error {
	return nil
}

func (m *MockStore) BeginTx(ctx context.Context) (Tx, error) {
	return &MockTx{}, nil
}

// MockRowScanner implementa RowScanner
type MockRowScanner struct{}

func (m *MockRowScanner) Scan(dest ...interface{}) error {
	return nil
}

// MockRows implementa Rows
type MockRows struct {
	closed bool
}

func (m *MockRows) Next() bool {
	return false
}

func (m *MockRows) Scan(dest ...interface{}) error {
	return nil
}

func (m *MockRows) Close() error {
	m.closed = true
	return nil
}

func (m *MockRows) Err() error {
	return nil
}

// MockTx implementa Tx
type MockTx struct{}

func (m *MockTx) Exec(ctx context.Context, query string, args ...interface{}) error {
	return nil
}

func (m *MockTx) QueryRow(ctx context.Context, query string, args ...interface{}) RowScanner {
	return &MockRowScanner{}
}

func (m *MockTx) Query(ctx context.Context, query string, args ...interface{}) (Rows, error) {
	return &MockRows{}, nil
}

func (m *MockTx) Commit() error {
	return nil
}

func (m *MockTx) Rollback() error {
	return nil
}

// FailingMockStore implementa Store com capacidade de falhar
type FailingMockStore struct {
	setError    error
	getError    error
	deleteError error
	cache       map[string]interface{}
	mu          sync.RWMutex
}

func (m *FailingMockStore) DB() *gorm.DB { return nil
}

func NewFailingMockStore() *FailingMockStore {
	return &FailingMockStore{
		cache: make(map[string]interface{}),
	}
}

func (m *FailingMockStore) CacheSet(ctx context.Context, key string, value interface{}, ttl int) error {
	if m.setError != nil {
		return m.setError
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cache[key] = value
	return nil
}

func (m *FailingMockStore) CacheGet(ctx context.Context, key string, target interface{}) error {
	if m.getError != nil {
		return m.getError
	}
	m.mu.RLock()
	defer m.mu.RUnlock()

	val, ok := m.cache[key]
	if !ok {
		return errors.New("key not found")
	}

	if ptr, ok := target.(*string); ok {
		if str, ok := val.(string); ok {
			*ptr = str
		}
	}
	return nil
}

func (m *FailingMockStore) CacheDelete(ctx context.Context, key string) error {
	if m.deleteError != nil {
		return m.deleteError
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.cache, key)
	return nil
}

func (m *FailingMockStore) QueryRow(ctx context.Context, query string, args ...interface{}) RowScanner {
	return &MockRowScanner{}
}

func (m *FailingMockStore) Query(ctx context.Context, query string, args ...interface{}) (Rows, error) {
	return &MockRows{}, nil
}

func (m *FailingMockStore) Exec(ctx context.Context, query string, args ...interface{}) error {
	return nil
}

func (m *FailingMockStore) BeginTx(ctx context.Context) (Tx, error) {
	return &MockTx{}, nil
}

// Testes

func TestMockStore_CacheSetGet(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	key := "test:key"
	value := "test value"

	// Set
	err := store.CacheSet(ctx, key, value, 3600)
	if err != nil {
		t.Fatalf("CacheSet failed: %v", err)
	}

	// Get
	var result string
	err = store.CacheGet(ctx, key, &result)
	if err != nil {
		t.Fatalf("CacheGet failed: %v", err)
	}

	if result != value {
		t.Errorf("Expected '%s', got '%s'", value, result)
	}
}

func TestMockStore_CacheGetNotFound(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	var result string
	err := store.CacheGet(ctx, "nonexistent", &result)

	if err == nil {
		t.Error("Expected error for nonexistent key")
	}
}

func TestMockStore_CacheDelete(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	key := "test:delete"

	// Set
	store.CacheSet(ctx, key, "value", 3600)

	// Delete
	err := store.CacheDelete(ctx, key)
	if err != nil {
		t.Fatalf("CacheDelete failed: %v", err)
	}

	// Verify deleted
	var result string
	err = store.CacheGet(ctx, key, &result)
	if err == nil {
		t.Error("Expected error after delete")
	}
}

func TestCache_New(t *testing.T) {
	store := NewMockStore()
	cache := NewCache(store)

	if cache == nil {
		t.Fatal("NewCache returned nil")
	}
}

func TestCache_Set(t *testing.T) {
	store := NewMockStore()
	cache := NewCache(store)
	ctx := context.Background()

	err := cache.Set(ctx, "key1", "value1", 3600*time.Second)
	if err != nil {
		t.Fatalf("Cache.Set failed: %v", err)
	}
}

func TestCache_SetWithDefaultTTL(t *testing.T) {
	store := NewMockStore()
	cache := NewCache(store)
	ctx := context.Background()

	err := cache.SetWithDefaultTTL(ctx, "key-default", "value-default")
	if err != nil {
		t.Fatalf("Cache.SetWithDefaultTTL failed: %v", err)
	}

	// Verify it was set
	var result string
	err = cache.Get(ctx, "key-default", &result)
	if err != nil {
		t.Fatalf("Cache.Get failed: %v", err)
	}

	if result != "value-default" {
		t.Errorf("Expected 'value-default', got '%s'", result)
	}
}

func TestCache_SetNegativeTTL(t *testing.T) {
	store := NewMockStore()
	cache := NewCache(store)
	ctx := context.Background()

	// Negative TTL should be treated as 0 (no expiration)
	err := cache.Set(ctx, "key-no-expire", "value", -1*time.Second)
	if err != nil {
		t.Fatalf("Cache.Set with negative TTL failed: %v", err)
	}
}

func TestCache_Get(t *testing.T) {
	store := NewMockStore()
	cache := NewCache(store)
	ctx := context.Background()

	// Set first
	cache.Set(ctx, "key2", "value2", 3600)

	// Get
	var result string
	err := cache.Get(ctx, "key2", &result)
	if err != nil {
		t.Fatalf("Cache.Get failed: %v", err)
	}

	if result != "value2" {
		t.Errorf("Expected 'value2', got '%s'", result)
	}
}

func TestCache_Delete(t *testing.T) {
	store := NewMockStore()
	cache := NewCache(store)
	ctx := context.Background()

	// Set
	cache.Set(ctx, "key3", "value3", 3600)

	// Delete
	err := cache.Delete(ctx, "key3")
	if err != nil {
		t.Fatalf("Cache.Delete failed: %v", err)
	}

	// Verify
	var result string
	err = cache.Get(ctx, "key3", &result)
	if err == nil {
		t.Error("Expected error after delete")
	}
}

func TestCache_GetOrSet_CacheHit(t *testing.T) {
	store := NewMockStore()
	cache := NewCache(store)
	ctx := context.Background()

	// Set initial value
	cache.Set(ctx, "cached", "cached value", 3600)

	generatorCalled := false
	generator := func() (interface{}, error) {
		generatorCalled = true
		return "new value", nil
	}

	var result string
	err := cache.GetOrSet(ctx, "cached", &result, 3600, generator)

	if err != nil {
		t.Fatalf("GetOrSet failed: %v", err)
	}

	if generatorCalled {
		t.Error("Generator should not be called on cache hit")
	}

	if result != "cached value" {
		t.Errorf("Expected 'cached value', got '%s'", result)
	}
}

func TestCache_GetOrSet_CacheMiss(t *testing.T) {
	store := NewMockStore()
	cache := NewCache(store)
	ctx := context.Background()

	generatorCalled := false
	expectedValue := "generated value"

	generator := func() (interface{}, error) {
		generatorCalled = true
		return expectedValue, nil
	}

	var result string
	err := cache.GetOrSet(ctx, "missing", &result, 3600, generator)

	if err != nil {
		t.Fatalf("GetOrSet failed: %v", err)
	}

	if !generatorCalled {
		t.Error("Generator should be called on cache miss")
	}
}

func TestCache_GetOrSet_GeneratorError(t *testing.T) {
	store := NewMockStore()
	cache := NewCache(store)
	ctx := context.Background()

	expectedErr := errors.New("generator error")

	generator := func() (interface{}, error) {
		return nil, expectedErr
	}

	var result string
	err := cache.GetOrSet(ctx, "error", &result, 3600*time.Second, generator)

	if err != expectedErr {
		t.Errorf("Expected error '%v', got '%v'", expectedErr, err)
	}
}

// Error Handling Tests

func TestCache_SetError(t *testing.T) {
	store := NewFailingMockStore()
	store.setError = errors.New("set failed")
	cache := NewCache(store)
	ctx := context.Background()

	err := cache.Set(ctx, "key", "value", time.Hour)
	assert.Error(t, err)
	assert.Equal(t, "set failed", err.Error())
}

func TestCache_GetError(t *testing.T) {
	store := NewFailingMockStore()
	store.getError = errors.New("get failed")
	cache := NewCache(store)
	ctx := context.Background()

	var result string
	err := cache.Get(ctx, "key", &result)
	assert.Error(t, err)
	assert.Equal(t, "get failed", err.Error())
}

func TestCache_DeleteError(t *testing.T) {
	store := NewFailingMockStore()
	store.deleteError = errors.New("delete failed")
	cache := NewCache(store)
	ctx := context.Background()

	err := cache.Delete(ctx, "key")
	assert.Error(t, err)
	assert.Equal(t, "delete failed", err.Error())
}

func TestCache_SetWithDefaultTTLError(t *testing.T) {
	store := NewFailingMockStore()
	store.setError = errors.New("set failed")
	cache := NewCache(store)
	ctx := context.Background()

	err := cache.SetWithDefaultTTL(ctx, "key", "value")
	assert.Error(t, err)
}

func TestCache_GetOrSetCacheSetError(t *testing.T) {
	store := NewFailingMockStore()
	cache := NewCache(store)
	ctx := context.Background()

	generator := func() (interface{}, error) {
		return "generated", nil
	}

	// Set error after generator succeeds
	store.setError = errors.New("cache set failed")

	var result string
	err := cache.GetOrSet(ctx, "missing", &result, time.Hour, generator)
	assert.Error(t, err)
	assert.Equal(t, "cache set failed", err.Error())
}

func TestCache_GetOrSetFinalGetError(t *testing.T) {
	store := NewFailingMockStore()
	cache := NewCache(store)
	ctx := context.Background()

	generator := func() (interface{}, error) {
		return "generated", nil
	}

	// First Get fails (cache miss), Set succeeds, final Get fails

	store.getError = errors.New("get failed")

	var result string
	err := cache.GetOrSet(ctx, "key", &result, time.Hour, generator)
	assert.Error(t, err)
}

// TTL Edge Cases

func TestCache_SetZeroTTL(t *testing.T) {
	store := NewMockStore()
	cache := NewCache(store)
	ctx := context.Background()

	err := cache.Set(ctx, "key-zero", "value", 0*time.Second)
	assert.NoError(t, err)

	// Verify it was set
	var result string
	err = cache.Get(ctx, "key-zero", &result)
	assert.NoError(t, err)
	assert.Equal(t, "value", result)
}

func TestCache_SetVeryLargeTTL(t *testing.T) {
	store := NewMockStore()
	cache := NewCache(store)
	ctx := context.Background()

	// 1 year TTL
	err := cache.Set(ctx, "key-long", "value", 365*24*time.Hour)
	assert.NoError(t, err)
}

func TestCache_SetMillisecondTTL(t *testing.T) {
	store := NewMockStore()
	cache := NewCache(store)
	ctx := context.Background()

	// Very short TTL (will be converted to 0 seconds)
	err := cache.Set(ctx, "key-ms", "value", 500*time.Millisecond)
	assert.NoError(t, err)
}

// Advanced Concurrency Tests

func TestCache_ConcurrentSetSameKey(t *testing.T) {
	store := NewMockStore()
	cache := NewCache(store)
	ctx := context.Background()

	var wg sync.WaitGroup
	key := "concurrent-key"

	// 100 goroutines setting the same key
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			cache.Set(ctx, key, n, time.Hour)
		}(i)
	}

	wg.Wait()

	// Should not panic or error
	var result string
	err := cache.Get(ctx, key, &result)
	assert.NoError(t, err)
}

func TestCache_ConcurrentSetGet(t *testing.T) {
	store := NewMockStore()
	cache := NewCache(store)
	ctx := context.Background()

	var wg sync.WaitGroup

	// Writers
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := "key-" + string(rune(n))
			cache.Set(ctx, key, n, time.Hour)
		}(i)
	}

	// Readers
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := "key-" + string(rune(n))
			var result int
			cache.Get(ctx, key, &result)
		}(i)
	}

	wg.Wait()
}

func TestCache_ConcurrentDelete(t *testing.T) {
	store := NewMockStore()
	cache := NewCache(store)
	ctx := context.Background()

	// Set initial value
	cache.Set(ctx, "delete-key", "value", time.Hour)

	var wg sync.WaitGroup

	// Multiple goroutines trying to delete the same key
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cache.Delete(ctx, "delete-key")
		}()
	}

	wg.Wait()
}

func TestCache_ConcurrentGetOrSet(t *testing.T) {
	store := NewMockStore()
	cache := NewCache(store)
	ctx := context.Background()

	var wg sync.WaitGroup
	var mu sync.Mutex
	callCount := 0

	generator := func() (interface{}, error) {
		mu.Lock()
		callCount++
		mu.Unlock()
		time.Sleep(10 * time.Millisecond)
		return "generated", nil
	}

	// Multiple goroutines calling GetOrSet for the same key
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var result string
			cache.GetOrSet(ctx, "concurrent-getorset", &result, time.Hour, generator)
		}()
	}

	wg.Wait()

	// Generator may be called multiple times due to race conditions
	// This is expected behavior without additional locking
	assert.Greater(t, callCount, 0)
}

func TestStore_Interface(t *testing.T) {
	// Verificar que MockStore implementa Store
	var _ Store = (*MockStore)(nil)
}

func TestStore_QueryRow(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	scanner := store.QueryRow(ctx, "SELECT * FROM users WHERE id = ?", 1)
	assert.NotNil(t, scanner)
}

func TestStore_QueryRowMultipleArgs(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	scanner := store.QueryRow(ctx, "SELECT * FROM users WHERE id = ? AND status = ?", 1, "active")
	assert.NotNil(t, scanner)
}

func TestStore_Query(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	rows, err := store.Query(ctx, "SELECT * FROM users")
	assert.NoError(t, err)
	assert.NotNil(t, rows)
}

func TestStore_QueryWithArgs(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	rows, err := store.Query(ctx, "SELECT * FROM users WHERE status = ?", "active")
	assert.NoError(t, err)
	assert.NotNil(t, rows)
}

func TestStore_Exec(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	err := store.Exec(ctx, "INSERT INTO users VALUES (?)", "test")
	assert.NoError(t, err)
}

func TestStore_ExecUpdate(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	err := store.Exec(ctx, "UPDATE users SET status = ? WHERE id = ?", "inactive", 1)
	assert.NoError(t, err)
}

func TestStore_ExecDelete(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	err := store.Exec(ctx, "DELETE FROM users WHERE id = ?", 1)
	assert.NoError(t, err)
}

func TestStore_BeginTx(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	tx, err := store.BeginTx(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, tx)
}

func TestTx_Commit(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	tx, err := store.BeginTx(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, tx)

	err = tx.Commit()
	assert.NoError(t, err)
}

func TestTx_Rollback(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	tx, err := store.BeginTx(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, tx)

	err = tx.Rollback()
	assert.NoError(t, err)
}

func TestTx_Exec(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	tx, _ := store.BeginTx(ctx)

	err := tx.Exec(ctx, "INSERT INTO users VALUES (?)", "test")
	assert.NoError(t, err)
}

func TestTx_QueryRow(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	tx, _ := store.BeginTx(ctx)

	scanner := tx.QueryRow(ctx, "SELECT * FROM users WHERE id = ?", 1)
	assert.NotNil(t, scanner)
}

func TestTx_Query(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	tx, _ := store.BeginTx(ctx)

	rows, err := tx.Query(ctx, "SELECT * FROM users")
	assert.NoError(t, err)
	assert.NotNil(t, rows)
}

func TestTx_FullWorkflow(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	// Begin transaction
	tx, err := store.BeginTx(ctx)
	assert.NoError(t, err)

	// Execute operations
	err = tx.Exec(ctx, "INSERT INTO users VALUES (?)", "user1")
	assert.NoError(t, err)

	err = tx.Exec(ctx, "INSERT INTO users VALUES (?)", "user2")
	assert.NoError(t, err)

	// Commit
	err = tx.Commit()
	assert.NoError(t, err)
}

func TestTx_RollbackWorkflow(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	// Begin transaction
	tx, err := store.BeginTx(ctx)
	assert.NoError(t, err)

	// Execute operation
	err = tx.Exec(ctx, "INSERT INTO users VALUES (?)", "user1")
	assert.NoError(t, err)

	// Rollback instead of commit
	err = tx.Rollback()
	assert.NoError(t, err)
}

func TestRows_Next(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	rows, _ := store.Query(ctx, "SELECT * FROM users")

	hasNext := rows.Next()
	assert.False(t, hasNext) // Mock returns false
}

func TestRows_Scan(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	rows, _ := store.Query(ctx, "SELECT * FROM users")

	var id int
	var name string
	err := rows.Scan(&id, &name)
	assert.NoError(t, err)
}

func TestRows_Close(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	rows, _ := store.Query(ctx, "SELECT * FROM users")

	err := rows.Close()
	assert.NoError(t, err)
}

func TestRows_Err(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	rows, _ := store.Query(ctx, "SELECT * FROM users")

	err := rows.Err()
	assert.NoError(t, err)
}

func TestRows_FullIteration(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	rows, err := store.Query(ctx, "SELECT * FROM users")
	assert.NoError(t, err)
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id int
		var name string
		err := rows.Scan(&id, &name)
		assert.NoError(t, err)
		count++
	}

	assert.Equal(t, 0, count) // Mock returns no rows

	err = rows.Err()
	assert.NoError(t, err)
}

func TestRowScanner_Scan(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	scanner := store.QueryRow(ctx, "SELECT id, name FROM users WHERE id = ?", 1)

	var id int
	var name string
	err := scanner.Scan(&id, &name)
	assert.NoError(t, err)
}

func TestRowScanner_ScanSingleValue(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	scanner := store.QueryRow(ctx, "SELECT COUNT(*) FROM users")

	var count int
	err := scanner.Scan(&count)
	assert.NoError(t, err)
}

func TestStore_ConcurrentAccess(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	var wg sync.WaitGroup

	// Concurrent writes
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := "concurrent"
			store.CacheSet(ctx, key, n, 3600)
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var result string
			store.CacheGet(ctx, "concurrent", &result)
		}()
	}

	wg.Wait()
}

func TestStore_QueryExec(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	// Query
	_, err := store.Query(ctx, "SELECT * FROM test")
	if err != nil {
		t.Errorf("Query failed: %v", err)
	}

	// Exec
	err = store.Exec(ctx, "INSERT INTO test VALUES (?)", "value")
	if err != nil {
		t.Errorf("Exec failed: %v", err)
	}
}

func TestCache_MultipleKeys(t *testing.T) {
	store := NewMockStore()
	cache := NewCache(store)
	ctx := context.Background()

	keys := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	// Set all
	for k, v := range keys {
		cache.Set(ctx, k, v, 3600)
	}

	// Get all
	for k, expectedV := range keys {
		var result string
		err := cache.Get(ctx, k, &result)
		if err != nil {
			t.Errorf("Get failed for key '%s': %v", k, err)
		}
		if result != expectedV {
			t.Errorf("Key '%s': expected '%s', got '%s'", k, expectedV, result)
		}
	}
}

// Benchmark tests

func BenchmarkCache_Set(b *testing.B) {
	store := NewMockStore()
	cache := NewCache(store)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Set(ctx, "bench-key", "bench-value", 3600)
	}
}

func BenchmarkCache_Get(b *testing.B) {
	store := NewMockStore()
	cache := NewCache(store)
	ctx := context.Background()

	cache.Set(ctx, "bench-key", "bench-value", 3600)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result string
		cache.Get(ctx, "bench-key", &result)
	}
}

func BenchmarkCache_GetOrSet(b *testing.B) {
	store := NewMockStore()
	cache := NewCache(store)
	ctx := context.Background()

	generator := func() (interface{}, error) {
		return "value", nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result string
		cache.GetOrSet(ctx, "bench-key", &result, 3600, generator)
	}
}

func BenchmarkStore_ConcurrentSet(b *testing.B) {
	store := NewMockStore()
	ctx := context.Background()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			store.CacheSet(ctx, "concurrent", i, 3600)
			i++
		}
	})
}
