package health

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// MockChecker implementa Checker para testes
type MockChecker struct {
	name   string
	status Status
	err    error
}

func NewMockChecker(name string, status Status) *MockChecker {
	return &MockChecker{
		name:   name,
		status: status,
	}
}

func (m *MockChecker) Check(ctx context.Context) Check {
	start := time.Now()

	check := Check{
		Name:      m.name,
		Status:    m.status,
		Duration:  time.Since(start),
		Timestamp: time.Now(),
	}

	if m.err != nil {
		check.Message = m.err.Error()
	}

	return check
}

func (m *MockChecker) SetStatus(status Status) {
	m.status = status
}

func (m *MockChecker) SetError(err error) {
	m.err = err
}

// Testes

func TestStatus_Constants(t *testing.T) {
	if StatusHealthy != "healthy" {
		t.Errorf("Expected StatusHealthy to be 'healthy', got '%s'", StatusHealthy)
	}

	if StatusDegraded != "degraded" {
		t.Errorf("Expected StatusDegraded to be 'degraded', got '%s'", StatusDegraded)
	}

	if StatusUnhealthy != "unhealthy" {
		t.Errorf("Expected StatusUnhealthy to be 'unhealthy', got '%s'", StatusUnhealthy)
	}
}

func TestNewService(t *testing.T) {
	version := "v1.0.0"
	service := NewService(version)

	if service == nil {
		t.Fatal("NewService returned nil")
	}

	if service.version != version {
		t.Errorf("Expected version '%s', got '%s'", version, service.version)
	}

	if service.checkers == nil {
		t.Error("Service checkers should be initialized")
	}
}

func TestService_Register(t *testing.T) {
	service := NewService("v1.0.0")
	checker := NewMockChecker("test", StatusHealthy)

	service.Register(checker)

	if len(service.checkers) != 1 {
		t.Errorf("Expected 1 checker, got %d", len(service.checkers))
	}
}

func TestService_RegisterMultiple(t *testing.T) {
	service := NewService("v1.0.0")

	checkers := []Checker{
		NewMockChecker("db", StatusHealthy),
		NewMockChecker("cache", StatusHealthy),
		NewMockChecker("queue", StatusHealthy),
	}

	for _, checker := range checkers {
		service.Register(checker)
	}

	if len(service.checkers) != len(checkers) {
		t.Errorf("Expected %d checkers, got %d", len(checkers), len(service.checkers))
	}
}

func TestService_Check_NoCheckers(t *testing.T) {
	service := NewService("v1.0.0")
	ctx := context.Background()

	health := service.Check(ctx)

	if health.Status != StatusHealthy {
		t.Errorf("Expected status '%s', got '%s'", StatusHealthy, health.Status)
	}

	if health.Version != "v1.0.0" {
		t.Errorf("Expected version 'v1.0.0', got '%s'", health.Version)
	}

	if len(health.Checks) != 0 {
		t.Errorf("Expected 0 checks, got %d", len(health.Checks))
	}
}

func TestService_Check_AllHealthy(t *testing.T) {
	service := NewService("v1.0.0")
	ctx := context.Background()

	service.Register(NewMockChecker("db", StatusHealthy))
	service.Register(NewMockChecker("cache", StatusHealthy))

	health := service.Check(ctx)

	if health.Status != StatusHealthy {
		t.Errorf("Expected overall status '%s', got '%s'", StatusHealthy, health.Status)
	}

	if len(health.Checks) != 2 {
		t.Errorf("Expected 2 checks, got %d", len(health.Checks))
	}
}

func TestService_Check_OneDegraded(t *testing.T) {
	service := NewService("v1.0.0")
	ctx := context.Background()

	service.Register(NewMockChecker("db", StatusHealthy))
	service.Register(NewMockChecker("cache", StatusDegraded))

	health := service.Check(ctx)

	if health.Status != StatusDegraded {
		t.Errorf("Expected overall status '%s', got '%s'", StatusDegraded, health.Status)
	}
}

func TestService_Check_OneUnhealthy(t *testing.T) {
	service := NewService("v1.0.0")
	ctx := context.Background()

	service.Register(NewMockChecker("db", StatusHealthy))
	service.Register(NewMockChecker("cache", StatusUnhealthy))

	health := service.Check(ctx)

	if health.Status != StatusUnhealthy {
		t.Errorf("Expected overall status '%s', got '%s'", StatusUnhealthy, health.Status)
	}
}

func TestService_Check_UnhealthyOverridesDegraded(t *testing.T) {
	service := NewService("v1.0.0")
	ctx := context.Background()

	service.Register(NewMockChecker("db", StatusDegraded))
	service.Register(NewMockChecker("cache", StatusUnhealthy))
	service.Register(NewMockChecker("queue", StatusHealthy))

	health := service.Check(ctx)

	if health.Status != StatusUnhealthy {
		t.Errorf("Expected overall status '%s', got '%s'", StatusUnhealthy, health.Status)
	}
}

func TestCheck_Fields(t *testing.T) {
	checker := NewMockChecker("test-service", StatusHealthy)
	ctx := context.Background()

	check := checker.Check(ctx)

	if check.Name != "test-service" {
		t.Errorf("Expected name 'test-service', got '%s'", check.Name)
	}

	if check.Status != StatusHealthy {
		t.Errorf("Expected status '%s', got '%s'", StatusHealthy, check.Status)
	}

	if check.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
}

func TestCheck_WithError(t *testing.T) {
	checker := NewMockChecker("failing-service", StatusUnhealthy)
	expectedErr := errors.New("connection failed")
	checker.SetError(expectedErr)

	ctx := context.Background()
	check := checker.Check(ctx)

	if check.Status != StatusUnhealthy {
		t.Errorf("Expected status '%s', got '%s'", StatusUnhealthy, check.Status)
	}

	if check.Message != expectedErr.Error() {
		t.Errorf("Expected message '%s', got '%s'", expectedErr.Error(), check.Message)
	}
}

func TestHealth_Timestamp(t *testing.T) {
	service := NewService("v1.0.0")
	ctx := context.Background()

	before := time.Now()
	health := service.Check(ctx)
	after := time.Now()

	if health.Timestamp.Before(before) || health.Timestamp.After(after) {
		t.Error("Health timestamp should be between before and after")
	}
}

func TestChecker_Interface(t *testing.T) {
	// Verificar que MockChecker implementa Checker
	var _ Checker = (*MockChecker)(nil)
}

func TestService_ConcurrentCheck(t *testing.T) {
	service := NewService("v1.0.0")
	ctx := context.Background()

	service.Register(NewMockChecker("db", StatusHealthy))
	service.Register(NewMockChecker("cache", StatusHealthy))

	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			service.Check(ctx)
		}()
	}

	wg.Wait()
}

func TestService_DynamicStatusChange(t *testing.T) {
	service := NewService("v1.0.0")
	ctx := context.Background()

	checker := NewMockChecker("dynamic", StatusHealthy)
	service.Register(checker)

	// First check - healthy
	health1 := service.Check(ctx)
	if health1.Status != StatusHealthy {
		t.Errorf("Expected status '%s', got '%s'", StatusHealthy, health1.Status)
	}

	// Change status
	checker.SetStatus(StatusUnhealthy)

	// Second check - unhealthy
	health2 := service.Check(ctx)
	if health2.Status != StatusUnhealthy {
		t.Errorf("Expected status '%s', got '%s'", StatusUnhealthy, health2.Status)
	}
}

func TestService_ChecksMap(t *testing.T) {
	service := NewService("v1.0.0")
	ctx := context.Background()

	names := []string{"db", "cache", "queue"}

	for _, name := range names {
		service.Register(NewMockChecker(name, StatusHealthy))
	}

	health := service.Check(ctx)

	for _, name := range names {
		if _, ok := health.Checks[name]; !ok {
			t.Errorf("Expected check '%s' in checks map", name)
		}
	}
}

func TestService_HTTPHandler(t *testing.T) {
	service := NewService("v1.0.0")

	handler := service.HTTPHandler()

	if handler == nil {
		t.Error("HTTPHandler should not return nil")
	}

	// Verificar que é uma função
	if handlerFunc, ok := handler.(func(context.Context) Health); ok {
		ctx := context.Background()
		health := handlerFunc(ctx)

		if health.Version != "v1.0.0" {
			t.Errorf("Expected version 'v1.0.0', got '%s'", health.Version)
		}
	} else {
		t.Error("HTTPHandler should return a function")
	}
}

func TestCheck_Duration(t *testing.T) {
	checker := NewMockChecker("test", StatusHealthy)
	ctx := context.Background()

	check := checker.Check(ctx)

	if check.Duration < 0 {
		t.Error("Duration should not be negative")
	}
}

func TestHealth_MultipleChecksWithDifferentStatuses(t *testing.T) {
	service := NewService("v1.0.0")
	ctx := context.Background()

	service.Register(NewMockChecker("healthy1", StatusHealthy))
	service.Register(NewMockChecker("healthy2", StatusHealthy))
	service.Register(NewMockChecker("degraded1", StatusDegraded))
	service.Register(NewMockChecker("unhealthy1", StatusUnhealthy))

	health := service.Check(ctx)

	// Overall should be unhealthy
	if health.Status != StatusUnhealthy {
		t.Errorf("Expected overall status '%s', got '%s'", StatusUnhealthy, health.Status)
	}

	// Should have all checks
	if len(health.Checks) != 4 {
		t.Errorf("Expected 4 checks, got %d", len(health.Checks))
	}

	// Verify individual statuses
	if health.Checks["healthy1"].Status != StatusHealthy {
		t.Error("healthy1 should be healthy")
	}
	if health.Checks["degraded1"].Status != StatusDegraded {
		t.Error("degraded1 should be degraded")
	}
	if health.Checks["unhealthy1"].Status != StatusUnhealthy {
		t.Error("unhealthy1 should be unhealthy")
	}
}

// Benchmark tests

func BenchmarkService_Check(b *testing.B) {
	service := NewService("v1.0.0")
	ctx := context.Background()

	service.Register(NewMockChecker("db", StatusHealthy))
	service.Register(NewMockChecker("cache", StatusHealthy))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.Check(ctx)
	}
}

func BenchmarkService_CheckWithManyCheckers(b *testing.B) {
	service := NewService("v1.0.0")
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		service.Register(NewMockChecker("checker", StatusHealthy))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.Check(ctx)
	}
}

func BenchmarkChecker_Check(b *testing.B) {
	checker := NewMockChecker("bench", StatusHealthy)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		checker.Check(ctx)
	}
}

func BenchmarkService_ConcurrentCheck(b *testing.B) {
	service := NewService("v1.0.0")
	ctx := context.Background()

	service.Register(NewMockChecker("db", StatusHealthy))

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			service.Check(ctx)
		}
	})
}
