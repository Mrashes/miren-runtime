// Package metrics provides a subsystem for building metrics collection components.
package metrics

import (
	"fmt"
	"log/slog"
	"time"

	"miren.dev/runtime/metrics"
	"miren.dev/runtime/pkg/subsystem"
)

// Config holds required configuration for the metrics subsystem.
type Config struct {
	Log     *slog.Logger
	Address string        // VictoriaMetrics address (e.g., "localhost:8428")
	Timeout time.Duration // Request timeout (0 uses default of 30s)
}

// Subsystem contains the built metrics components.
type Subsystem struct {
	Writer      *metrics.VictoriaMetricsWriter
	Reader      *metrics.VictoriaMetricsReader
	CPUUsage    *metrics.CPUUsage
	MemoryUsage *metrics.MemoryUsage
	HTTPMetrics *metrics.HTTPMetrics
}

// Option configures optional dependencies for the metrics subsystem.
type Option func(*buildOpts)

type buildOpts struct {
	// Override components for testing
	writer *metrics.VictoriaMetricsWriter
	reader *metrics.VictoriaMetricsReader

	// Skip building specific components
	skipCPU    bool
	skipMemory bool
	skipHTTP   bool
}

// WithWriter overrides the default VictoriaMetricsWriter.
func WithWriter(w *metrics.VictoriaMetricsWriter) Option {
	return func(o *buildOpts) {
		o.writer = w
	}
}

// WithReader overrides the default VictoriaMetricsReader.
func WithReader(r *metrics.VictoriaMetricsReader) Option {
	return func(o *buildOpts) {
		o.reader = r
	}
}

// WithoutCPUMetrics disables building CPUUsage.
func WithoutCPUMetrics() Option {
	return func(o *buildOpts) {
		o.skipCPU = true
	}
}

// WithoutMemoryMetrics disables building MemoryUsage.
func WithoutMemoryMetrics() Option {
	return func(o *buildOpts) {
		o.skipMemory = true
	}
}

// WithoutHTTPMetrics disables building HTTPMetrics.
func WithoutHTTPMetrics() Option {
	return func(o *buildOpts) {
		o.skipHTTP = true
	}
}

// New creates a new metrics subsystem with all components.
func New(cfg Config, opts ...Option) (*Subsystem, error) {
	// Validate required config
	v := subsystem.NewValidator("metrics")
	v.Required("Log", cfg.Log)
	v.RequiredString("Address", cfg.Address)
	if err := v.Error(); err != nil {
		return nil, err
	}

	// Apply options
	bo := &buildOpts{}
	for _, opt := range opts {
		opt(bo)
	}

	sub := &Subsystem{}

	// Build or use provided writer
	if bo.writer != nil {
		sub.Writer = bo.writer
	} else {
		sub.Writer = metrics.NewVictoriaMetricsWriter(cfg.Log, cfg.Address, cfg.Timeout)
		sub.Writer.Start()
	}

	// Build or use provided reader
	if bo.reader != nil {
		sub.Reader = bo.reader
	} else {
		sub.Reader = metrics.NewVictoriaMetricsReader(cfg.Log, cfg.Address, cfg.Timeout)
	}

	// Build CPU usage metrics
	if !bo.skipCPU {
		sub.CPUUsage = metrics.NewCPUUsage(cfg.Log, sub.Writer, sub.Reader)
		if err := sub.CPUUsage.Setup(); err != nil {
			sub.Close()
			return nil, fmt.Errorf("metrics: failed to setup CPU usage: %w", err)
		}
	}

	// Build memory usage metrics
	if !bo.skipMemory {
		sub.MemoryUsage = metrics.NewMemoryUsage(cfg.Log, sub.Writer, sub.Reader)
		if err := sub.MemoryUsage.Setup(); err != nil {
			sub.Close()
			return nil, fmt.Errorf("metrics: failed to setup memory usage: %w", err)
		}
	}

	// Build HTTP metrics
	if !bo.skipHTTP {
		sub.HTTPMetrics = metrics.NewHTTPMetrics(cfg.Log, sub.Writer, sub.Reader)
		if err := sub.HTTPMetrics.Setup(); err != nil {
			sub.Close()
			return nil, fmt.Errorf("metrics: failed to setup HTTP metrics: %w", err)
		}
	}

	return sub, nil
}

// Close stops the metrics writer and cleans up resources.
func (s *Subsystem) Close() error {
	if s.Writer != nil {
		return s.Writer.Close()
	}
	return nil
}
