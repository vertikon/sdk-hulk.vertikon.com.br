package telemetry

import (
	"context"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// Logger fornece operações de logging estruturado com suporte a tracing.
type Logger struct {
	base *zap.Logger
}

// NewLogger cria um novo Logger.
func NewLogger(base *zap.Logger) *Logger {
	return &Logger{base: base}
}

// WithTraceID adiciona um TraceID ao logger.
func (l *Logger) WithTraceID(traceID string) *zap.Logger {
	return l.base.With(zap.String("trace_id", traceID))
}

// WithContext extrai TraceID do context e adiciona ao logger.
func (l *Logger) WithContext(ctx context.Context) *zap.Logger {
	// Tenta extrair TraceID do context (padrão OpenTelemetry)
	if traceID := extractTraceID(ctx); traceID != "" {
		return l.WithTraceID(traceID)
	}
	return l.base
}

// WithModule adiciona informações do módulo ao logger.
func (l *Logger) WithModule(moduleID, moduleName string) *zap.Logger {
	return l.base.With(
		zap.String("module_id", moduleID),
		zap.String("module_name", moduleName),
	)
}

// WithFields adiciona campos customizados ao logger.
func (l *Logger) WithFields(fields ...zap.Field) *zap.Logger {
	return l.base.With(fields...)
}

// extractTraceID tenta extrair o TraceID do context.
// Suporta OpenTelemetry e formatos customizados.
func extractTraceID(ctx context.Context) string {
	// OpenTelemetry padrão
	span := trace.SpanFromContext(ctx)
	if span != nil {
		spanCtx := span.SpanContext()
		if spanCtx.IsValid() {
			return spanCtx.TraceID().String()
		}
	}

	// Formato customizado (se o HULK usar)
	if traceID, ok := ctx.Value("trace_id").(string); ok {
		return traceID
	}

	return ""
}

// LoggerWithTrace enriquece o logger com TraceID e SpanID se disponíveis no contexto
// Função helper para uso direto (conforme blueprint)
func LoggerWithTrace(ctx context.Context, logger *zap.Logger) *zap.Logger {
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return logger // Retorna logger original se não houver trace ativo
	}

	return logger.With(
		zap.String("trace_id", span.SpanContext().TraceID().String()),
		zap.String("span_id", span.SpanContext().SpanID().String()),
	)
}

// Debug logs a message at DebugLevel.
func (l *Logger) Debug(ctx context.Context, msg string, fields ...zap.Field) {
	l.WithContext(ctx).Debug(msg, fields...)
}

// Info logs a message at InfoLevel.
func (l *Logger) Info(ctx context.Context, msg string, fields ...zap.Field) {
	l.WithContext(ctx).Info(msg, fields...)
}

// Warn logs a message at WarnLevel.
func (l *Logger) Warn(ctx context.Context, msg string, fields ...zap.Field) {
	l.WithContext(ctx).Warn(msg, fields...)
}

// Error logs a message at ErrorLevel.
func (l *Logger) Error(ctx context.Context, msg string, err error, fields ...zap.Field) {
	if err != nil {
		fields = append(fields, zap.Error(err))
	}
	l.WithContext(ctx).Error(msg, fields...)
}

