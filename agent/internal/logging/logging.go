// CherryPi: A multi-agent-multi-human self-hosted agency
// Copyright (C) 2026 CherryPi authors <cherrypi.dev>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

// Package logging builds the agent's structured logger. Records go to stdout
// (text or JSON) and, unless they are marked local, are mirrored into the
// agent's event hub so the agency can persist and show them.
package logging

import (
	"context"
	"io"
	"log"
	"log/slog"
	"os"
	"strings"
	"time"
)

// Sink receives one payload per mirrored record. The agent's event hub
// satisfies it (Publish(kind, payload)).
type Sink func(kind string, payload map[string]any)

// Config configures the process logger.
type Config struct {
	// Format is "text" (default) or "json".
	Format string
	// Level is the minimum level: debug, info, warn, error. Default info.
	Level string
	// Output is where the rendered lines go. Default os.Stdout.
	Output io.Writer
	// Sink, when set, mirrors every non-local record as a `log` event.
	Sink Sink
}

// localKey tags a record that must not be mirrored. See Local.
const localKey = "_local"

// Setup installs the structured logger as the process default and routes the
// standard library `log` package through it, so every line reaches stdout and
// the sink. It returns the process logger.
func Setup(cfg Config) *slog.Logger {
	out := cfg.Output
	if out == nil {
		out = os.Stdout
	}
	opts := &slog.HandlerOptions{Level: ParseLevel(cfg.Level)}

	var base slog.Handler
	if strings.EqualFold(strings.TrimSpace(cfg.Format), "json") {
		base = slog.NewJSONHandler(out, opts)
	} else {
		base = slog.NewTextHandler(out, opts)
	}

	handler := base
	if cfg.Sink != nil {
		handler = &sinkHandler{next: base, sink: cfg.Sink}
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)

	// Existing log.Printf/Fatalf call sites keep working: send them through
	// the same handler instead of the raw stderr logger.
	log.SetOutput(&logWriter{logger: logger})
	log.SetFlags(0)

	return logger
}

// Local returns the process logger tagged local-only: its records reach stdout
// but skip the hub mirror. Connectivity is recorded centrally by the agency,
// so mirroring it from the agent would duplicate every line.
func Local() *slog.Logger {
	return slog.Default().With(slog.Bool(localKey, true))
}

// ParseLevel maps a level name to a slog level. Unknown values are info.
func ParseLevel(name string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// LevelName is the lower-case wire name for a level, used both in the payload
// and in the stdout text handler.
func LevelName(level slog.Level) string {
	switch {
	case level >= slog.LevelError:
		return "error"
	case level >= slog.LevelWarn:
		return "warn"
	case level >= slog.LevelInfo:
		return "info"
	default:
		return "debug"
	}
}

// sinkHandler mirrors records into the sink, then delegates to the stdout
// handler. Local records bypass the sink.
type sinkHandler struct {
	next slog.Handler
	sink Sink
	pre  []slog.Attr
}

func (h *sinkHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *sinkHandler) Handle(ctx context.Context, r slog.Record) error {
	if !h.isLocal(r) {
		payload := map[string]any{
			"level":   LevelName(r.Level),
			"message": r.Message,
		}
		for _, a := range h.pre {
			setAttr(payload, a)
		}
		r.Attrs(func(a slog.Attr) bool {
			setAttr(payload, a)
			return true
		})
		h.sink("log", payload)
	}
	return h.next.Handle(ctx, r)
}

func (h *sinkHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	pre := make([]slog.Attr, 0, len(h.pre)+len(attrs))
	pre = append(pre, h.pre...)
	pre = append(pre, attrs...)

	// The local marker drives the sink decision; keep it out of stdout.
	next := make([]slog.Attr, 0, len(attrs))
	for _, a := range attrs {
		if a.Key != localKey {
			next = append(next, a)
		}
	}
	return &sinkHandler{next: h.next.WithAttrs(next), sink: h.sink, pre: pre}
}

func (h *sinkHandler) WithGroup(name string) slog.Handler {
	return &sinkHandler{next: h.next.WithGroup(name), sink: h.sink, pre: h.pre}
}

func (h *sinkHandler) isLocal(r slog.Record) bool {
	for _, a := range h.pre {
		if a.Key == localKey && a.Value.Bool() {
			return true
		}
	}
	local := false
	r.Attrs(func(a slog.Attr) bool {
		if a.Key == localKey && a.Value.Bool() {
			local = true
		}
		return true
	})
	return local
}

// setAttr copies a resolved attribute into the payload, dropping the local
// marker and flattening errors/times into JSON-friendly values.
func setAttr(payload map[string]any, a slog.Attr) {
	if a.Key == localKey {
		return
	}
	payload[a.Key] = value(a.Value.Resolve())
}

func value(v slog.Value) any {
	switch v.Kind() {
	case slog.KindString:
		return v.String()
	case slog.KindInt64:
		return v.Int64()
	case slog.KindUint64:
		return v.Uint64()
	case slog.KindFloat64:
		return v.Float64()
	case slog.KindBool:
		return v.Bool()
	case slog.KindDuration:
		return v.Duration().String()
	case slog.KindTime:
		return v.Time().UTC().Format(time.RFC3339Nano)
	case slog.KindAny:
		if err, ok := v.Any().(error); ok {
			return err.Error()
		}
		return v.Any()
	default:
		return v.String()
	}
}

// logWriter adapts the standard library logger to slog so legacy log.Printf
// lines reach stdout and the sink.
type logWriter struct{ logger *slog.Logger }

func (w *logWriter) Write(p []byte) (int, error) {
	if msg := strings.TrimRight(string(p), "\n"); msg != "" {
		w.logger.Info(msg)
	}
	return len(p), nil
}
