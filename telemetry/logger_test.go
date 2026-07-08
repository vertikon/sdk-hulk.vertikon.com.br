package telemetry

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestNewLogger(t *testing.T) {
	base, _ := zap.NewDevelopment()
	logger := NewLogger(base)
	assert.NotNil(t, logger)
}

func TestLogger_WithTraceID(t *testing.T) {
	observedZapCore, observedLogs := observer.New(zap.InfoLevel)
	base := zap.New(observedZapCore)
	logger := NewLogger(base)

	log := logger.WithTraceID("test-trace-id")
	log.Info("test message")

	assert.Equal(t, 1, observedLogs.Len())
	entry := observedLogs.All()[0]
	assert.Equal(t, "test-trace-id", entry.ContextMap()["trace_id"])
}

func TestLogger_WithContext_NoTrace(t *testing.T) {
	observedZapCore, observedLogs := observer.New(zap.InfoLevel)
	base := zap.New(observedZapCore)
	logger := NewLogger(base)

	ctx := context.Background()
	log := logger.WithContext(ctx)
	log.Info("test message")

	assert.Equal(t, 1, observedLogs.Len())
	entry := observedLogs.All()[0]
	_, hasTrace := entry.ContextMap()["trace_id"]
	assert.False(t, hasTrace)
}

func TestLogger_WithContext_CustomTrace(t *testing.T) {
	observedZapCore, observedLogs := observer.New(zap.InfoLevel)
	base := zap.New(observedZapCore)
	logger := NewLogger(base)

	ctx := ContextWithTraceID(context.Background(), "custom-trace-id")
	log := logger.WithContext(ctx)
	log.Info("test message")

	assert.Equal(t, 1, observedLogs.Len())
	entry := observedLogs.All()[0]
	assert.Equal(t, "custom-trace-id", entry.ContextMap()["trace_id"])
}

func TestLogger_WithModule(t *testing.T) {
	observedZapCore, observedLogs := observer.New(zap.InfoLevel)
	base := zap.New(observedZapCore)
	logger := NewLogger(base)

	log := logger.WithModule("mod-1", "Module 1")
	log.Info("test message")

	assert.Equal(t, 1, observedLogs.Len())
	entry := observedLogs.All()[0]
	assert.Equal(t, "mod-1", entry.ContextMap()["module_id"])
	assert.Equal(t, "Module 1", entry.ContextMap()["module_name"])
}

func TestLogger_WithFields(t *testing.T) {
	observedZapCore, observedLogs := observer.New(zap.InfoLevel)
	base := zap.New(observedZapCore)
	logger := NewLogger(base)

	log := logger.WithFields(zap.String("key", "value"))
	log.Info("test message")

	assert.Equal(t, 1, observedLogs.Len())
	entry := observedLogs.All()[0]
	assert.Equal(t, "value", entry.ContextMap()["key"])
}
