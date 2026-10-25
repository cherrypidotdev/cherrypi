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

package desktop

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

var ErrUnavailable = errors.New("desktop is not available")

type Config struct {
	Display string

	XAUTHORITY string

	X11VNCBin string

	XdpyinfoBin string

	VNCPort uint16

	StartupTimeout time.Duration

	probe func(ctx context.Context, display string) (uint16, uint16, error)

	ready func(ctx context.Context, port int) error
}

type Info struct {
	Display  string
	Width    uint16
	Height   uint16
	Attached bool
}

type bridge struct {
	cmd  *exec.Cmd
	port int
}

type Manager struct {
	cfg Config

	mu      sync.Mutex
	bridges map[bool]*bridge
}

func NewManager(cfg Config) *Manager {
	if cfg.Display == "" {
		cfg.Display = os.Getenv("DISPLAY")
	}
	if cfg.XAUTHORITY == "" {
		cfg.XAUTHORITY = os.Getenv("XAUTHORITY")
	}
	if cfg.X11VNCBin == "" {
		cfg.X11VNCBin = "x11vnc"
	}
	if cfg.XdpyinfoBin == "" {
		cfg.XdpyinfoBin = "xdpyinfo"
	}
	if cfg.StartupTimeout == 0 {
		cfg.StartupTimeout = 10 * time.Second
	}
	if cfg.probe == nil {
		cfg.probe = probeDisplay
	}
	if cfg.ready == nil {
		cfg.ready = func(ctx context.Context, port int) error {
			return waitPort(ctx, port, cfg.StartupTimeout)
		}
	}
	return &Manager{cfg: cfg, bridges: map[bool]*bridge{}}
}

func (m *Manager) Display() string { return m.cfg.Display }

func (m *Manager) Availability(ctx context.Context) (bool, string) {
	if m.cfg.Display == "" {
		return false, "no X display configured (set DISPLAY)"
	}
	if _, err := exec.LookPath(m.cfg.X11VNCBin); err != nil {
		return false, "x11vnc is not installed (use the desktop image)"
	}
	if _, _, err := m.cfg.probe(ctx, m.cfg.Display); err != nil {
		return false, fmt.Sprintf("cannot reach X display %s: %v", m.cfg.Display, err)
	}
	return true, ""
}

func (m *Manager) Info(ctx context.Context) (Info, bool) {
	info := Info{Display: m.cfg.Display}
	width, height, err := m.cfg.probe(ctx, m.cfg.Display)
	if err != nil {
		return info, false
	}
	info.Width = width
	info.Height = height

	m.mu.Lock()
	for _, b := range m.bridges {
		if b != nil {
			info.Attached = true
			break
		}
	}
	m.mu.Unlock()
	return info, true
}

func (m *Manager) Ensure(ctx context.Context, viewOnly bool) (int, error) {
	if ok, reason := m.Availability(ctx); !ok {
		return 0, fmt.Errorf("%w: %s", ErrUnavailable, reason)
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if m.bridges == nil {
		m.bridges = map[bool]*bridge{}
	}
	if existing := m.bridges[viewOnly]; existing != nil {
		return existing.port, nil
	}

	port := int(m.cfg.VNCPort)
	if port != 0 {
		for _, b := range m.bridges {
			if b != nil && b.port == port {
				port = 0
				break
			}
		}
	}
	if port == 0 {
		var err error
		if port, err = freePort(); err != nil {
			return 0, err
		}
	}

	cmd := exec.Command(m.cfg.X11VNCBin, m.args(port, viewOnly)...)
	cmd.Env = m.env()

	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("start x11vnc: %w", err)
	}

	m.bridges[viewOnly] = &bridge{cmd: cmd, port: port}
	go m.reap(cmd, viewOnly)

	if err := m.cfg.ready(ctx, port); err != nil {
		m.stopLocked(viewOnly)
		return 0, fmt.Errorf("x11vnc did not start: %w", err)
	}
	return port, nil
}

func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for viewOnly := range m.bridges {
		m.stopLocked(viewOnly)
	}
}

func (m *Manager) stopLocked(viewOnly bool) {
	b := m.bridges[viewOnly]
	delete(m.bridges, viewOnly)
	if b == nil || b.cmd.Process == nil {
		return
	}
	_ = syscall.Kill(-b.cmd.Process.Pid, syscall.SIGTERM)
}

func (m *Manager) reap(cmd *exec.Cmd, viewOnly bool) {
	err := cmd.Wait()

	m.mu.Lock()
	b := m.bridges[viewOnly]
	unexpected := b != nil && b.cmd == cmd
	if unexpected {
		delete(m.bridges, viewOnly)
	}
	m.mu.Unlock()

	if unexpected && err != nil {
		log.Printf("desktop: x11vnc exited unexpectedly: %v", err)
	}
}

func (m *Manager) args(port int, viewOnly bool) []string {
	args := []string{
		"-display", m.cfg.Display,
		"-rfbport", strconv.Itoa(port),
		"-localhost",
		"-forever",
		"-shared",
		"-nopw",
		"-quiet",
		"-noxdamage",
	}
	if viewOnly {
		args = append(args, "-viewonly")
	}
	return args
}

func (m *Manager) env() []string {
	env := os.Environ()
	env = setEnv(env, "DISPLAY", m.cfg.Display)
	if m.cfg.XAUTHORITY != "" {
		env = setEnv(env, "XAUTHORITY", m.cfg.XAUTHORITY)
	}
	return env
}

var dimensionsRe = regexp.MustCompile(`dimensions:\s+(\d+)x(\d+)\s+pixels`)

func probeDisplay(ctx context.Context, display string) (uint16, uint16, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "xdpyinfo", "-display", display)
	out, err := cmd.Output()
	if err != nil {
		return 0, 0, err
	}
	match := dimensionsRe.FindSubmatch(out)
	if match == nil {
		return 0, 0, errors.New("xdpyinfo reported no dimensions")
	}
	width, _ := strconv.ParseUint(string(match[1]), 10, 16)
	height, _ := strconv.ParseUint(string(match[2]), 10, 16)
	return uint16(width), uint16(height), nil
}

func freePort() (int, error) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, fmt.Errorf("allocate VNC port: %w", err)
	}
	defer func() { _ = lis.Close() }()
	return lis.Addr().(*net.TCPAddr).Port, nil
}

func waitPort(ctx context.Context, port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		conn, err := net.DialTimeout(
			"tcp",
			net.JoinHostPort("127.0.0.1", strconv.Itoa(port)),
			time.Second,
		)
		if err == nil {
			_ = conn.Close()
			return nil
		}
		if time.Now().After(deadline) {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
}

func setEnv(env []string, key, value string) []string {
	prefix := key + "="
	for i, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			env[i] = prefix + value
			return env
		}
	}
	return append(env, prefix+value)
}
