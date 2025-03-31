package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"example.com/my-inventory/semconv"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
)

var log = logrus.New()

// --- Métricas do Prometheus ---

var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Número total de requisições HTTP recebidas",
		},
		[]string{"path", "method", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duração das requisições HTTP em segundos",
			Buckets: prometheus.DefBuckets, // Use os buckets padrão do Prometheus (boa opção inicial)
			// Ou defina seus próprios buckets:
			// Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"path", "method"}, // Não inclua o status code no histograma de latência!
	)

	activeConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "http_active_connections",
		Help: "Número de conexões HTTP ativas",
	})

	//Exemplo de métrica específica da aplicação
	productsInDB = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "products_in_db",
		Help: "Número de produtos no banco de dados",
	})

	//Exemplo de métrica de erro
	sqlErrorsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "sql_errors_total",
		Help: "Número total de erros de SQL",
	})
)

// ResponseWriterWrapper para capturar o status code
type ResponseWriterWrapper struct {
	http.ResponseWriter // Assim ResponseWriterWraper terá acesso a todos os métodos da interface http.ResponseWriter automaticamente.
	statusCode          int
}

func (rw *ResponseWriterWrapper) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func NewResponseWriterWrapper(w http.ResponseWriter) *ResponseWriterWrapper {
	return &ResponseWriterWrapper{w, http.StatusOK} // Status padrão
}

/// --- Middleware (agora com tracing!) ---

func prometheusMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wrappedWriter := NewResponseWriterWrapper(w)
		startTime := time.Now()

		activeConnections.Inc()

		// --- Criação do Span (OpenTelemetry) ---
		ctx := r.Context() // Obtém o contexto da requisição
		ctx, span := otel.Tracer("http-middleware").Start(
			ctx, // Contexto pai
			fmt.Sprintf("%s %s", r.Method, r.URL.Path), // Nome do span
			trace.WithAttributes(
				semconv.HTTPMethodKey.String(r.Method),                     // Use o código gerado
				semconv.HTTPRouteKey.String(r.URL.Path),                    // Use o código gerado
				semconv.NetPeerIPKey.String(r.RemoteAddr),                  // Use o código gerado
				semconv.SpanKindKey.String(string(semconv.SpanKindServer)), // Use o código gerado
			),
		)
		defer span.End()

		r = r.WithContext(ctx) // Propaga o contexto

		// --- Fim da Criação do Span ---

		next.ServeHTTP(wrappedWriter, r)

		activeConnections.Dec()

		duration := time.Since(startTime)
		statusCode := wrappedWriter.statusCode

		// --- Atualização do Span ---
		span.SetStatus(codeFromHTTPStatus(statusCode), "")
		span.SetAttributes(semconv.HTTPStatusCodeKey.Int(statusCode))

		// --- Métricas do Prometheus (sem alterações) ---
		httpRequestsTotal.With(prometheus.Labels{
			"path":   r.URL.Path,
			"method": r.Method,
			"status": fmt.Sprintf("%d", statusCode),
		}).Inc()

		httpRequestDuration.With(prometheus.Labels{
			"path":   r.URL.Path,
			"method": r.Method,
		}).Observe(duration.Seconds())

		// --- Logging (com traceID e spanID) ---
		log.WithFields(logrus.Fields{
			"path":        r.URL.Path,
			"method":      r.Method,
			"status_code": statusCode,
			"duration_ms": duration.Milliseconds(),
			"remote_addr": r.RemoteAddr,
			"user_agent":  r.UserAgent(),
			"trace_id":    span.SpanContext().TraceID().String(),
			"span_id":     span.SpanContext().SpanID().String(),
		}).Info("Requisição HTTP processada")
	})
}

// initTracerProvider inicializa o TracerProvider do OpenTelemetry, que cria os tracers!
func initTracerProvider() (*trace.TracerProvider, error) {
	ctx := context.Background()

	// Cria um novo exportador OTLP para enviar traces para o Jaeger.
	exporter, err := otlptracehttp.New(ctx,
		// Configurações do exportador.  Usamos as variáveis de ambiente
		// para tornar a configuração mais flexível.
		otlptracehttp.WithEndpoint(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")), // Endereço do coletor
		otlptracehttp.WithInsecure(),                                         // Sem TLS (para desenvolvimento)
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create exporter: %w", err)
	}

	// Define atributos que serão adicionados a todos os spans.
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(os.Getenv("OTEL_SERVICE_NAME")), // Nome do serviço
			attribute.String("environment", "development"),                // Exemplo de atributo adicional
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Cria o TracerProvider.
	tracerProvider := trace.NewTracerProvider(
		trace.WithBatcher(exporter), // Usa o exportador OTLP.
		trace.WithResource(res),     // Define os atributos.
	)
	return tracerProvider, nil
}

func init() {
	log.SetLevel(logrus.InfoLevel)
	log.SetFormatter(&logrus.JSONFormatter{})
	log.SetOutput(os.Stdout)

	// Inicializa o TracerProvider
	tp, err := initTracerProvider()
	if err != nil {
		log.Fatal(err) // Erro crítico: não podemos continuar sem tracing
	}

	// Define o TracerProvider global.  Isso permite que outras partes do
	// seu código acessem o tracer usando otel.Tracer("nome").
	otel.SetTracerProvider(tp)
}

func main() {
	var wg sync.WaitGroup
	wg.Add(1)

	// Inicia o servidor de métricas
	go func() {
		defer wg.Done()
		log.Info("Serviço de métricas iniciado na porta :2113")
		http.Handle("/metrics", promhttp.Handler())

		if err := http.ListenAndServe(":2113", nil); err != nil && err != http.ErrServerClosed {
			log.WithError(err).Fatal("Erro ao iniciar o servidor de métricas")
		}
	}()

	app := App{}
	err := app.Initialise()
	if err != nil {
		log.Fatal(err)
	}

	// Inicia a aplicação principal
	go func() {
		defer wg.Done() // Sinaliza quando a goroutine do servidor terminar
		log.Info("Aplicação iniciada na porta :10000")
		if err := http.ListenAndServe(":10000", app.Router); err != nil && err != http.ErrServerClosed {
			log.WithError(err).Fatal("Erro ao iniciar o servidor da aplicação")
		}
	}()

	wg.Wait()
}

// Função auxiliar para converter status code HTTP para o código do OpenTelemetry
func codeFromHTTPStatus(status int) codes.Code {
	switch {
	case status < 400:
		return codes.Ok // 1xx, 2xx, 3xx são considerados "Ok"
	case status >= 400 && status < 500:
		return codes.Error // 4xx são erros, mas não afetam o status do span "principal"
	case status >= 500:
		return codes.Error // 5xx são erros
	default:
		return codes.Unset // Status desconhecido
	}
}
