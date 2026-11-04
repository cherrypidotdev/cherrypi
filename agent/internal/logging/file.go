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
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

const (
	DefaultMaxBytes int64 = 8 << 20

	DefaultKeep = 3
)

type fileSink struct {
	mu       sync.Mutex
	path     string
	maxBytes int64
	keep     int

	file *os.File
	size int64
}

func openFileSink(path string, maxBytes int64, keep int) (*fileSink, error) {
	if path == "" {
		return nil, nil
	}
	if maxBytes <= 0 {
		maxBytes = DefaultMaxBytes
	}
	if keep < 0 {
		keep = 0
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create log directory: %w", err)
	}
	s := &fileSink{path: path, maxBytes: maxBytes, keep: keep}
	if err := s.open(); err != nil {
		return nil, err
	}
	return s, nil
}

// size limit, so a single line is never split across files.
func (s *fileSink) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.file == nil {
		if err := s.open(); err != nil {
			return 0, err
		}
	}
	if s.size > 0 && s.size+int64(len(p)) > s.maxBytes {

		if err := s.rotate(); err != nil {
			return 0, err
		}
	}
	n, err := s.file.Write(p)
	s.size += int64(n)
	return n, err
}

func (s *fileSink) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.file == nil {
		return nil
	}
	err := s.file.Close()
	s.file = nil
	return err
}

func (s *fileSink) Path() string {
	return s.path
}

func (s *fileSink) open() error {
	file, err := os.OpenFile(s.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return fmt.Errorf("stat log file: %w", err)
	}
	s.file = file
	s.size = info.Size()
	return nil
}

func (s *fileSink) rotate() error {
	if s.keep > 0 {
		_ = os.Remove(fmt.Sprintf("%s.%d", s.path, s.keep))
		for i := s.keep - 1; i >= 1; i-- {
			_ = os.Rename(
				fmt.Sprintf("%s.%d", s.path, i),
				fmt.Sprintf("%s.%d", s.path, i+1),
			)
		}
		if err := os.Rename(s.path, s.path+".1"); err != nil {
			return fmt.Errorf("rotate log file: %w", err)
		}
	}
	if err := s.file.Close(); err != nil {
		return fmt.Errorf("close rotated log file: %w", err)
	}
	s.file = nil
	s.size = 0
	if s.keep == 0 {
		if err := os.Remove(s.path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove log file: %w", err)
		}
	}
	return s.open()
}
