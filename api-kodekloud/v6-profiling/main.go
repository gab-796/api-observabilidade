package main

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof" // Importa pprof para profiling
	"os"
	"sync"
	"time"

	"github.com/grafana/pyroscope-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"

	// Importando o pacote de trace
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	// trace pro logrus
	"github.com/uptrace/opentelemetry-go-extra/otellogrus"
)

// Função helper para adicionar trace_id e span_id aos logs
func logWithTrace(ctx context.Context) *logrus.Entry {
	entry := logrus.WithContext(ctx)
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		entry = entry.WithFields(logrus.Fields{
			"trace_id": span.SpanContext().TraceID().String(),
			"span_id":  span.SpanContext().SpanID().String(),
		})
	}
	return entry
}

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

// --- Middleware  ---
func prometheusMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wrappedWriter := NewResponseWriterWrapper(w)
		startTime := time.Now()

		activeConnections.Inc() // Incrementa no início da requisição
		next.ServeHTTP(wrappedWriter, r)
		activeConnections.Dec() // Decrementa no final da requisição

		duration := time.Since(startTime)
		statusCode := wrappedWriter.statusCode

		httpRequestsTotal.With(prometheus.Labels{
			"path":   r.URL.Path,
			"method": r.Method,
			"status": fmt.Sprintf("%d", statusCode),
		}).Inc()

		// Registra a duração no histograma
		httpRequestDuration.With(prometheus.Labels{
			"path":   r.URL.Path,
			"method": r.Method,
		}).Observe(duration.Seconds())

		// Verificação de debug para ver se há span ativo no contexto
		span := trace.SpanFromContext(r.Context())

		// Usando WithContext para garantir que o trace seja capturado - AGORA COM LOGRUS GLOBAL
		entry := logrus.WithContext(r.Context()).WithFields(logrus.Fields{
			"component":    "http_middleware",
			"path":        r.URL.Path,
			"method":      r.Method,
			"status_code": statusCode,
			"duration_ms": duration.Milliseconds(),
			"remote_addr": r.RemoteAddr,
			"user_agent":  r.UserAgent(),
		})

		// Adicionar manualmente trace_id e span_id se o span for válido
		if span.SpanContext().IsValid() {
			entry = entry.WithFields(logrus.Fields{
				"trace_id": span.SpanContext().TraceID().String(),
				"span_id":  span.SpanContext().SpanID().String(),
			})
		}

		entry.Info("Requisição HTTP processada")
	})
}

// --- init  ---
func init() {
	// Configuração do logger GLOBAL do logrus (seguindo a documentação do otellogrus)
	logrus.SetLevel(logrus.InfoLevel)
	logrus.SetFormatter(&logrus.JSONFormatter{
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "time",
			logrus.FieldKeyLevel: "level",
			logrus.FieldKeyMsg:   "msg",
		},
		TimestampFormat: time.RFC3339,
	})
	logrus.SetOutput(os.Stdout)

	// Hook do otellogrus será adicionado no main() após o tracer provider estar configurado
}

func main() {
	// --- Inicialização do Pyroscope para Profiling Contínuo ---
	pyroscopeURL := os.Getenv("PYROSCOPE_URL")
	if pyroscopeURL == "" {
		pyroscopeURL = "http://pyroscope:4040" // URL padrão do container
	}

	profiler, err := pyroscope.Start(pyroscope.Config{
		ApplicationName: "inventory-app",
		ServerAddress:   pyroscopeURL,
		Logger:          nil, // Remove StandardLogger para reduzir verbosidade
		Tags: map[string]string{
			"service":     "inventory-app",
			"environment": "local",
			"version":     "1.0.0",
		},
		ProfileTypes: []pyroscope.ProfileType{
			pyroscope.ProfileCPU,          // Profile de CPU
			pyroscope.ProfileAllocObjects, // Profile de alocação de objetos
			pyroscope.ProfileAllocSpace,   // Profile de espaço alocado
			pyroscope.ProfileInuseObjects, // Profile de objetos em uso
			pyroscope.ProfileInuseSpace,   // Profile de espaço em uso
			pyroscope.ProfileGoroutines,   // Profile de goroutines
		},
		UploadRate: 15 * time.Second, // Enviar profiles a cada 15 segundos
	})
	if err != nil {
		logrus.WithError(err).Warn("Erro ao inicializar Pyroscope - profiling desabilitado")
	} else {
		logrus.Info("Pyroscope profiling iniciado com sucesso")
		defer profiler.Stop()
	}

	// Define o endpoint do OTLP a partir de uma variável de ambiente
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
			endpoint = "otel-collector-service.api-app-go:4317" // Usado em ambiente k8s com ns api-app-go
		} else {
			endpoint = "otel-collector:4317" // Usado em ambiente Docker, com o container otel-collector e a porta grpc.
		}
	}
	// --- Inicialização do OpenTelemetry ---

	// 1. Criar o TracerProvider principal (para HTTP e outros)
	mainServiceName := "inventory-app" // Nome do Service do Tracer automatizado do pacote http.
	logrus.Infof("Tentando criar TracerProvider principal para o serviço: %s", mainServiceName)
	mainTp, err := newTracerProvider(endpoint, mainServiceName) //mainTP --> Main Tracer Provider, ou seja, pro pacote http.
	if err != nil {
		// Este Fatalf já existe e é crucial. Se ele ocorrer, os logs abaixo não aparecerão.
		logrus.WithError(err).Fatalf("Erro ao inicializar o TracerProvider principal (%s)", mainServiceName)
	}
	// Definir como o provider global
	otel.SetTracerProvider(mainTp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	logrus.Infof("TracerProvider principal (%s) configurado como global.", mainServiceName)

	// Adiciona o hook do otellogrus GLOBAL exatamente como na documentação
	logrus.AddHook(otellogrus.NewHook(
		otellogrus.WithLevels(
			logrus.PanicLevel,
			logrus.FatalLevel,
			logrus.ErrorLevel,
			logrus.WarnLevel,
			logrus.InfoLevel,
			logrus.DebugLevel,
			logrus.TraceLevel,
		),
	))

	// Log de teste para verificar se o hook está funcionando
	logrus.Info("Hook do otellogrus adicionado - testando correlação de traces")

	// Função de shutdown para o provider principal
	defer func() {
		logrus.Infof("Desligando o TracerProvider principal (%s)...", mainServiceName)
		if err := mainTp.Shutdown(context.Background()); err != nil {
			logrus.WithError(err).Errorf("Erro ao desligar o TracerProvider principal (%s)", mainServiceName)
		} else {
			logrus.Infof("TracerProvider principal (%s) desligado.", mainServiceName)
		}
	}()

	// 2. Criar o TracerProvider secundário (para SQL)
	sqlServiceName := "my-inventory-mysql" // Nome do service que vai aparecer no Tempo relacionado ao BD MySQL.
	logrus.Infof("Tentando criar TracerProvider para SQL para o serviço: %s", sqlServiceName)
	sqlTp, err := newTracerProvider(endpoint, sqlServiceName) // sqlTp --> TracerProvider pro SQL.
	if err != nil {
		logrus.WithError(err).Fatalf("Erro ao inicializar o TracerProvider do SQL (%s)", sqlServiceName)
	}
	logrus.Infof("TracerProvider para SQL (%s) criado.", sqlServiceName)

	// Função de shutdown para o provider do SQL
	defer func() {
		logrus.Infof("Desligando o TracerProvider do SQL (%s)...", sqlServiceName)
		if err := sqlTp.Shutdown(context.Background()); err != nil {
			logrus.WithError(err).Errorf("Erro ao desligar o TracerProvider do SQL (%s)", sqlServiceName)
		} else {
			logrus.Infof("TracerProvider do SQL (%s) desligado.", sqlServiceName)
		}
	}()
	// --- Fim da Inicialização do OpenTelemetry ---

	var wg sync.WaitGroup
	wg.Add(2) // Incrementando para 2 goroutines

	// Inicia o servidor de métricas
	go func() {
		defer wg.Done()
		metricsAddr := ":2113"
		logrus.Infof("Serviço de métricas iniciado na porta %s", metricsAddr)
		muxMetrics := http.NewServeMux()
		muxMetrics.Handle("/metrics", promhttp.Handler()) // Use um mux dedicado para métricas
		if err := http.ListenAndServe(metricsAddr, muxMetrics); err != nil && err != http.ErrServerClosed {
			logrus.WithError(err).Fatalf("Erro ao iniciar o servidor de métricas na porta %s", metricsAddr)
		}
	}()

	// Inicializa a aplicação, passando o TracerProvider do SQL
	app := App{}
	// Passa o sqlTp para a inicialização da App
	err = app.Initialise(sqlTp)
	if err != nil {
		logrus.WithError(err).Fatal("Erro fatal ao inicializar a aplicação")
	}

	// Inicia a aplicação principal
	go func() {
		defer wg.Done()
		appAddr := ":10000"
		logrus.Infof("Aplicação principal iniciando na porta %s", appAddr)
		// O otelhttp.NewHandler usará o TracerProvider GLOBAL (mainTp)
		handler := otelhttp.NewHandler(app.Router, mainServiceName) // Usa o nome do serviço principal aqui
		if err := http.ListenAndServe(appAddr, handler); err != nil && err != http.ErrServerClosed {
			logrus.WithError(err).Fatalf("Erro ao iniciar o servidor da aplicação na porta %s", appAddr)
		}
	}()

	logrus.Info("Servidores iniciados. Aguardando...")
	wg.Wait()
	logrus.Info("Todos os servidores foram encerrados.")
}

// newTracerProvider: Função que cria um TracerProvider com um nome de serviço específico.
func newTracerProvider(endpoint string, serviceName string) (*sdktrace.TracerProvider, error) {
	ctx := context.Background() // Contexto base para operações que não são a conexão inicial

	// Usa grpc.NewClient conforme sugerido pela IDE.
	// A conexão pode ocorrer em background. Erros de conexão podem aparecer
	// mais tarde, durante a exportação dos spans.
	conn, err := grpc.NewClient(endpoint,
		// grpc.WithTransportCredentials() ainda é necessário para configurar TLS ou insecure.
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		// Erros aqui são geralmente de configuração das opções, não da conexão em si.
		return nil, fmt.Errorf("falha ao configurar gRPC client para OTLP exporter em %s para o serviço %s: %w", endpoint, serviceName, err)
	}
	logrus.Infof("Conexão gRPC com OTLP exporter (%s) estabelecida para o serviço %s", endpoint, serviceName)

	// Cria o exporter OTLP/gRPC
	traceExporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		// Tenta fechar a conexão se a criação do exporter falhar
		if closeErr := conn.Close(); closeErr != nil {
			logrus.WithError(closeErr).Warnf("Erro ao fechar conexão gRPC após falha na criação do exporter para %s", serviceName)
		}
		return nil, fmt.Errorf("falha ao criar OTLP trace exporter para o serviço %s: %w", serviceName, err)
	}
	logrus.Infof("OTLP trace exporter criado para o serviço %s", serviceName)

	// Define o recurso (Resource) com o nome do serviço e o Schema URL.
	// Este é o bloco que você indicou, agora corrigido:
	res, err := resource.New(ctx,
		resource.WithSchemaURL(semconv.SchemaURL), // <<< CORRIGIDO: Usa WithSchemaURL para definir o schema
		resource.WithAttributes( // <<< CORRIGIDO: Apenas os atributos KeyValue aqui
			semconv.ServiceNameKey.String(serviceName), // Define o nome do serviço
			attribute.String("environment", "local"),   // Mantendo seu atributo de ambiente
			// semconv.ServiceVersionKey.String("1.0.0"), // Exemplo de outro atributo
		),
	)
	if err != nil {
		conn.Close() // Fecha a conexão se a criação do recurso falhar.
		return nil, fmt.Errorf("falha ao criar recurso para o serviço %s: %w", serviceName, err)
	}
	logrus.Infof("Recurso OpenTelemetry criado para o serviço %s", serviceName)

	// Cria o TracerProvider com o BatchSpanProcessor e o Recurso
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter), // Envia spans em batches
		sdktrace.WithResource(res),          // Associa o recurso ao provider
		// Você pode adicionar outros samplers ou span processors aqui se necessário
		// sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	logrus.Infof("TracerProvider (%s) criado com sucesso.", serviceName)

	// Nota: A conexão gRPC (conn) não deve ser fechada aqui,
	// pois o exporter a utiliza. O Shutdown do TracerProvider cuidará disso.

	return tp, nil
}
