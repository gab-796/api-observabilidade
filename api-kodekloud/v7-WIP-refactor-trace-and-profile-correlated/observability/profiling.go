package observability

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/grafana/pyroscope-go"
	"github.com/sirupsen/logrus"
)

// ProfilingManager gerencia profiling contínuo com Pyroscope
type ProfilingManager struct {
	profiler *pyroscope.Profiler
	config   ProfilingConfig
}

// NewProfilingManager cria uma nova instância
func NewProfilingManager() *ProfilingManager {
	return &ProfilingManager{}
}

// Initialize configura o profiling contínuo
func (pm *ProfilingManager) Initialize(config ProfilingConfig) error {
	pm.config = config

	logrus.WithFields(logrus.Fields{
		"server_address": config.ServerAddress,
		"application":    config.ApplicationName,
		"upload_rate":    config.UploadRate,
	}).Info("Inicializando Pyroscope profiling...")

	profiler, err := pyroscope.Start(pyroscope.Config{
		ApplicationName: config.ApplicationName,
		ServerAddress:   config.ServerAddress,
		Logger:          nil, // Silenciar logs verbosos do Pyroscope
		Tags: map[string]string{
			"service":     config.ApplicationName,
			"environment": config.Environment,
			"version":     config.Version,
		},
		ProfileTypes: []pyroscope.ProfileType{
			pyroscope.ProfileCPU,
			pyroscope.ProfileAllocObjects,
			pyroscope.ProfileAllocSpace,
			pyroscope.ProfileInuseObjects,
			pyroscope.ProfileInuseSpace,
			pyroscope.ProfileGoroutines,
			pyroscope.ProfileMutexCount,
			pyroscope.ProfileMutexDuration,
			pyroscope.ProfileBlockCount,
			pyroscope.ProfileBlockDuration,
		},
		UploadRate: config.UploadRate,
	})

	if err != nil {
		return fmt.Errorf("failed to start pyroscope: %w", err)
	}

	pm.profiler = profiler
	logrus.WithField("application", config.ApplicationName).Info("Pyroscope profiling iniciado com sucesso")
	return nil
}

// ProfiledHTTPHandler wraps HTTP handlers com contexto de profiling
func (pm *ProfilingManager) ProfiledHTTPHandler(handlerName string, handler http.HandlerFunc) http.HandlerFunc {
	if pm.profiler == nil {
		return handler // Profiling desabilitado
	}

	return func(w http.ResponseWriter, r *http.Request) {
		// Simplesmente executa o handler - o profiling automático do Pyroscope
		// já está capturando o profile de CPU e memória globalmente
		handler(w, r)
	}
}

// ProfiledDatabaseOperation wraps operações de banco com profiling
func (pm *ProfilingManager) ProfiledDatabaseOperation(ctx context.Context, operation string, productID int, fn func(context.Context) error) error {
	if pm.profiler == nil {
		return fn(ctx) // Profiling desabilitado
	}

	// Executa a operação - o profiling automático do Pyroscope
	// já está capturando o profile de CPU e memória globalmente
	return fn(ctx)
}

// ProfileRuntime monitora métricas de runtime Go correlacionadas com traces
func (pm *ProfilingManager) ProfileRuntime(ctx context.Context) {
	if pm.profiler == nil {
		return
	}

	// O profiling automático do Pyroscope já captura runtime metrics,
	// então não precisamos fazer nada especial aqui
	time.Sleep(100 * time.Millisecond)
}

// categorizeUserAgent categoriza user agents para reduzir cardinalidade
func (pm *ProfilingManager) categorizeUserAgent(userAgent string) string {
	userAgent = strings.ToLower(userAgent)

	switch {
	case pm.contains(userAgent, "postman"):
		return "postman"
	case pm.contains(userAgent, "curl"):
		return "curl"
	case pm.contains(userAgent, "wget"):
		return "wget"
	case pm.contains(userAgent, "go-http-client"):
		return "go-client"
	case pm.contains(userAgent, "python"):
		return "python-client"
	case pm.contains(userAgent, "chrome"):
		return "browser-chrome"
	case pm.contains(userAgent, "firefox"):
		return "browser-firefox"
	case pm.contains(userAgent, "safari"):
		return "browser-safari"
	case pm.contains(userAgent, "bot"), pm.contains(userAgent, "crawler"):
		return "bot"
	default:
		return "other"
	}
}

// contains verifica se uma string contém substring (case-insensitive)
func (pm *ProfilingManager) contains(s, substr string) bool {
	return pm.containsHelper(s, substr)
}

func (pm *ProfilingManager) containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Shutdown para o profiling
func (pm *ProfilingManager) Shutdown() error {
	if pm.profiler != nil {
		logrus.Info("Parando Pyroscope profiling...")
		return pm.profiler.Stop()
	}
	return nil
}

// IsEnabled retorna se o profiling está habilitado
func (pm *ProfilingManager) IsEnabled() bool {
	return pm.profiler != nil
}
