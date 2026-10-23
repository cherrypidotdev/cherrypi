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

// Package desktop exposes the agent's already-running X desktop to the
// browser. CherryPi does not boot a desktop session: the operator provides one
// (the host user's Xfce/X11 session, or a container with the host's X socket
// and XAUTHORITY mounted in) and the agent attaches x11vnc to that display,
// bridging RFB to the browser over the tunnel.
//
// There is exactly one desktop per agent, shared by every authenticated user —
// unlike terminals, which are per-user. The VNC bridge is started lazily on
// the first WebSocket attach and torn down with the agent.
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

// ErrUnavailable is returned when no X display is reachable or the VNC bridge
// cannot start.
var ErrUnavailable = errors.New("desktop is not available")

// Config controls how the desktop bridge spawns.
type Config struct {
	// Display is the X display to attach to. Empty disables the desktop.
	// Default: $DISPLAY.
	Display string
	// XAUTHORITY is the X authority file for the display. Empty inherits the
	// agent's environment (default: $XAUTHORITY).
	XAUTHORITY string

	// X11VNCBin is the VNC server binary. Default: x11vnc.
	X11VNCBin string
	// XdpyinfoBin probes the display geometry. Default: xdpyinfo.
	XdpyinfoBin string

	// VNCPort is the loopback RFB port. 0 (the default) picks a free port.
	VNCPort uint16

	// StartupTimeout bounds waiting for the display probe and for the VNC
	// bridge to accept connections. Default 10s.
	StartupTimeout time.Duration

	// probe overrides display probing (tests).
	probe func(ctx context.Context, display string) (uint16, uint16, error)

	// ready overrides waiting for the VNC bridge to accept (tests).
	ready func(ctx context.Context, port int) error
}

// Info is the public view of the desktop.
type Info struct {
	Display  string
	Width    uint16
	Height   uint16
	Attached bool
}

// bridge is one x11vnc process and its loopback port.
type bridge struct {
	cmd  *exec.Cmd
	port int
}

// Manager owns the agent's VNC bridges to the running X session. Owners get a
// read-write bridge; everyone else shares a second, view-only bridge so the
// input restriction is enforced by x11vnc rather than only by noVNC.
type Manager struct {
	cfg Config

	mu      sync.Mutex
	bridges map[bool]*bridge
}

// NewManager builds a desktop manager, resolving defaults.
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

// Display returns the configured X display (possibly empty).
func (m *Manager) Display() string { return m.cfg.Display }

// Availability reports whether the desktop can be served: an X display is
// configured and reachable, and x11vnc is installed.
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

// Info returns the desktop's display, geometry, and whether the VNC bridge is
// currently attached. The second result is false when the display is not
// reachable.
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

// Ensure starts the requested VNC bridge if it is not already running and
// returns its loopback port. It is idempotent per mode; concurrent callers
// share one bridge. viewOnly selects the bridge x11vnc starts with -viewonly.
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
	// Run the bridge in its own process group so teardown kills it and any
	// helpers it spawned.
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

// Stop tears down every VNC bridge. It is a no-op when none is running.
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

// args is the x11vnc command line. The bridge listens on loopback only; the
// agency authenticates the browser and relays the RFB bytes, so no VNC
// password is set here. The view-only bridge additionally refuses pointer and
// keyboard input at the VNC server.
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

// env is the environment the VNC bridge inherits: the agent's own, with the
// target display (and authority file, when set) forced.
func (m *Manager) env() []string {
	env := os.Environ()
	env = setEnv(env, "DISPLAY", m.cfg.Display)
	if m.cfg.XAUTHORITY != "" {
		env = setEnv(env, "XAUTHORITY", m.cfg.XAUTHORITY)
	}
	return env
}

// dimensionsRe matches `xdpyinfo`'s geometry line.
var dimensionsRe = regexp.MustCompile(`dimensions:\s+(\d+)x(\d+)\s+pixels`)

// probeDisplay runs xdpyinfo against the display and parses its geometry.
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

// freePort asks the kernel for an unused loopback port.
func freePort() (int, error) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, fmt.Errorf("allocate VNC port: %w", err)
	}
	defer func() { _ = lis.Close() }()
	return lis.Addr().(*net.TCPAddr).Port, nil
}

// waitPort waits until something accepts on the loopback port.
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

// setEnv sets key=value in an env slice, replacing an existing entry.
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
