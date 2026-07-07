package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vertikon/sdk-hulk.vertikon.com.br/internal/health"
)

func TestHealthEndpoint(t *testing.T) {
	// Setup
	healthService := health.NewService("v1.0.0")
	handler := NewHandler(healthService)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	// Execute
	handler.ServeHTTP(w, req)

	// Verify
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "healthy", response["status"])
}

func TestRootEndpoint(t *testing.T) {
	// Setup
	healthService := health.NewService("v1.0.0")
	handler := NewHandler(healthService)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	// Execute
	handler.ServeHTTP(w, req)

	// Verify
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "SDK-HULK", response["name"])
}

func TestInvalidMethod(t *testing.T) {
	// Setup
	healthService := health.NewService("v1.0.0")
	handler := NewHandler(healthService)

	req := httptest.NewRequest("POST", "/health", nil)
	w := httptest.NewRecorder()

	// Execute
	handler.ServeHTTP(w, req)

	// Verify
	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestNotFound(t *testing.T) {
	// Setup
	healthService := health.NewService("v1.0.0")
	handler := NewHandler(healthService)

	req := httptest.NewRequest("GET", "/invalid", nil)
	w := httptest.NewRecorder()

	// Execute
	handler.ServeHTTP(w, req)

	// Verify
	assert.Equal(t, http.StatusNotFound, w.Code)
}
