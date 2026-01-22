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
			Buckets: prometheus.DefBuckets, // Usa os buckets padrão do Prometheus
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
			"path":   r.URL.Path, // ponto de atenção --> explode em cardinalidade no Prometheus.
			"method": r.Method,
			"status": fmt.Sprintf("%d", statusCode),
		}).Inc()

		// Registra a duração no histograma
		httpRequestDuration.With(prometheus.Labels{
			"path":   r.URL.Path,
			"method": r.Method,
		}).Observe(duration.Seconds())

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
/*
Esta função implementa um middleware de observabilidade que instrumenta automaticamente
todas as requisições HTTP da aplicação,coletando métricas Prometheus e logs estruturados
sem modificar os handlers existentes.
Ele existe para contar conexões ativas, logar com o logrus e adicionar métricas customizadas.

O middleware nativo do Prometheus é promhttp.InstrumentHandler*, mas ele só mede requisições HTTP.
Só conseguiria fornecer métricas de latência, quantidade de requests e status.
(promhttp.InstrumentHandlerDuration, promhttp.InstrumentHandlerCounter, etc)

"path":   r.URL.Path, --> Só esta dando certo pq tenho poucos itens na minha tabela.
Se tivesse muitos, o ideal seria usar algo como:
"path":   strings.Split(r.URL.Path, "/")[1], // Pega só a primeira parte do path
Assim evitaria a explosão de cardinalidade no Prometheus.


O bloco de httpRequestDuration cria as seguintes séries:
- http_request_duration_seconds_count → total de requisições observadas
(igual ao número de vezes que você chamou Observe()).

- http_request_duration_seconds_sum → soma total dos segundos.

- http_request_duration_seconds_bucket{le="0.5"} → quantas requests ficaram até 0.5s, e assim por diante.
*/


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
/*
O servidor de métricas é uma go routine anônima, assim como a aplicação principal.
Ele fica apartado do servidor principal da aplicação. Isso é proposital, pois permite que o
servidor de métricas continue funcionando mesmo se o servidor da app principal falhar ou for reiniciado.
Quando jogarmos pro k8s, a app terá o service principal, e o servidor de métricas usará seu service próprio.
É aqui que escolhemos o path onde as métricas serão expostas: /metrics,
via handler oficial do Prometheus: promhttp.Handler()
*/
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

/*

wg.Wait() --> Bloqueia a função main() até que ambas as goroutines
(servidor de métricas e servidor da aplicação) finalizem.



O pacote "github.com/prometheus/client_golang/prometheus" é a biblioteca oficial do Prometheus para Go,
fornecendo os tipos fundamentais de métricas como Counter, Gauge, Histogram e Summary.
Este pacote permite criar métricas customizadas, definir labels dimensionais,
e configurar registries para organizar as métricas da aplicação.
É o núcleo do sistema de instrumentação,
oferecendo APIs de baixo nível para controle preciso sobre como as métricas são coletadas e expostas.

O "github.com/prometheus/client_golang/prometheus/promauto" simplifica significativamente o processo de criação
e registro automático de métricas.
Em vez de criar métricas e registrá-las manualmente no registry padrão,
o promauto faz isso automaticamente quando você usa funções como NewCounter(), NewGauge() ou NewHistogram().
Isso reduz boilerplate code e elimina erros comuns de esquecimento de registro,
tornando a instrumentação mais limpa e menos propensa a bugs.


O pacote "github.com/prometheus/client_golang/prometheus/promhttp" fornece handlers HTTP prontos
para expor métricas no formato esperado pelo Prometheus.
Principalmente através da função promhttp.Handler(), que cria um endpoint /metrics automaticamente formatado.
Este handler serializa todas as métricas registradas no formato de texto do Prometheus,
permitindo que o servidor Prometheus faça scraping dos dados periodicamente sem necessidade de implementação manual.

"github.com/sirupsen/logrus" é uma biblioteca de logging estruturado amplamente adotada na comunidade Go.
Diferente do pacote log padrão,
o Logrus permite logging em formato JSON, níveis de log configuráveis (Debug, Info, Warning, Error), e fields estruturados 
que facilitam parsing e análise em sistemas como ELK Stack ou Loki.
Complementa as métricas Prometheus fornecendo contexto detalhado sobre eventos específicos da aplicação.
*/