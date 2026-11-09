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

package telemetry

import (
	"io"
	"log"
	"log/slog"
	"regexp"
	"strings"
	"sync"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	sdklog "go.opentelemetry.io/otel/sdk/log"
)

// while keeping every line on the process log (its previous writer, which the

// those calls. This adapter turns each written line into one info record. The

// the line again.
func bridgeStdLog(provider *sdklog.LoggerProvider) func() {
	handler := otelslog.NewHandler(
		"cherrypi.vladfaust.com/agency",
		otelslog.WithLoggerProvider(provider),
	)
	logger := slog.New(handler)

	previousWriter := log.Writer()
	previousPrefix := log.Prefix()

	bridge := &logBridge{
		logger:   logger,
		fallback: previousWriter,
		prefix:   previousPrefix,
	}
	log.SetOutput(bridge)

	return func() {
		log.SetOutput(previousWriter)
		log.SetPrefix(previousPrefix)
	}
}

var stdLogStamp = regexp.MustCompile(`^\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}(\.\d+)? `)

type logBridge struct {
	logger   *slog.Logger
	fallback io.Writer
	prefix   string

	mu      sync.Mutex
	partial string
}

func (b *logBridge) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	n, err := b.fallback.Write(p)
	if err != nil {
		return n, err
	}

	text := b.partial + string(p)
	lines := strings.Split(text, "\n")
	b.partial = lines[len(lines)-1]
	for _, line := range lines[:len(lines)-1] {
		if line == "" {
			continue
		}
		b.logger.Info(b.message(line))
	}

	return n, nil
}

// line. A line with different flags keeps its stamp rather than losing text.
func (b *logBridge) message(line string) string {
	line = strings.TrimPrefix(line, b.prefix)

	return stdLogStamp.ReplaceAllString(line, "")
}
