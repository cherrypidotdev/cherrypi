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

type Sink func(kind string, payload map[string]any)

type Config struct {
	Format string

	Level string

	Output io.Writer

	Sink Sink
}

const localKey = "_local"

// standard library `log` package through it, so every line reaches stdout and

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

	log.SetOutput(&logWriter{logger: logger})
	log.SetFlags(0)

	return logger
}

func Local() *slog.Logger {
	return slog.Default().With(slog.Bool(localKey, true))
}

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

type logWriter struct{ logger *slog.Logger }

func (w *logWriter) Write(p []byte) (int, error) {
	if msg := strings.TrimRight(string(p), "\n"); msg != "" {
		w.logger.Info(msg)
	}
	return len(p), nil
}
