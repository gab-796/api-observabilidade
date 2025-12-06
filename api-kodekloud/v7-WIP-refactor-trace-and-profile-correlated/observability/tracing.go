package observability

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/sirupsen/logrus"
	"github.com/uptrace/opentelemetry-go-extra/otellogrus"
)
/*
A SDK da uptrace foi escolhida pela facilidade de capturar trace com log
A Uptrace desenvolveu isso focando no backend deles.
É mais acoplado ao modelo de tracing, não ao de logs nativos OTel.
O ideal seria ter adotado o padrão otel, ou seja, usar esse aqui:
"github.com/uptrace/opentelemetry-go-extra/otellogrus"

*/


// TracingManager gerencia configuração e lifecycle do OpenTelemetry
type TracingManager struct {
	mainProvider *sdktrace.TracerProvider
	sqlProvider  *sdktrace.TracerProvider
	config       TracingConfig
}

// NewTracingManager cria uma nova instância
func NewTracingManager() *TracingManager {
	return &TracingManager{}
}

// Initialize configura o OpenTelemetry tracing
func (tm *TracingManager) Initialize(ctx context.Context, config TracingConfig) error {
	tm.config = config

	logrus.WithFields(logrus.Fields{
		"endpoint":    config.Endpoint,
		"service":     config.ServiceName,
		"sql_service": config.SQLServiceName,
	}).Info("Inicializando OpenTelemetry tracing...")

	// Criar TracerProvider principal
	mainTp, err := tm.createTracerProvider(ctx, config.Endpoint, config.ServiceName)
	if err != nil {
		return fmt.Errorf("failed to create main tracer provider: %w", err)
	}
	tm.mainProvider = mainTp

	// Criar TracerProvider para SQL
	sqlTp, err := tm.createTracerProvider(ctx, config.Endpoint, config.SQLServiceName)
	if err != nil {
		return fmt.Errorf("failed to create SQL tracer provider: %w", err)
	}
	tm.sqlProvider = sqlTp

	// Configurar como global
	otel.SetTracerProvider(mainTp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Adicionar hook do otellogrus para correlação de logs
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

	logrus.WithField("service", config.ServiceName).Info("OpenTelemetry tracing configurado com sucesso")
	return nil
}

// createTracerProvider cria um TracerProvider para um serviço específico
func (tm *TracingManager) createTracerProvider(ctx context.Context, endpoint, serviceName string) (*sdktrace.TracerProvider, error) {
	// Conectar via gRPC
	conn, err := grpc.NewClient(endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection to %s: %w", endpoint, err)
	}

	logrus.WithFields(logrus.Fields{
		"endpoint": endpoint,
		"service":  serviceName,
	}).Debug("Conexão gRPC estabelecida com OTLP exporter")

	// Criar exporter OTLP/gRPC
	traceExporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create OTLP trace exporter: %w", err)
	}

	// Criar resource com metadados do serviço
	res, err := resource.New(ctx,
		resource.WithSchemaURL(semconv.SchemaURL),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
			semconv.ServiceVersionKey.String(tm.config.Version),
			semconv.DeploymentEnvironmentKey.String(tm.config.Environment),
			attribute.String("telemetry.sdk.name", "opentelemetry"),
			attribute.String("telemetry.sdk.language", "go"),
		),
	)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create resource for %s: %w", serviceName, err)
	}

	// Criar TracerProvider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()), // Para desenvolvimento
	)

	logrus.WithField("service", serviceName).Debug("TracerProvider criado com sucesso")
	return tp, nil
}

// GetMainProvider retorna o TracerProvider principal
func (tm *TracingManager) GetMainProvider() *sdktrace.TracerProvider {
	return tm.mainProvider
}

// GetSQLProvider retorna o TracerProvider para SQL
func (tm *TracingManager) GetSQLProvider() *sdktrace.TracerProvider {
	return tm.sqlProvider
}

// Shutdown encerra gracefully os TracerProviders
func (tm *TracingManager) Shutdown(ctx context.Context) error {
	logrus.Info("Desligando OpenTelemetry tracing...")

	var errs []error

	if tm.sqlProvider != nil {
		logrus.Debug("Desligando SQL TracerProvider...")
		if err := tm.sqlProvider.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("SQL tracer shutdown: %w", err))
		}
	}

	if tm.mainProvider != nil {
		logrus.Debug("Desligando main TracerProvider...")
		if err := tm.mainProvider.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("main tracer shutdown: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("tracing shutdown errors: %v", errs)
	}

	logrus.Info("OpenTelemetry tracing desligado com sucesso")
	return nil
}
