package observability

import "time"

// Config centraliza todas as configurações de observabilidade
type Config struct {
	Tracing   TracingConfig
	Metrics   MetricsConfig
	Profiling ProfilingConfig
	Logging   LoggingConfig
}

// TracingConfig configurações para OpenTelemetry traces
type TracingConfig struct {
	Endpoint        string
	ServiceName     string
	Environment     string
	Version         string
	SQLServiceName  string
	Enabled         bool
}

// MetricsConfig configurações para Prometheus metrics
type MetricsConfig struct {
	Port     string
	Path     string
	Enabled  bool
}

// ProfilingConfig configurações para Pyroscope profiling
type ProfilingConfig struct {
	ServerAddress   string
	ApplicationName string
	Environment     string
	Version         string
	UploadRate      time.Duration
	Enabled         bool
}

// LoggingConfig configurações para logging estruturado
type LoggingConfig struct {
	Level  string
	Format string // json ou text
}
