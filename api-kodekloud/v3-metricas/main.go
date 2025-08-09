package main

import (
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
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

// --- Middleware (modificado para incluir o histograma) ---
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
		}).Observe(duration.Seconds()) // Observe em segundos!

		log.WithFields(logrus.Fields{
			"path":        r.URL.Path,
			"method":      r.Method,
			"status_code": statusCode,
			"duration_ms": duration.Milliseconds(),
			"remote_addr": r.RemoteAddr,
			"user_agent":  r.UserAgent(),
		}).Info("Requisição HTTP processada")
	})
}

func init() {
	log.SetLevel(logrus.InfoLevel)
	log.SetFormatter(&logrus.JSONFormatter{})
	log.SetOutput(os.Stdout)
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
