package telemetry

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Config define as opções de inicialização.
type Config struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	CollectorURL   string // Ex: "localhost:4317"
	SampleRatio    float64
}

// ShutdownFunc é a função para limpar recursos ao desligar a app.
type ShutdownFunc func(context.Context) error

// Init inicializa o OpenTelemetry (Tracing, Metrics e Propagation).
func Init(ctx context.Context, cfg Config) (ShutdownFunc, error) {
	// 1. Criar Recurso (Metadados da App)
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
			semconv.DeploymentEnvironment(cfg.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("falha ao criar resource: %w", err)
	}

	// 2. Configurar Conexão gRPC com o Collector
	// Nota: Usamos WithBlock() para garantir que a conexão existe antes de continuar.
	dialCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(dialCtx, cfg.CollectorURL,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("falha ao conectar gRPC OTel: %w", err)
	}

	// 3. Criar Exportador de Traces (OTLP gRPC)
	traceExporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, fmt.Errorf("falha ao criar trace exporter: %w", err)
	}

	// 4. Configurar Tracer Provider
	sampler := sdktrace.AlwaysSample()
	if strings.EqualFold(cfg.Environment, "production") {
		ratio := cfg.SampleRatio
		if ratio <= 0 || ratio > 1 {
			ratio = 0.1
		}
		sampler = sdktrace.ParentBased(sdktrace.TraceIDRatioBased(ratio))
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
		sdktrace.WithBatcher(traceExporter),
	)

	// 5. Configurar Métricas (MeterProvider)
	metricExporter, err := otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, fmt.Errorf("falha ao criar metric exporter: %w", err)
	}

	mp := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(metric.NewPeriodicReader(metricExporter,
			metric.WithInterval(10*time.Second),
		)),
	)

	// 6. Registrar Globalmente
	otel.SetTracerProvider(tp)
	otel.SetMeterProvider(mp)

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Função de limpeza (shutdown de traces e métricas)
	return func(ctx context.Context) error {
		if err := tp.Shutdown(ctx); err != nil {
			return err
		}
		return mp.Shutdown(ctx)
	}, nil
}

// GetTracer retorna uma instância nomeada do tracer.
func GetTracer(name string) interface{} {
	return otel.Tracer(name)
}
