package sandbox

import (
	"bytes"
	"log/slog"
	"os"
	"testing"

	"github.com/moby/buildkit/identity"
	"github.com/stretchr/testify/require"
	"miren.dev/runtime/observability"
)

type mockLogWriter struct {
	entries []mockLogEntry
}

type mockLogEntry struct {
	entity string
	log    observability.LogEntry
}

func (m *mockLogWriter) WriteEntry(entity string, le observability.LogEntry) error {
	m.entries = append(m.entries, mockLogEntry{
		entity: entity,
		log:    le,
	})
	return nil
}

func TestSandboxLogs(t *testing.T) {
	t.Run("processes stdout lines", func(t *testing.T) {
		r := require.New(t)

		mock := &mockLogWriter{}
		entityID := identity.NewID()

		logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
		sl := NewSandboxLogs(logger, entityID, map[string]string{}, mock)

		// Write some lines
		input := []byte("line 1\nline 2\nline 3\n")
		n, err := sl.Write(input)
		r.NoError(err)
		r.Equal(len(input), n)

		r.Len(mock.entries, 3)
		r.Equal("line 1", mock.entries[0].log.Body)
		r.Equal("line 2", mock.entries[1].log.Body)
		r.Equal("line 3", mock.entries[2].log.Body)

		for i, entry := range mock.entries {
			r.Equal(entityID, entry.entity, "entry %d should have correct entity", i)
			r.Equal(observability.Stdout, entry.log.Stream, "entry %d should be stdout", i)
		}
	})

	t.Run("buffers partial lines", func(t *testing.T) {
		r := require.New(t)

		mock := &mockLogWriter{}
		entityID := identity.NewID()

		logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
		sl := NewSandboxLogs(logger, entityID, map[string]string{}, mock)

		// Write partial line
		n, err := sl.Write([]byte("partial"))
		r.NoError(err)
		r.Equal(7, n)

		// Should not have written anything yet
		r.Len(mock.entries, 0)

		// Complete the line
		n, err = sl.Write([]byte(" line\n"))
		r.NoError(err)
		r.Equal(6, n)

		// Now should have one entry
		r.Len(mock.entries, 1)
		r.Equal("partial line", mock.entries[0].log.Body)
	})

	t.Run("handles USER prefix", func(t *testing.T) {
		r := require.New(t)

		mock := &mockLogWriter{}
		entityID := identity.NewID()

		logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
		sl := NewSandboxLogs(logger, entityID, map[string]string{}, mock)

		input := []byte("!USER this is a user log\n")
		n, err := sl.Write(input)
		r.NoError(err)
		r.Equal(len(input), n)

		r.Len(mock.entries, 1)
		r.Equal("this is a user log", mock.entries[0].log.Body)
		r.Equal(observability.UserOOB, mock.entries[0].log.Stream)
	})

	t.Run("handles ERROR prefix", func(t *testing.T) {
		r := require.New(t)

		mock := &mockLogWriter{}
		entityID := identity.NewID()

		logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
		sl := NewSandboxLogs(logger, entityID, map[string]string{}, mock)

		input := []byte("!ERROR this is an error log\n")
		n, err := sl.Write(input)
		r.NoError(err)
		r.Equal(len(input), n)

		r.Len(mock.entries, 1)
		r.Equal("this is an error log", mock.entries[0].log.Body)
		r.Equal(observability.Error, mock.entries[0].log.Stream)
	})

	t.Run("extracts trace ID from log", func(t *testing.T) {
		r := require.New(t)

		mock := &mockLogWriter{}
		entityID := identity.NewID()
		traceID := identity.NewID()

		logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
		sl := NewSandboxLogs(logger, entityID, map[string]string{}, mock)

		input := []byte("log with trace_id=" + traceID + "\n")
		n, err := sl.Write(input)
		r.NoError(err)
		r.Equal(len(input), n)

		r.Len(mock.entries, 1)
		r.Equal(traceID, mock.entries[0].log.TraceID)
	})

	t.Run("includes attributes in logs", func(t *testing.T) {
		r := require.New(t)

		mock := &mockLogWriter{}
		entityID := identity.NewID()

		attrs := map[string]string{
			"miren.sandbox":   "test-sandbox",
			"miren.container": "test-container",
			"miren.version":   "v1.0.0",
		}

		logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
		sl := NewSandboxLogs(logger, entityID, attrs, mock)

		input := []byte("log with attributes\n")
		n, err := sl.Write(input)
		r.NoError(err)
		r.Equal(len(input), n)

		r.Len(mock.entries, 1)
		r.Equal(attrs, mock.entries[0].log.Attributes)
	})

	t.Run("Stderr returns clone with stderr stream", func(t *testing.T) {
		r := require.New(t)

		mock := &mockLogWriter{}
		entityID := identity.NewID()

		logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
		sl := NewSandboxLogs(logger, entityID, map[string]string{}, mock)

		stderr := sl.Stderr()

		// Write to stderr clone
		input := []byte("error line\n")
		n, err := stderr.Write(input)
		r.NoError(err)
		r.Equal(len(input), n)

		r.Len(mock.entries, 1)
		r.Equal("error line", mock.entries[0].log.Body)
		r.Equal(observability.Stderr, mock.entries[0].log.Stream)

		// Original should still be stdout
		input2 := []byte("stdout line\n")
		n, err = sl.Write(input2)
		r.NoError(err)
		r.Equal(len(input2), n)

		r.Len(mock.entries, 2)
		r.Equal("stdout line", mock.entries[1].log.Body)
		r.Equal(observability.Stdout, mock.entries[1].log.Stream)
	})

	t.Run("handles multiple lines in single write", func(t *testing.T) {
		r := require.New(t)

		mock := &mockLogWriter{}
		entityID := identity.NewID()

		logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
		sl := NewSandboxLogs(logger, entityID, map[string]string{}, mock)

		input := []byte("line 1\nline 2\nline 3\n")
		n, err := sl.Write(input)
		r.NoError(err)
		r.Equal(len(input), n)

		r.Len(mock.entries, 3)
		r.Equal("line 1", mock.entries[0].log.Body)
		r.Equal("line 2", mock.entries[1].log.Body)
		r.Equal("line 3", mock.entries[2].log.Body)
	})

	t.Run("trims trailing newlines and tabs", func(t *testing.T) {
		r := require.New(t)

		mock := &mockLogWriter{}
		entityID := identity.NewID()

		logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
		sl := NewSandboxLogs(logger, entityID, map[string]string{}, mock)

		input := []byte("line with trailing\t\r\n")
		n, err := sl.Write(input)
		r.NoError(err)
		r.Equal(len(input), n)

		r.Len(mock.entries, 1)
		r.Equal("line with trailing", mock.entries[0].log.Body)
	})
}

func BenchmarkSandboxLogs(b *testing.B) {
	mock := &mockLogWriter{}
	entityID := identity.NewID()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	sl := NewSandboxLogs(logger, entityID, map[string]string{}, mock)

	input := []byte("benchmark log line\n")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sl.Write(input)
	}
}

func TestParseStructuredJSON(t *testing.T) {
	t.Run("valid JSON with msg", func(t *testing.T) {
		body, extra, stream, ok := parseStructuredJSON(`{"time":"2026-01-01T00:00:00Z","level":"INFO","msg":"hello","key":"val"}`)
		if !ok {
			t.Fatal("expected ok=true")
		}
		if body != "hello" {
			t.Errorf("body = %q, want %q", body, "hello")
		}
		if extra["key"] != "val" {
			t.Errorf("extra[key] = %q, want %q", extra["key"], "val")
		}
		for _, skip := range []string{"time", "level", "msg"} {
			if _, exists := extra[skip]; exists {
				t.Errorf("field %q should be stripped", skip)
			}
		}
		if stream != "" {
			t.Errorf("stream = %q, want empty for INFO", stream)
		}
	})

	t.Run("valid JSON with message field", func(t *testing.T) {
		body, _, _, ok := parseStructuredJSON(`{"time":"2026-01-01T00:00:00Z","level":"INFO","message":"hello"}`)
		if !ok {
			t.Fatal("expected ok=true")
		}
		if body != "hello" {
			t.Errorf("body = %q, want %q", body, "hello")
		}
	})

	t.Run("ERROR level sets stderr stream", func(t *testing.T) {
		_, _, stream, ok := parseStructuredJSON(`{"level":"ERROR","msg":"fail"}`)
		if !ok {
			t.Fatal("expected ok=true")
		}
		if stream != observability.Stderr {
			t.Errorf("stream = %q, want stderr", stream)
		}
	})

	t.Run("WARN level sets stderr stream", func(t *testing.T) {
		_, _, stream, ok := parseStructuredJSON(`{"level":"WARN","msg":"warning"}`)
		if !ok {
			t.Fatal("expected ok=true")
		}
		if stream != observability.Stderr {
			t.Errorf("stream = %q, want stderr", stream)
		}
	})

	t.Run("non-JSON returns false", func(t *testing.T) {
		_, _, _, ok := parseStructuredJSON("plain text")
		if ok {
			t.Error("expected ok=false for plain text")
		}
	})

	t.Run("JSON without msg field returns false", func(t *testing.T) {
		_, _, _, ok := parseStructuredJSON(`{"key":"value","other":"data"}`)
		if ok {
			t.Error("expected ok=false for JSON without msg")
		}
	})

	t.Run("non-string values are formatted", func(t *testing.T) {
		_, extra, _, ok := parseStructuredJSON(`{"msg":"hi","count":42,"flag":true}`)
		if !ok {
			t.Fatal("expected ok=true")
		}
		if extra["count"] != "42" {
			t.Errorf("extra[count] = %q, want %q", extra["count"], "42")
		}
		if extra["flag"] != "true" {
			t.Errorf("extra[flag] = %q, want %q", extra["flag"], "true")
		}
	})
}

func TestProcessLineJSON(t *testing.T) {
	mock := &mockLogWriter{}
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	entityID := "test-entity"
	baseAttrs := map[string]string{"miren.sandbox": "sandbox/test-abc"}

	sl := NewSandboxLogs(logger, entityID, baseAttrs, mock)
	sl.Write([]byte(`{"time":"2026-01-01T00:00:00Z","level":"INFO","msg":"processing step","component":"provisioner","cluster_id":"ZA8"}` + "\n"))

	if len(mock.entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(mock.entries))
	}

	entry := mock.entries[0].log
	if entry.Body != "processing step" {
		t.Errorf("body = %q, want %q", entry.Body, "processing step")
	}
	if entry.Attributes["component"] != "provisioner" {
		t.Errorf("attrs[component] = %q, want %q", entry.Attributes["component"], "provisioner")
	}
	if entry.Attributes["cluster_id"] != "ZA8" {
		t.Errorf("attrs[cluster_id] = %q, want %q", entry.Attributes["cluster_id"], "ZA8")
	}
	if entry.Attributes["miren.sandbox"] != "sandbox/test-abc" {
		t.Errorf("base attrs should be preserved: attrs[miren.sandbox] = %q", entry.Attributes["miren.sandbox"])
	}
	if _, hasTime := entry.Attributes["time"]; hasTime {
		t.Error("time field should be stripped from attributes")
	}
}

func BenchmarkSandboxLogsLargeBuffer(b *testing.B) {
	mock := &mockLogWriter{}
	entityID := identity.NewID()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	sl := NewSandboxLogs(logger, entityID, map[string]string{}, mock)

	// Create a large buffer with many lines
	var buf bytes.Buffer
	for i := 0; i < 100; i++ {
		buf.WriteString("log line ")
		buf.WriteString(string('0' + rune(i%10)))
		buf.WriteByte('\n')
	}
	input := buf.Bytes()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sl.Write(input)
		mock.entries = mock.entries[:0] // Reset
	}
}
