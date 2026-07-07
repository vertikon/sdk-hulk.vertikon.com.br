package http

import (
	"time"

	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// OTelMiddleware instrumenta as rotas Echo automaticamente
// [BLOCO-P] Middleware de Observabilidade (Traces + Métricas)
func OTelMiddleware(serviceName string) echo.MiddlewareFunc {
	// Criar métricas uma vez (reutilizáveis)
	meter := otel.Meter("sdk-hulk/http")

	// Contador de requisições HTTP
	httpRequestsCounter, _ := meter.Int64Counter(
		"http_requests_total",
		metric.WithDescription("Total number of HTTP requests"),
		metric.WithUnit("1"),
	)

	// Histograma de latência HTTP
	httpRequestDuration, _ := meter.Float64Histogram(
		"http_request_duration_seconds",
		metric.WithDescription("HTTP request duration in seconds"),
		metric.WithUnit("s"),
	)

	// Contador de erros HTTP
	httpErrorsCounter, _ := meter.Int64Counter(
		"http_errors_total",
		metric.WithDescription("Total number of HTTP errors"),
		metric.WithUnit("1"),
	)

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			startTime := time.Now()

			// 1. Extrair Contexto (Propagation)
			// Permite continuar um trace que veio de fora (ex: de outro microsserviço ou gateway)
			ctx := req.Context()
			ctx = otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(req.Header))

			// 2. Iniciar Span
			// Nome do Span: "GET /api/v1/orders" (Usamos o Path da rota, não a URL crua, para agrupar melhor)
			tracer := otel.Tracer("sdk-hulk/http")
			path := c.Path()
			if path == "" {
				path = req.URL.Path // Fallback se rota não encontrada
			}
			spanName := req.Method + " " + path

			ctx, span := tracer.Start(ctx, spanName,
				oteltrace.WithAttributes(
					semconv.HTTPMethod(req.Method),
					semconv.HTTPRoute(path),
					semconv.HTTPURL(req.URL.String()),
					semconv.NetHostName(req.Host),
					semconv.ServiceName(serviceName),
				),
				oteltrace.WithSpanKind(oteltrace.SpanKindServer),
			)
			defer span.End()

			// 3. Injetar Contexto no Request
			// Atualiza o request com o contexto que contém o Span
			c.SetRequest(req.WithContext(ctx))

			// 4. Executar Próximo Handler
			err := next(c)

			// 5. Calcular Latência
			duration := time.Since(startTime).Seconds()
			status := c.Response().Status

			// 6. Atributos para Métricas
			attrs := []attribute.KeyValue{
				attribute.String("http.method", req.Method),
				attribute.String("http.route", path),
				attribute.Int("http.status_code", status),
				attribute.String("service.name", serviceName),
			}

			// 7. Registrar Métricas
			httpRequestsCounter.Add(ctx, 1, metric.WithAttributes(attrs...))
			httpRequestDuration.Record(ctx, duration, metric.WithAttributes(attrs...))

			// 8. Registrar Erros
			if status >= 400 {
				httpErrorsCounter.Add(ctx, 1, metric.WithAttributes(attrs...))
			}

			// 9. Registrar Resultado no Span
			span.SetAttributes(semconv.HTTPStatusCode(status))

			if err != nil {
				span.RecordError(err)
				span.SetAttributes(attribute.String("error.message", err.Error()))
				// Echo erro handler pode mudar o status, mas registramos o erro cru aqui
			}

			return err
		}
	}
}
