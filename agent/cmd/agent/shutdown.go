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

package main

import (
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"
)

const shutdownTeardownTimeout = 5 * time.Second

const shutdownNoteTimeout = 500 * time.Millisecond

// the very line that explains it. stop disarms the guard once teardown

func guardShutdown(
	sig <-chan os.Signal,
	timeout time.Duration,
	note func(string),
	dump func(),
	exit func(int),
) (stop func()) {
	done := make(chan struct{})
	var once sync.Once
	stop = func() { once.Do(func() { close(done) }) }

	go func() {
		timer := time.NewTimer(timeout)
		defer timer.Stop()

		select {
		case <-done:
		case <-sig:
			note("Forced shutdown: exiting without waiting for teardown")
			exit(0)
		case <-timer.C:
			dump()
			note("Shutdown teardown did not finish in time: exiting")
			exit(0)
		}
	}()

	return stop
}

// finishShutdown writes the final process-log line and only then disarms the

func finishShutdown(stop func(), write func(string)) {
	write("Shutdown complete")
	stop()
}

func shutdownNote(message string) {
	writeStderr(message)
}

func dumpGoroutines() {
	buf := make([]byte, 1<<20)
	n := runtime.Stack(buf, true)

	report := "Shutdown teardown goroutine dump:\n" + string(buf[:n])
	if n == len(buf) {
		report += "(truncated)\n"
	}

	writeStderr(report)
}

func writeStderr(message string) {
	written := make(chan struct{})
	go func() {
		_, _ = fmt.Fprintln(os.Stderr, message)
		close(written)
	}()

	select {
	case <-written:
	case <-time.After(shutdownNoteTimeout):
	}
}
