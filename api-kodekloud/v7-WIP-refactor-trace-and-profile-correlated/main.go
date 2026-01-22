package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"example.com/my-inventory/observability"
)

func main() {
	ctx := context.Background()

	// Configurar observabilidade a partir de variáveis de ambiente
	config := getObservabilityConfig()

	// Inicializar stack de observabilidade
	telemetry := observability.NewTelemetryManager()
	if err := telemetry.Initialize(ctx, config); err != nil {
		log.Fatalf("Falha ao inicializar observabilidade: %v", err)
	}
	defer func() {
		if err := telemetry.Shutdown(ctx); err != nil {
			log.Printf("Erro ao desligar observabilidade: %v", err)
		}
	}()

	// Inicializar aplicação
	app := App{}
	if err := app.Initialise(telemetry.Tracing.GetSQLProvider(), telemetry.Profiling, telemetry.Metrics); err != nil {
		log.Fatalf("Falha ao inicializar aplicação: %v", err)
	}

	// Configurar middlewares da aplicação
	if telemetry.Metrics != nil {
		app.Router.Use(telemetry.Metrics.CreatePrometheusMiddleware())
	}

	// Servidor concorrente
	var wg sync.WaitGroup
	wg.Add(2)

	// Servidor de métricas
	go func() {
		defer wg.Done()
		if err := telemetry.Metrics.StartServer(":2113"); err != nil {
			log.Printf("Servidor de métricas encerrado: %v", err)
		}
	}()

	// Servidor principal da aplicação
	go func() {
		defer wg.Done()
		handler := otelhttp.NewHandler(app.Router, config.Tracing.ServiceName)

		server := &http.Server{
			Addr:         ":10000",
			Handler:      handler,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
		}

		log.Printf("Servidor principal iniciando em %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Servidor principal encerrado: %v", err)
		}
	}()

	log.Println("Servidores iniciados. Aguardando...")
	wg.Wait()
	log.Println("Aplicação encerrada.")
}

// getObservabilityConfig constrói configuração a partir de variáveis de ambiente
func getObservabilityConfig() observability.Config {
	config := observability.GetDefaultConfig()

	// Sobrescrever com variáveis de ambiente se disponíveis
	if endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"); endpoint != "" {
		config.Tracing.Endpoint = endpoint
	} else {
		// Detectar ambiente automaticamente
		if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
			config.Tracing.Endpoint = "otel-collector-service.api-app-go:4317"
		} else {
			config.Tracing.Endpoint = "otel-collector:4317"
		}
	}

	if pyroscopeURL := os.Getenv("PYROSCOPE_URL"); pyroscopeURL != "" {
		config.Profiling.ServerAddress = pyroscopeURL
	}

	if level := os.Getenv("LOG_LEVEL"); level != "" {
		config.Logging.Level = level
	}

	// Desabilitar componentes via env vars se necessário
	if os.Getenv("DISABLE_TRACING") == "true" {
		config.Tracing.Enabled = false
	}

	if os.Getenv("DISABLE_PROFILING") == "true" {
		config.Profiling.Enabled = false
	}

	if os.Getenv("DISABLE_METRICS") == "true" {
		config.Metrics.Enabled = false
	}

	return config
}

// getEnv retorna valor da variável de ambiente ou valor padrão
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

