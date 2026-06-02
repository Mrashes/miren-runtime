package sandbox

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"miren.dev/runtime/observability"
)

var traceIdRegx = regexp.MustCompile(`trace_id"?\s*[=:]\s*\"?(\w+)`)

type SandboxLogs struct {
	log    *slog.Logger
	entity string
	attrs  map[string]string
	buf    bytes.Buffer
	stream observability.LogStream
	lw     observability.LogWriter
}

func NewSandboxLogs(
	log *slog.Logger,
	entity string,
	attrs map[string]string,
	lw observability.LogWriter,
) *SandboxLogs {
	return &SandboxLogs{
		log:    log,
		entity: entity,
		attrs:  attrs,
		stream: observability.Stdout,
		lw:     lw,
	}
}

func (s *SandboxLogs) Write(p []byte) (n int, err error) {
	n = len(p)

	if s.buf.Len() > 0 {
		s.buf.Write(p)
		p = s.buf.Bytes()
	}

	for len(p) > 0 {
		nl := bytes.IndexByte(p, '\n')
		if nl == -1 {
			s.buf.Write(p)
			break
		}

		s.processLine(string(p[:nl]))

		p = p[nl+1:]
	}

	return
}

var jsonLogSkipFields = map[string]bool{
	"time":    true,
	"level":   true,
	"msg":     true,
	"message": true,
}

var jsonLevelToStream = map[string]observability.LogStream{
	"ERROR": observability.Stderr,
	"error": observability.Stderr,
	"WARN":  observability.Stderr,
	"warn":  observability.Stderr,
}

func (s *SandboxLogs) processLine(line string) {
	ts := time.Now()

	line = strings.TrimRight(line, "\t\n\r")

	stream := s.stream

	if strings.HasPrefix(line, "!USER ") {
		line = strings.TrimPrefix(line, "!USER ")
		stream = observability.UserOOB
	} else if strings.HasPrefix(line, "!ERROR ") {
		line = strings.TrimPrefix(line, "!ERROR ")
		stream = observability.Error
	}

	traceId := ""
	if matches := traceIdRegx.FindStringSubmatch(line); len(matches) > 1 {
		traceId = matches[1]
	}

	attrs := s.attrs
	if body, extra, lvlStream, ok := parseStructuredJSON(line); ok {
		extra["user.orig_msg"] = line
		line = body
		if lvlStream != "" {
			stream = lvlStream
		}
		attrs = make(map[string]string)
		for k, v := range s.attrs {
			attrs[k] = v
		}
		for k, v := range extra {
			attrs[k] = v
		}
	}

	err := s.lw.WriteEntry(s.entity, observability.LogEntry{
		Timestamp:  ts,
		Stream:     stream,
		Body:       line,
		TraceID:    traceId,
		Attributes: attrs,
	})
	if err != nil {
		s.log.Error("failed to write log entry", "error", err, "line", line)
	}
}

// parseStructuredJSON detects structured JSON log lines and extracts fields.
// Returns the message body, extra attributes, an optional stream override, and whether parsing succeeded.
func parseStructuredJSON(line string) (string, map[string]string, observability.LogStream, bool) {
	if len(line) == 0 || line[0] != '{' {
		return "", nil, "", false
	}

	var fields map[string]any
	if err := json.Unmarshal([]byte(line), &fields); err != nil {
		return "", nil, "", false
	}

	msg, _ := fields["msg"].(string)
	if msg == "" {
		msg, _ = fields["message"].(string)
	}
	if msg == "" {
		return "", nil, "", false
	}

	var stream observability.LogStream
	if level, ok := fields["level"].(string); ok {
		stream = jsonLevelToStream[level]
	}

	extra := make(map[string]string, len(fields))
	for k, v := range fields {
		if jsonLogSkipFields[k] {
			continue
		}
		switch val := v.(type) {
		case string:
			extra[k] = val
		default:
			extra[k] = fmt.Sprintf("%v", v)
		}
	}

	return msg, extra, stream, true
}

func (s *SandboxLogs) Stderr() *SandboxLogs {
	x := *s
	x.stream = observability.Stderr

	return &x
}
