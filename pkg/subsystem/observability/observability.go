// Package observability provides a subsystem for building observability components.
package observability

import (
	"log/slog"
	"time"

	"miren.dev/runtime/metrics"
	"miren.dev/runtime/observability"
	"miren.dev/runtime/pkg/subsystem"
)

// Config holds required configuration for the observability subsystem.
type Config struct {
	Log              *slog.Logger
	VictoriaLogsAddr string        // VictoriaLogs address (e.g., "localhost:9428")
	Timeout          time.Duration // Request timeout (0 uses default of 30s)

	// Optional: metrics writer/reader for ResourcesMonitor
	// If nil, ResourcesMonitor won't be created
	MetricsWriter *metrics.VictoriaMetricsWriter
	MetricsReader *metrics.VictoriaMetricsReader
}

// Subsystem contains the built observability components.
type Subsystem struct {
	StatusMonitor    *observability.StatusMonitor
	LogWriter        *observability.PersistentLogWriter
	LogReader        *observability.LogReader
	LogsMaintainer   *observability.LogsMaintainer
	ResourcesMonitor *observability.ResourcesMonitor
}

// Option configures optional dependencies for the observability subsystem.
type Option func(*buildOpts)

type buildOpts struct {
	// Override components for testing
	statusMonitor *observability.StatusMonitor
	logWriter     *observability.PersistentLogWriter
	logReader     *observability.LogReader
}

// WithStatusMonitor overrides the default StatusMonitor.
func WithStatusMonitor(sm *observability.StatusMonitor) Option {
	return func(o *buildOpts) {
		o.statusMonitor = sm
	}
}

// WithLogWriter overrides the default PersistentLogWriter.
func WithLogWriter(lw *observability.PersistentLogWriter) Option {
	return func(o *buildOpts) {
		o.logWriter = lw
	}
}

// WithLogReader overrides the default LogReader.
func WithLogReader(lr *observability.LogReader) Option {
	return func(o *buildOpts) {
		o.logReader = lr
	}
}

// New creates a new observability subsystem with all components.
func New(cfg Config, opts ...Option) (*Subsystem, error) {
	// Validate required config
	v := subsystem.NewValidator("observability")
	v.Required("Log", cfg.Log)
	v.RequiredString("VictoriaLogsAddr", cfg.VictoriaLogsAddr)
	if err := v.Error(); err != nil {
		return nil, err
	}

	// Apply options
	bo := &buildOpts{}
	for _, opt := range opts {
		opt(bo)
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	sub := &Subsystem{}

	// Build or use provided StatusMonitor
	if bo.statusMonitor != nil {
		sub.StatusMonitor = bo.statusMonitor
	} else {
		sub.StatusMonitor = observability.NewStatusMonitor(cfg.Log)
	}

	// Build or use provided LogWriter
	if bo.logWriter != nil {
		sub.LogWriter = bo.logWriter
	} else {
		sub.LogWriter = observability.NewPersistentLogWriter(cfg.VictoriaLogsAddr, timeout)
	}

	// Build or use provided LogReader
	if bo.logReader != nil {
		sub.LogReader = bo.logReader
	} else {
		sub.LogReader = observability.NewLogReader(cfg.VictoriaLogsAddr, timeout)
	}

	// Build LogsMaintainer
	sub.LogsMaintainer = observability.NewLogsMaintainer()

	// Build ResourcesMonitor if metrics writer/reader provided
	if cfg.MetricsWriter != nil && cfg.MetricsReader != nil {
		sub.ResourcesMonitor = observability.NewResourcesMonitor(cfg.Log, cfg.MetricsWriter, cfg.MetricsReader)
	}

	return sub, nil
}

// Close cleans up observability subsystem resources.
func (s *Subsystem) Close() error {
	if s.ResourcesMonitor != nil && s.ResourcesMonitor.Writer != nil {
		return s.ResourcesMonitor.Writer.Close()
	}
	return nil
}
