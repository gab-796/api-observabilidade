package observability

import (
	"context"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/trace"
)

// LoggingManager gerencia configuração de logs estruturados
type LoggingManager struct {
	config LoggingConfig
}

// NewLoggingManager cria uma nova instância
func NewLoggingManager() *LoggingManager {
	return &LoggingManager{}
}

// Initialize configura o sistema de logging
func (lm *LoggingManager) Initialize(config LoggingConfig) error {
	lm.config = config

	// Configurar nível de log
	level, err := logrus.ParseLevel(config.Level)
	if err != nil {
		level = logrus.InfoLevel
		logrus.Warnf("Nível de log inválido '%s', usando 'info'", config.Level)
	}
	logrus.SetLevel(level)

	// Configurar formato
	if config.Format == "json" {
		logrus.SetFormatter(&logrus.JSONFormatter{
			FieldMap: logrus.FieldMap{
				logrus.FieldKeyTime:  "time",
				logrus.FieldKeyLevel: "level",
				logrus.FieldKeyMsg:   "msg",
			},
			TimestampFormat: time.RFC3339,
		})
	} else {
		logrus.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: time.RFC3339,
		})
	}

	// Configurar saída
	logrus.SetOutput(os.Stdout)

	logrus.WithFields(logrus.Fields{
		"level":  config.Level,
		"format": config.Format,
	}).Info("Sistema de logging configurado")

	return nil
}

// LogWithTrace adiciona informações de trace ao log entry
func (lm *LoggingManager) LogWithTrace(ctx context.Context) *logrus.Entry {
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

// LogHTTPRequest cria log estruturado para requisições HTTP
func (lm *LoggingManager) LogHTTPRequest(ctx context.Context, method, path string, statusCode int, duration time.Duration, userAgent, remoteAddr string) {
	entry := lm.LogWithTrace(ctx).WithFields(logrus.Fields{
		"component":    "http_middleware",
		"method":       method,
		"path":         path,
		"status_code":  statusCode,
		"duration_ms":  duration.Milliseconds(),
		"user_agent":   userAgent,
		"remote_addr":  remoteAddr,
	})

	if statusCode >= 500 {
		entry.Error("Requisição HTTP processada com erro")
	} else if statusCode >= 400 {
		entry.Warn("Requisição HTTP processada com aviso")
	} else {
		entry.Info("Requisição HTTP processada com sucesso")
	}
}

// LogDatabaseOperation cria log estruturado para operações de banco
func (lm *LoggingManager) LogDatabaseOperation(ctx context.Context, operation string, productID int, duration time.Duration, err error) {
	entry := lm.LogWithTrace(ctx).WithFields(logrus.Fields{
		"component":   "database",
		"operation":   operation,
		"duration_ms": duration.Milliseconds(),
	})

	if productID > 0 {
		entry = entry.WithField("product_id", productID)
	}

	if err != nil {
		entry.WithError(err).Error("Operação de banco de dados falhou")
	} else {
		entry.Debug("Operação de banco de dados concluída")
	}
}

// LogApplicationEvent cria log estruturado para eventos da aplicação
func (lm *LoggingManager) LogApplicationEvent(ctx context.Context, event, component string, fields map[string]interface{}) {
	entry := lm.LogWithTrace(ctx).WithFields(logrus.Fields{
		"component": component,
		"event":     event,
	})

	// Adicionar campos customizados
	for key, value := range fields {
		entry = entry.WithField(key, value)
	}

	entry.Info("Evento da aplicação")
}

// Shutdown finaliza o sistema de logging
func (lm *LoggingManager) Shutdown() error {
	logrus.Info("Sistema de logging finalizado")
	return nil
}
