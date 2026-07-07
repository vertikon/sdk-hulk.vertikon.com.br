package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/vertikon/sdk-hulk.vertikon.com.br/internal/health"
	"go.uber.org/zap"
)

// NewHandler cria o handler HTTP para o servidor.
func NewHandler(healthService *health.Service) http.Handler {
	mux := http.NewServeMux()

	// Endpoint /health usando a infraestrutura do SDK
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		ctx := r.Context()
		healthStatus := healthService.Check(ctx)

		w.Header().Set("Content-Type", "application/json")

		// Status HTTP baseado no health status
		statusCode := http.StatusOK
		if healthStatus.Status == health.StatusUnhealthy {
			statusCode = http.StatusServiceUnavailable
		} else if healthStatus.Status == health.StatusDegraded {
			statusCode = http.StatusOK // 200 mas com status degraded
		}

		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(healthStatus)
	})

	// Endpoint raiz com informações do SDK
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"name":        "SDK-HULK",
			"version":     "v1.0.0",
			"description": "SDK para desenvolvimento de módulos Vertikon",
			"endpoints": []string{
				"GET /health - Health check endpoint",
				"GET / - SDK information",
			},
		})
	})

	return mux
}

// main é o ponto de entrada do SDK-HULK.
// Este arquivo serve como exemplo de como usar o SDK.
// Demonstra como implementar um servidor HTTP com health check.
func main() {
	logger, _ := zap.NewDevelopment()
	logger.Info("SDK-HULK - SDK para desenvolvimento de módulos Vertikon",
		zap.String("version", "v1.0.0"),
	)

	// Criar serviço de health check
	healthService := health.NewService("v1.0.0")

	// Configurar servidor HTTP
	handler := NewHandler(healthService)

	server := &http.Server{
		Addr:         ":8080",
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Canal para erros do servidor
	serverErrors := make(chan error, 1)

	// Iniciar servidor em goroutine
	go func() {
		logger.Info("Servidor HTTP iniciado",
			zap.String("addr", server.Addr),
			zap.String("health_endpoint", "http://localhost:8080/health"),
		)
		fmt.Println("\n🚀 SDK-HULK está pronto para uso!")
		fmt.Println("📍 Health check: http://localhost:8080/health")
		fmt.Println("📍 Info: http://localhost:8080/")
		fmt.Println("\n💡 Para usar o SDK, implemente a interface hulk.Module em seus módulos.")
		fmt.Println("📚 Veja examples/ para exemplos de implementação.")
		fmt.Println("\n⏹  Pressione Ctrl+C para parar o servidor")

		serverErrors <- server.ListenAndServe()
	}()

	// Canal para sinais de shutdown
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Aguardar shutdown ou erro
	select {
	case err := <-serverErrors:
		logger.Fatal("Erro ao iniciar servidor", zap.Error(err))
	case sig := <-shutdown:
		logger.Info("Iniciando shutdown graceful", zap.String("signal", sig.String()))

		// Criar contexto com timeout para shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Tentar shutdown graceful
		if err := server.Shutdown(ctx); err != nil {
			logger.Error("Erro durante shutdown", zap.Error(err))
			if err := server.Close(); err != nil {
				logger.Fatal("Erro ao forçar fechamento do servidor", zap.Error(err))
			}
		}

		logger.Info("Servidor encerrado com sucesso")
	}
}
