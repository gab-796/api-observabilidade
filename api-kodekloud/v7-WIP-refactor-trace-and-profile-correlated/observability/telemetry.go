package observability

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// TelemetryManager coordena todos os pilares de observabilidade
type TelemetryManager struct {
	Tracing   *TracingManager
	Metrics   *MetricsManager
	Profiling *ProfilingManager
	Logging   *LoggingManager
	config    Config
}

// NewTelemetryManager cria uma nova instância do manager
func NewTelemetryManager() *TelemetryManager {
	return &TelemetryManager{
		Tracing:   NewTracingManager(),
		Metrics:   NewMetricsManager(),
		Profiling: NewProfilingManager(),
		Logging:   NewLoggingManager(),
	}
}

// Initialize configura toda a stack de observabilidade
func (tm *TelemetryManager) Initialize(ctx context.Context, config Config) error {
	tm.config = config

	logrus.Info("Inicializando stack de observabilidade...")

	var wg sync.WaitGroup
	errChan := make(chan error, 4)

	// Inicializar cada pilar em paralelo
	wg.Add(4)

	// Logging primeiro (para ter logs dos outros componentes)
	go func() {
		defer wg.Done()
		if err := tm.Logging.Initialize(config.Logging); err != nil {
			errChan <- fmt.Errorf("logging init failed: %w", err)
		}
	}()

	// Aguardar logging estar pronto
	time.Sleep(100 * time.Millisecond)

	// Tracing
	go func() {
		defer wg.Done()
		if config.Tracing.Enabled {
			if err := tm.Tracing.Initialize(ctx, config.Tracing); err != nil {
				errChan <- fmt.Errorf("tracing init failed: %w", err)
			}
		}
	}()

	// Metrics
	go func() {
		defer wg.Done()
		if config.Metrics.Enabled {
			if err := tm.Metrics.Initialize(config.Metrics); err != nil {
				errChan <- fmt.Errorf("metrics init failed: %w", err)
			}
		}
	}()

	// Profiling (após tracing para correlação)
	go func() {
		defer wg.Done()
		if config.Profiling.Enabled {
			if err := tm.Profiling.Initialize(config.Profiling); err != nil {
				errChan <- fmt.Errorf("profiling init failed: %w", err)
			}
		}
	}()

	wg.Wait()
	close(errChan)

	// Verificar erros
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	logrus.Info("Stack de observabilidade inicializada com sucesso")
	return nil
}

// Shutdown encerra todos os componentes gracefully
func (tm *TelemetryManager) Shutdown(ctx context.Context) error {
	logrus.Info("Desligando stack de observabilidade...")

	var errs []error

	// Shutdown em ordem reversa da inicialização
	if tm.Profiling != nil && tm.config.Profiling.Enabled {
		if err := tm.Profiling.Shutdown(); err != nil {
			errs = append(errs, fmt.Errorf("profiling shutdown: %w", err))
		}
	}

	if tm.Metrics != nil && tm.config.Metrics.Enabled {
		if err := tm.Metrics.Shutdown(); err != nil {
			errs = append(errs, fmt.Errorf("metrics shutdown: %w", err))
		}
	}

	if tm.Tracing != nil && tm.config.Tracing.Enabled {
		if err := tm.Tracing.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("tracing shutdown: %w", err))
		}
	}

	if tm.Logging != nil {
		if err := tm.Logging.Shutdown(); err != nil {
			errs = append(errs, fmt.Errorf("logging shutdown: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("shutdown errors: %v", errs)
	}

	logrus.Info("Stack de observabilidade desligada com sucesso")
	return nil
}

// GetDefaultConfig retorna uma configuração padrão
func GetDefaultConfig() Config {
	return Config{
		Tracing: TracingConfig{
			Endpoint:       "otel-collector:4317",
			ServiceName:    "inventory-app",
			SQLServiceName: "my-inventory-mysql",
			Environment:    "local",
			Version:        "1.0.0",
			Enabled:        true,
		},
		Metrics: MetricsConfig{
			Port:    "2113",
			Path:    "/metrics",
			Enabled: true,
		},
		Profiling: ProfilingConfig{
			ServerAddress:   "http://pyroscope:4040",
			ApplicationName: "inventory-app",
			Environment:     "local",
			Version:         "1.0.0",
			UploadRate:      15 * time.Second,
			Enabled:         true,
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
		},
	}
}
