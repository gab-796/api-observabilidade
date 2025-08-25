package observability

import (
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

// MetricsManager gerencia métricas Prometheus
type MetricsManager struct {
	config Config
	server *http.Server
	
	// Métricas HTTP
	HTTPRequestsTotal    *prometheus.CounterVec
	HTTPRequestDuration  *prometheus.HistogramVec
	ActiveConnections    prometheus.Gauge
	
	// Métricas da aplicação
	ProductsInDB   prometheus.Gauge
	SQLErrorsTotal prometheus.Counter
}

// NewMetricsManager cria uma nova instância
func NewMetricsManager() *MetricsManager {
	return &MetricsManager{}
}

// Initialize configura as métricas Prometheus
func (mm *MetricsManager) Initialize(config MetricsConfig) error {
	mm.config = Config{Metrics: config}

	logrus.WithFields(logrus.Fields{
		"port": config.Port,
		"path": config.Path,
	}).Info("Inicializando métricas Prometheus...")

	// Definir métricas HTTP
	mm.HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Número total de requisições HTTP recebidas",
		},
		[]string{"path", "method", "status"},
	)

	mm.HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duração das requisições HTTP em segundos",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"path", "method"},
	)

	mm.ActiveConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "http_active_connections",
		Help: "Número de conexões HTTP ativas",
	})

	// Métricas da aplicação
	mm.ProductsInDB = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "products_in_db",
		Help: "Número de produtos no banco de dados",
	})

	mm.SQLErrorsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "sql_errors_total",
		Help: "Número total de erros de SQL",
	})

	logrus.Info("Métricas Prometheus configuradas com sucesso")
	return nil
}

// StartServer inicia o servidor de métricas
func (mm *MetricsManager) StartServer(addr string) error {
	mux := http.NewServeMux()
	mux.Handle(mm.config.Metrics.Path, promhttp.Handler())

	mm.server = &http.Server{
		Addr:         ":" + mm.config.Metrics.Port,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	logrus.WithFields(logrus.Fields{
		"addr": mm.server.Addr,
		"path": mm.config.Metrics.Path,
	}).Info("Iniciando servidor de métricas Prometheus")

	if err := mm.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("metrics server failed: %w", err)
	}

	return nil
}

// PrometheusMiddleware wrapper para capturar o status code
type ResponseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (rw *ResponseWriterWrapper) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func NewResponseWriterWrapper(w http.ResponseWriter) *ResponseWriterWrapper {
	return &ResponseWriterWrapper{w, http.StatusOK}
}

// CreatePrometheusMiddleware cria middleware para coletar métricas HTTP
func (mm *MetricsManager) CreatePrometheusMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			wrappedWriter := NewResponseWriterWrapper(w)
			startTime := time.Now()

			mm.ActiveConnections.Inc()
			defer mm.ActiveConnections.Dec()

			next.ServeHTTP(wrappedWriter, r)

			duration := time.Since(startTime)
			statusCode := wrappedWriter.statusCode

			mm.HTTPRequestsTotal.With(prometheus.Labels{
				"path":   r.URL.Path,
				"method": r.Method,
				"status": fmt.Sprintf("%d", statusCode),
			}).Inc()

			mm.HTTPRequestDuration.With(prometheus.Labels{
				"path":   r.URL.Path,
				"method": r.Method,
			}).Observe(duration.Seconds())

			logrus.WithFields(logrus.Fields{
				"component":    "http_middleware",
				"path":        r.URL.Path,
				"method":      r.Method,
				"status_code": statusCode,
				"duration_ms": duration.Milliseconds(),
				"remote_addr": r.RemoteAddr,
				"user_agent":  r.UserAgent(),
			}).Info("Requisição HTTP processada")
		})
	}
}

// UpdateProductCount atualiza métrica de produtos no banco
func (mm *MetricsManager) UpdateProductCount(count int) {
	mm.ProductsInDB.Set(float64(count))
}

// IncrementSQLErrors incrementa contador de erros SQL
func (mm *MetricsManager) IncrementSQLErrors() {
	mm.SQLErrorsTotal.Inc()
}

// Shutdown encerra o servidor de métricas
func (mm *MetricsManager) Shutdown() error {
	if mm.server == nil {
		return nil
	}

	logrus.Info("Desligando servidor de métricas...")
	return mm.server.Close()
}
