package health

import (
	"context"
	"time"
)

// Status representa o status de saúde do sistema.
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusDegraded  Status = "degraded"
	StatusUnhealthy Status = "unhealthy"
)

// Check representa uma verificação de saúde.
type Check struct {
	Name      string        `json:"name"`
	Status    Status        `json:"status"`
	Message   string        `json:"message,omitempty"`
	Duration  time.Duration `json:"duration_ms"`
	Timestamp time.Time     `json:"timestamp"`
}

// Health representa o estado geral de saúde.
type Health struct {
	Status    Status           `json:"status"`
	Checks    map[string]Check `json:"checks"`
	Timestamp time.Time        `json:"timestamp"`
	Version   string           `json:"version"`
}

// Checker é uma interface para verificações de saúde.
type Checker interface {
	Check(ctx context.Context) Check
}

// Service gerencia verificações de saúde.
type Service struct {
	checkers []Checker
	version  string
}

// NewService cria um novo serviço de health check.
func NewService(version string) *Service {
	return &Service{
		checkers: make([]Checker, 0),
		version:  version,
	}
}

// Register adiciona um checker ao serviço.
func (s *Service) Register(checker Checker) {
	s.checkers = append(s.checkers, checker)
}

// Check executa todas as verificações e retorna o estado geral.
func (s *Service) Check(ctx context.Context) Health {
	checks := make(map[string]Check)
	overallStatus := StatusHealthy

	for _, checker := range s.checkers {
		check := checker.Check(ctx)
		checks[check.Name] = check

		if check.Status == StatusUnhealthy {
			overallStatus = StatusUnhealthy
		} else if check.Status == StatusDegraded && overallStatus == StatusHealthy {
			overallStatus = StatusDegraded
		}
	}

	return Health{
		Status:    overallStatus,
		Checks:    checks,
		Timestamp: time.Now(),
		Version:   s.version,
	}
}

// HTTPHandler retorna um handler HTTP para health check.
// Este é um exemplo - a implementação real ficará no hulk-core.
func (s *Service) HTTPHandler() interface{} {
	// Retorna uma função que pode ser usada como handler HTTP
	// A implementação real usará Echo ou outro framework
	return func(ctx context.Context) Health {
		return s.Check(ctx)
	}
}
