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

package processlog

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/sys/unix"
)

const DefaultKeep = 7

const StampLayout = "2006-01-02T15:04:05.000Z07:00"

type Config struct {
	Dir string

	Keep int

	Version string

	Now func() time.Time
}

type Log struct {
	dir       string
	path      string
	name      string
	version   string
	startedAt time.Time
	pid       int
	keep      int
	now       func() time.Time

	mu   sync.Mutex
	file *os.File

	also io.Writer

	pipe *os.File
}

type Meta struct {
	Version   string `json:"version"`
	File      string `json:"file"`
	Dir       string `json:"dir"`
	StartedAt string `json:"started_at"`
	PID       int    `json:"pid"`
	UptimeMs  int64  `json:"uptime_ms"`
	Keep      int    `json:"keep"`
}

func New(cfg Config) (*Log, error) {
	if cfg.Dir == "" {
		return nil, errors.New("process log directory is required")
	}
	keep := cfg.Keep
	if keep <= 0 {
		keep = DefaultKeep
	}
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	if err := os.MkdirAll(cfg.Dir, 0o755); err != nil {
		return nil, fmt.Errorf("create log directory: %w", err)
	}

	started := now().UTC()
	name := fmt.Sprintf("agency-%s-%d.log", started.Format("20060102T150405Z"), os.Getpid())
	path := filepath.Join(cfg.Dir, name)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open log file: %w", err)
	}

	l := &Log{
		dir:       cfg.Dir,
		path:      path,
		name:      name,
		version:   cfg.Version,
		startedAt: started,
		pid:       os.Getpid(),
		keep:      keep,
		now:       now,
		file:      file,
	}
	l.prune()
	return l, nil
}

// into it. Every line is written to the file prefixed with a UTC timestamp and

func Setup(cfg Config) (*Log, error) {
	l, err := New(cfg)
	if err != nil {
		return nil, err
	}
	if err := l.capture(); err != nil {

		return l, fmt.Errorf("capture process output: %w", err)
	}
	return l, nil
}

func (l *Log) capture() error {
	read, write, err := os.Pipe()
	if err != nil {
		return err
	}

	saved, err := unix.Dup(int(os.Stdout.Fd()))
	if err != nil {
		_ = read.Close()
		_ = write.Close()
		return err
	}

	if err := unix.Dup2(int(write.Fd()), int(os.Stdout.Fd())); err != nil {
		_ = unix.Close(saved)
		_ = read.Close()
		_ = write.Close()
		return err
	}
	if err := unix.Dup2(int(write.Fd()), int(os.Stderr.Fd())); err != nil {
		_ = unix.Close(saved)
		_ = read.Close()
		_ = write.Close()
		return err
	}

	_ = write.Close()

	l.mu.Lock()
	l.also = os.NewFile(uintptr(saved), "stdout")
	l.pipe = read
	l.mu.Unlock()

	go l.pump(read)
	return nil
}

// pump copies the captured output into the file, one line at a time.
func (l *Log) pump(reader io.Reader) {
	br := bufio.NewReaderSize(reader, 64*1024)
	for {
		line, err := br.ReadString('\n')
		if line != "" {
			l.append(strings.TrimSuffix(line, "\n"))
		}
		if err != nil {
			return
		}
	}
}

// append writes one stamped line to the file and the raw line to the original

func (l *Log) append(line string) {
	l.mu.Lock()
	if l.file != nil {
		_, _ = l.file.WriteString(l.now().UTC().Format(StampLayout) + " " + line + "\n")
	}
	l.mu.Unlock()

	if l.also != nil {
		_, _ = io.WriteString(l.also, line+"\n")
	}
}

func (l *Log) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file == nil {
		return nil
	}
	err := l.file.Close()
	l.file = nil
	return err
}

func (l *Log) Path() string { return l.path }

func (l *Log) Name() string { return l.name }

func (l *Log) Dir() string { return l.dir }

func (l *Log) Keep() int { return l.keep }

func (l *Log) Meta() Meta {
	return Meta{
		Version:   l.version,
		File:      l.name,
		Dir:       l.dir,
		StartedAt: l.startedAt.Format(time.RFC3339),
		PID:       l.pid,
		UptimeMs:  l.now().Sub(l.startedAt).Milliseconds(),
		Keep:      l.keep,
	}
}

func (l *Log) Tail(n int) ([]string, int64, error) {
	if n <= 0 {
		n = 500
	}
	lines, size, err := tailFile(l.path, n)
	if err != nil {
		return nil, 0, err
	}
	return lines, size, nil
}

// current end"), reporting after each line the byte offset that resumes past

func (l *Log) Follow(
	ctx context.Context,
	offset int64,
	emit func(offset int64, line string) bool,
	ping func() bool,
) error {
	file, err := os.Open(l.path)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	info, err := file.Stat()
	if err != nil {
		return err
	}
	if offset <= 0 || offset > info.Size() {
		offset = info.Size()
	}
	if _, err := file.Seek(offset, io.SeekStart); err != nil {
		return err
	}

	var pending []byte
	pendingStart := offset
	buf := make([]byte, 32*1024)
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	lastPing := time.Now()

	for {
		for {
			n, err := file.Read(buf)
			if n > 0 {
				pending = append(pending, buf[:n]...)
				for {
					idx := bytes.IndexByte(pending, '\n')
					if idx < 0 {
						break
					}
					line := string(pending[:idx])
					next := pendingStart + int64(idx) + 1
					if !emit(next, line) {
						return nil
					}
					pendingStart = next
					pending = pending[idx+1:]
				}
			}
			if err != nil {
				break
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if ping != nil && time.Since(lastPing) >= 10*time.Second {
				if !ping() {
					return nil
				}
				lastPing = time.Now()
			}
		}
	}
}

func tailFile(path string, n int) ([]string, int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = file.Close() }()

	info, err := file.Stat()
	if err != nil {
		return nil, 0, err
	}
	size := info.Size()

	const chunk = 64 * 1024
	var data []byte
	for readFrom := size; readFrom > 0; {
		start := readFrom - chunk
		if start < 0 {
			start = 0
		}
		buf := make([]byte, readFrom-start)
		if _, err := file.ReadAt(buf, start); err != nil && !errors.Is(err, io.EOF) {
			return nil, 0, err
		}
		data = append(buf, data...)
		readFrom = start
		if bytes.Count(data, []byte("\n")) > n {
			break
		}
	}

	lines := strings.Split(string(data), "\n")

	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return lines, size, nil
}

func (l *Log) prune() {
	entries, err := os.ReadDir(l.dir)
	if err != nil {
		return
	}
	type run struct {
		path string
		at   time.Time
	}
	var runs []run
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, "agency-") || !strings.HasSuffix(name, ".log") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		runs = append(runs, run{path: filepath.Join(l.dir, name), at: info.ModTime()})
	}
	if len(runs) <= l.keep {
		return
	}
	sort.Slice(runs, func(i, j int) bool { return runs[i].at.After(runs[j].at) })
	for _, old := range runs[l.keep:] {
		if old.path != l.path {
			_ = os.Remove(old.path)
		}
	}
}

func ParseKeep(value string) int {
	keep, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || keep <= 0 {
		return 0
	}
	return keep
}
