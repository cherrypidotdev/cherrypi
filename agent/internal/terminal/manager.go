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

// Package terminal brokers real PTYs to the browser. A terminal is a shell in
// the container — never the Pi process, which is headless in RPC mode. Control
// (create/list/resize/close) is exposed over gRPC; the PTY data plane is a
// WebSocket per terminal.
package terminal

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sort"
	"sync"
	"time"

	"cherrypi.vladfaust.com/agent/internal/store"
	"github.com/coder/websocket"
	"github.com/creack/pty"
)

// Kind mirrors agent.v1.TerminalKind. Never renumber.
type Kind int64

const (
	KindUnspecified Kind = 0
	// KindRoot is a shell in the container.
	KindRoot Kind = 1
)

var (
	// ErrNotFound is returned for an unknown terminal id.
	ErrNotFound = errors.New("terminal not found")
	// ErrForbidden is returned when an owner acts on another owner's terminal.
	ErrForbidden = errors.New("terminal owned by another user")
	// ErrInvalidKind is returned when a create request omits the kind.
	ErrInvalidKind = errors.New("terminal kind is required")
)

// Config controls how terminals spawn.
type Config struct {
	// Shell is the shell executable. Default: $SHELL, then /bin/bash.
	Shell string
	// DefaultCols/DefaultRows are the fallback PTY dimensions (80x24).
	DefaultCols uint16
	DefaultRows uint16
}

// Info is the public view of a terminal.
type Info struct {
	ID        string
	Kind      Kind
	Owner     string
	Cwd       string
	Cols      uint16
	Rows      uint16
	CreatedAt time.Time
}

// Stats counts a user's live terminals.
type Stats struct {
	Root  int
	Total int
}

// outboundBuffer bounds a consumer's pending-frame queue. When it fills, a
// read-only consumer is dropped rather than stalling the PTY reader and every
// other consumer. The controller is never dropped for falling behind: its
// queue applies backpressure to the PTY instead, so output is slowed rather
// than lost.
const outboundBuffer = 128

// sinkWriteTimeout bounds a single frame write to a consumer.
const sinkWriteTimeout = 10 * time.Second

// maxOutputFrame bounds one coalesced PTY frame. Coalescing keeps a chatty
// shell (many short lines, `yes`, a build log) from producing one WebSocket
// message per read, which used to overflow consumer queues and saturate the
// tunnel and the browser. It stays at 32 KiB, coder/websocket's default read
// limit, so a newer agent still talks to an older agency relay.
const maxOutputFrame = 32 * 1024

// outputFlushInterval bounds how long a partial frame waits for more output,
// so interactive echo is never held back by coalescing.
const outputFlushInterval = 8 * time.Millisecond

// outputChunkQueue bounds the PTY chunks waiting to be coalesced. When it
// fills, the PTY reader blocks and the shell stalls on a full PTY buffer, the
// same backpressure a physical terminal applies.
const outputChunkQueue = 32

// outbound is one queued frame for a consumer. When `done` is non-nil it is
// closed once the frame has been written (or abandoned), so a producer can
// flush a final frame before the socket closes.
type outbound struct {
	typ  websocket.MessageType
	data []byte
	done chan struct{}
}

// attachSink is one consumer of a terminal: a live view, plus the keyboard when
// it is the controller. Exactly one sink controls; the rest are read-only
// observers. Each sink has a single pump goroutine draining `out`, so frames
// reach it in order regardless of which producer queued them.
type attachSink struct {
	conn   *websocket.Conn
	ctx    context.Context
	cancel context.CancelFunc
	out    chan outbound
}

// Terminal is a live PTY plus its shell process.
type Terminal struct {
	ID   string
	Kind Kind
	// Owner is the authenticated username that created the terminal. It is
	// immutable after creation.
	Owner     string
	Cwd       string
	CreatedAt time.Time

	mu              sync.Mutex
	ptmx            *os.File
	cmd             *exec.Cmd
	cols            uint16
	rows            uint16
	sinks           map[*attachSink]struct{}
	controller      *attachSink
	scrollback      [][]byte
	scrollbackBytes int
	closed          bool
	closeReason     string
	exited          chan struct{}
	// done is closed at the very end of reap, once the final state and log have
	// been persisted, so Close can wait for a terminal to settle.
	done   chan struct{}
	onExit func(id string)
	onEnd  func(event, reason, detail string)
}

// Manager owns every terminal.
type Manager struct {
	cfg   Config
	store store.Store

	mu   sync.Mutex
	byID map[string]*Terminal
}

// NewManager builds a terminal manager.
func NewManager(cfg Config, st store.Store) *Manager {
	if cfg.Shell == "" {
		cfg.Shell = os.Getenv("SHELL")
	}
	if cfg.Shell == "" {
		cfg.Shell = "/bin/bash"
	}
	if cfg.DefaultCols == 0 {
		cfg.DefaultCols = 80
	}
	if cfg.DefaultRows == 0 {
		cfg.DefaultRows = 24
	}
	return &Manager{cfg: cfg, store: st, byID: map[string]*Terminal{}}
}

// Create spawns a shell for the requested kind. Opening it never touches Pi.
// Owner is the authenticated username the terminal belongs to.
func (m *Manager) Create(
	ctx context.Context,
	kind Kind,
	cols, rows uint16,
	owner string,
) (Info, error) {
	if kind != KindRoot {
		return Info{}, ErrInvalidKind
	}
	if cols == 0 {
		cols = m.cfg.DefaultCols
	}
	if rows == 0 {
		rows = m.cfg.DefaultRows
	}

	cwd := ""
	if home, err := os.UserHomeDir(); err == nil {
		cwd = home
	}

	if cwd == "" {
		cwd = "/"
	}

	cmd := exec.Command(m.cfg.Shell)
	cmd.Dir = cwd
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: cols, Rows: rows})
	if err != nil {
		return Info{}, fmt.Errorf("start shell: %w", err)
	}

	t := &Terminal{
		ID:        newID(),
		Kind:      kind,
		Owner:     owner,
		Cwd:       cwd,
		CreatedAt: time.Now(),
		ptmx:      ptmx,
		cmd:       cmd,
		cols:      cols,
		rows:      rows,
		sinks:     map[*attachSink]struct{}{},
		exited:    make(chan struct{}),
		done:      make(chan struct{}),
		onExit:    m.remove,
	}
	t.onEnd = func(event, reason, detail string) {
		m.finish(t, event, reason, detail)
	}

	m.mu.Lock()
	m.byID[t.ID] = t
	m.mu.Unlock()

	// Persist the record and its creation event so the terminal survives (as
	// metadata) even if the agent restarts and the PTY is lost.
	if err := m.store.CreateTerminalRecord(
		context.Background(),
		t.ID,
		int64(kind),
		owner,
		cwd,
		int64(cols),
		int64(rows),
		t.CreatedAt.UnixMilli(),
	); err != nil {
		log.Printf("[terminal %s] persist create: %v", t.ID, err)
	}
	m.logEvent(t.ID, kind, owner, "created", "", cwd)

	go t.reap()
	go t.readLoop(ptmx)

	return t.info(), nil
}

// Get returns a terminal's public info. It does not enforce ownership; callers
// that act on behalf of a user must check Info.Owner (or use the manager
// methods, which do).
func (m *Manager) Get(id string) (Info, error) {
	m.mu.Lock()
	t := m.byID[id]
	m.mu.Unlock()
	if t == nil {
		return Info{}, ErrNotFound
	}
	return t.info(), nil
}

// List returns live terminals, oldest first. An empty owner lists every
// terminal (server-internal callers); otherwise only that owner's.
func (m *Manager) List(owner string) []Info {
	m.mu.Lock()
	out := make([]Info, 0, len(m.byID))
	for _, t := range m.byID {
		if !t.ownedBy(owner) {
			continue
		}
		out = append(out, t.info())
	}
	m.mu.Unlock()

	sortInfos(out)
	return out
}

// Stats counts owner's live terminals (all owners when owner is empty).
func (m *Manager) Stats(owner string) Stats {
	m.mu.Lock()
	defer m.mu.Unlock()

	var s Stats
	for _, t := range m.byID {
		if !t.ownedBy(owner) {
			continue
		}
		if t.Kind == KindRoot {
			s.Root++
		}
	}
	s.Total = s.Root
	return s
}

// Resize changes a terminal's PTY dimensions. Only the owner may resize.
func (m *Manager) Resize(id string, cols, rows uint16, owner string) error {
	m.mu.Lock()
	t := m.byID[id]
	m.mu.Unlock()
	if t == nil {
		return ErrNotFound
	}
	if !t.ownedBy(owner) {
		return ErrForbidden
	}
	return t.resize(cols, rows)
}

// Close kills a terminal's shell and forgets it. Idempotent for unknown ids;
// only the owner may close.
func (m *Manager) Close(id, owner string) error {
	m.mu.Lock()
	t := m.byID[id]
	if t != nil && t.ownedBy(owner) {
		delete(m.byID, id)
	}
	m.mu.Unlock()
	if t == nil {
		return nil
	}
	if !t.ownedBy(owner) {
		return ErrForbidden
	}
	t.close("closed by user")
	return nil
}

// CloseFor closes every terminal matching the owner (all owners when empty)
// and kind (every kind when Unspecified). It returns how many were closed.
func (m *Manager) CloseFor(owner string, kind Kind) int {
	m.mu.Lock()
	var closed []*Terminal
	for id, t := range m.byID {
		if !t.ownedBy(owner) {
			continue
		}
		if kind != KindUnspecified && t.Kind != kind {
			continue
		}
		delete(m.byID, id)
		closed = append(closed, t)
	}
	m.mu.Unlock()

	for _, t := range closed {
		t.close("closed by user")
	}
	return len(closed)
}

// CloseAll kills every terminal. Used at shutdown.
func (m *Manager) CloseAll() {
	m.mu.Lock()
	all := make([]*Terminal, 0, len(m.byID))
	for _, t := range m.byID {
		all = append(all, t)
	}
	m.byID = map[string]*Terminal{}
	m.mu.Unlock()

	for _, t := range all {
		t.close("agent shutting down")
	}
}

// Recover marks terminals left open by a previous process as interrupted, so
// the dashboard can show they were lost (and offer to restore them).
func (m *Manager) Recover(ctx context.Context) error {
	open, err := m.store.ListTerminalsByState(ctx, "open")
	if err != nil {
		return err
	}
	if len(open) == 0 {
		return nil
	}
	now := time.Now().UnixMilli()
	for _, row := range open {
		if err := m.store.SetTerminalState(
			ctx,
			row.ID,
			"interrupted",
			"agent restarted",
			now,
		); err != nil {
			log.Printf("[terminal %s] mark interrupted: %v", row.ID, err)
			continue
		}
		m.logEvent(
			row.ID,
			Kind(row.Kind),
			row.Owner,
			"interrupted",
			"agent restarted",
			row.Cwd,
		)
	}
	log.Printf("Marked %d terminal(s) interrupted after restart", len(open))
	return nil
}

// LogEntry is one terminal lifecycle event.
type LogEntry struct {
	ID         int64
	TerminalID string
	Kind       Kind
	Owner      string
	Event      string
	Reason     string
	Detail     string
	At         time.Time
}

// Logs returns the terminal lifecycle log, newest first.
func (m *Manager) Logs(ctx context.Context, limit int64) ([]LogEntry, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	rows, err := m.store.ListTerminalLogs(ctx, limit)
	if err != nil {
		return nil, err
	}
	out := make([]LogEntry, len(rows))
	for i, r := range rows {
		out[i] = LogEntry{
			ID:         r.ID,
			TerminalID: r.TerminalID,
			Kind:       Kind(r.Kind),
			Owner:      r.Owner,
			Event:      r.Event,
			Reason:     r.Reason,
			Detail:     r.Detail,
			At:         time.UnixMilli(r.At),
		}
	}
	return out, nil
}

// logEvent appends one lifecycle event. Best-effort: a store failure must not
// fail the terminal operation.
func (m *Manager) logEvent(
	terminalID string,
	kind Kind,
	owner, event, reason, detail string,
) {
	if err := m.store.InsertTerminalLog(
		context.Background(),
		terminalID,
		int64(kind),
		owner,
		event,
		reason,
		detail,
		time.Now().UnixMilli(),
	); err != nil {
		log.Printf("[terminal %s] log %s: %v", terminalID, event, err)
	}
}

// finish persists a terminal's terminal state and logs the closing event.
func (m *Manager) finish(t *Terminal, event, reason, detail string) {
	now := time.Now().UnixMilli()
	state := "closed"
	if event == "interrupted" {
		state = "interrupted"
	}
	if err := m.store.SetTerminalState(context.Background(), t.ID, state, reason, now); err != nil {
		log.Printf("[terminal %s] persist %s: %v", t.ID, event, err)
	}
	m.logEvent(t.ID, t.Kind, t.Owner, event, reason, detail)
}

func (m *Manager) remove(id string) {
	m.mu.Lock()
	delete(m.byID, id)
	m.mu.Unlock()
}

// ownedBy reports whether owner may control the terminal. An empty owner means
// no auth context (server-internal callers and tests) and is allowed.
func (t *Terminal) ownedBy(owner string) bool {
	return owner == "" || t.Owner == owner
}

func (t *Terminal) info() Info {
	t.mu.Lock()
	defer t.mu.Unlock()
	return Info{
		ID:        t.ID,
		Kind:      t.Kind,
		Owner:     t.Owner,
		Cwd:       t.Cwd,
		Cols:      t.cols,
		Rows:      t.rows,
		CreatedAt: t.CreatedAt,
	}
}

// reap waits for the shell to exit and forgets the terminal. A shell that
// exits on its own logs `exited` (with the exit status); one killed by Close
// logs `closed` (with the close reason).
func (t *Terminal) reap() {
	exitReason := "shell exited"
	if t.cmd != nil {
		if err := t.cmd.Wait(); err != nil {
			exitReason = err.Error()
		}
	}

	t.mu.Lock()
	ptmx := t.ptmx
	t.ptmx = nil
	event, reason := "exited", exitReason
	if t.closed {
		event, reason = "closed", t.closeReason
	}
	t.mu.Unlock()
	if ptmx != nil {
		_ = ptmx.Close()
	}

	select {
	case <-t.exited:
	default:
		close(t.exited)
	}

	if t.onEnd != nil {
		t.onEnd(event, reason, t.Cwd)
	}
	if t.onExit != nil {
		t.onExit(t.ID)
	}
	close(t.done)
}

func (t *Terminal) resize(cols, rows uint16) error {
	if cols == 0 || rows == 0 {
		return nil
	}

	t.mu.Lock()
	ptmx := t.ptmx
	t.cols, t.rows = cols, rows
	t.mu.Unlock()
	if ptmx == nil {
		return ErrNotFound
	}
	return pty.Setsize(ptmx, &pty.Winsize{Cols: cols, Rows: rows})
}

func (t *Terminal) close(reason string) {
	t.mu.Lock()
	closed := t.closed
	t.closed = true
	t.closeReason = reason
	cmd := t.cmd
	ptmx := t.ptmx
	sinks := snapshotSinks(t.sinks)
	t.sinks = nil
	t.controller = nil
	t.mu.Unlock()
	if closed {
		return
	}

	// Detach every consumer so its pump goroutine unblocks.
	for _, sink := range sinks {
		sink.cancel()
	}
	if ptmx != nil {
		_ = ptmx.Close()
	}
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}

	// Wait for the reaper so the closing state and log are persisted before
	// Close returns (shutdown and tests rely on it being settled).
	select {
	case <-t.done:
	case <-time.After(2 * time.Second):
	}
}

// scrollbackCap bounds the per-terminal output retained for consumers that
// attach later.
const scrollbackCap = 256 * 1024

// appendScrollbackLocked retains recent output so a consumer that attaches
// later can be caught up. Whole chunks are dropped oldest-first once the cap is
// exceeded; a chunk boundary can split an escape sequence, so the oldest
// retained line may render oddly.
func (t *Terminal) appendScrollbackLocked(chunk []byte) {
	t.scrollback = append(t.scrollback, chunk)
	t.scrollbackBytes += len(chunk)
	for len(t.scrollback) > 1 && t.scrollbackBytes > scrollbackCap {
		t.scrollbackBytes -= len(t.scrollback[0])
		t.scrollback = t.scrollback[1:]
	}
}

// scrollbackSnapshotLocked concatenates the retained output into one frame.
func (t *Terminal) scrollbackSnapshotLocked() []byte {
	if t.scrollbackBytes == 0 {
		return nil
	}
	out := make([]byte, 0, t.scrollbackBytes)
	for _, chunk := range t.scrollback {
		out = append(out, chunk...)
	}
	return out
}

// readLoop is the terminal's single PTY reader. It hands output to the
// coalescer, which broadcasts it to every consumer (and retains it for late
// ones). One reader means consumers always see bytes in the same order.
func (t *Terminal) readLoop(ptmx *os.File) {
	buf := make([]byte, 32*1024)
	chunks := make(chan []byte, outputChunkQueue)
	go t.broadcast(chunks)

	for {
		n, err := ptmx.Read(buf)
		if n > 0 {
			chunk := append([]byte(nil), buf[:n]...)
			t.mu.Lock()
			t.appendScrollbackLocked(chunk)
			t.mu.Unlock()
			// Blocking when the coalescer is behind throttles the PTY, so the
			// shell slows down instead of the consumer losing output.
			chunks <- chunk
		}
		if err != nil {
			close(chunks)
			return
		}
	}
}

// broadcast coalesces PTY chunks into frames of up to maxOutputFrame and fans
// each frame out to the consumers. A partial frame is flushed after
// outputFlushInterval, so a burst of short lines arrives as one message per
// flush rather than one message per line.
func (t *Terminal) broadcast(chunks <-chan []byte) {
	frame := make([]byte, 0, maxOutputFrame)

	timer := time.NewTimer(outputFlushInterval)
	if !timer.Stop() {
		<-timer.C
	}
	defer timer.Stop()
	var deadline <-chan time.Time

	arm := func() {
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		timer.Reset(outputFlushInterval)
		deadline = timer.C
	}

	flush := func() {
		if len(frame) == 0 {
			return
		}
		data := frame
		frame = make([]byte, 0, maxOutputFrame)
		t.fanout(websocket.MessageBinary, data)
	}

	for {
		select {
		case chunk, ok := <-chunks:
			if !ok {
				flush()
				return
			}
			for len(chunk) > 0 {
				n := min(maxOutputFrame-len(frame), len(chunk))
				frame = append(frame, chunk[:n]...)
				chunk = chunk[n:]
				if len(frame) >= maxOutputFrame {
					flush()
				}
			}
			if len(frame) > 0 && deadline == nil {
				arm()
			}
		case <-deadline:
			deadline = nil
			flush()
		}
	}
}

// fanout delivers one coalesced output frame to every consumer. The
// controller is queued with backpressure, so a slow browser slows the shell
// (as a physical terminal does) instead of losing output. An observer that
// cannot keep up is dropped, because it must never stall the controller.
func (t *Terminal) fanout(typ websocket.MessageType, data []byte) {
	t.mu.Lock()
	sinks := snapshotSinks(t.sinks)
	controller := t.controller
	t.mu.Unlock()

	for _, sink := range sinks {
		frame := outbound{typ: typ, data: data}
		if sink == controller {
			sendBlocking(sink, frame)
			continue
		}
		trySendFrame(sink, frame)
	}
}

// sendBlocking queues a frame, waiting for space when the consumer is behind.
// A cancelled sink (a closed socket or a write timeout) unblocks it.
func sendBlocking(sink *attachSink, frame outbound) {
	select {
	case sink.out <- frame:
	case <-sink.ctx.Done():
		if frame.done != nil {
			close(frame.done)
		}
	}
}

// trySend queues one frame for a consumer without blocking the producer. A
// consumer too far behind to keep up is dropped rather than stalling the PTY.
func trySend(sink *attachSink, typ websocket.MessageType, data []byte) {
	trySendFrame(sink, outbound{typ: typ, data: data})
}

// trySendFrame queues a prepared frame; see trySend.
func trySendFrame(sink *attachSink, frame outbound) {
	select {
	case sink.out <- frame:
	case <-sink.ctx.Done():
		if frame.done != nil {
			close(frame.done)
		}
	default:
		if frame.done != nil {
			close(frame.done)
		}
		sink.cancel()
	}
}

// pump writes a consumer's queued frames in order. It is the only writer for a
// sink, so a single goroutine can never interleave frames from two producers.
func (t *Terminal) pump(sink *attachSink) {
	defer sink.cancel()
	for {
		select {
		case <-sink.ctx.Done():
			return
		case frame := <-sink.out:
			ctx, cancel := context.WithTimeout(sink.ctx, sinkWriteTimeout)
			err := sink.conn.Write(ctx, frame.typ, frame.data)
			cancel()
			if frame.done != nil {
				close(frame.done)
			}
			if err != nil {
				return
			}
		}
	}
}

// snapshotSinks copies a sink set so callers can fan out without holding the
// terminal lock.
func snapshotSinks(sinks map[*attachSink]struct{}) []*attachSink {
	out := make([]*attachSink, 0, len(sinks))
	for sink := range sinks {
		out = append(out, sink)
	}
	return out
}

// exitMessage is the text control frame the agent sends when a shell exits, so
// a client can tell a real exit from a takeover or disconnect. The PTY stream
// itself is always binary, so text frames are free for control.
var exitMessage = []byte(`{"type":"exit"}`)

// controlMsg is the JSON payload of a text frame from the browser.
type controlMsg struct {
	Type string `json:"type"`
	Cols uint16 `json:"cols,omitempty"`
	Rows uint16 `json:"rows,omitempty"`
	Data string `json:"data,omitempty"`
}

// controlState is the JSON payload of a text control frame to the browser: the
// whether it controls the terminal and the PTY's current size.
type controlState struct {
	Type       string `json:"type"`
	Controller bool   `json:"controller"`
	Cols       uint16 `json:"cols,omitempty"`
	Rows       uint16 `json:"rows,omitempty"`
}

// Attach bridges a WebSocket to the terminal's PTY until either side ends.
// Several consumers may attach at once: the first controls, later ones watch
// until they ask to take control. Only the owner may attach.
func (m *Manager) Attach(ctx context.Context, id, owner string, conn *websocket.Conn) error {
	m.mu.Lock()
	t := m.byID[id]
	m.mu.Unlock()
	if t == nil {
		return ErrNotFound
	}
	if !t.ownedBy(owner) {
		return ErrForbidden
	}
	return t.attach(ctx, conn)
}

func (t *Terminal) attach(ctx context.Context, conn *websocket.Conn) error {
	conn.SetReadLimit(1 << 20)

	attachCtx, cancel := context.WithCancel(ctx)
	sink := &attachSink{
		conn:   conn,
		ctx:    attachCtx,
		cancel: cancel,
		out:    make(chan outbound, outboundBuffer),
	}

	t.mu.Lock()
	if t.closed || t.ptmx == nil {
		t.mu.Unlock()
		cancel()
		return ErrNotFound
	}
	// The first consumer controls; later consumers watch until they ask for
	// control, so opening a second window never silently steals the keyboard.
	if t.controller == nil {
		t.controller = sink
	}
	isController := t.controller == sink
	ptmx := t.ptmx
	cols, rows := t.cols, t.rows
	exited := t.exited
	// Queue the replay and the role frame *before* joining the broadcast set,
	// so live output can never slip ahead of the history.
	if scrollback := t.scrollbackSnapshotLocked(); len(scrollback) > 0 {
		for len(scrollback) > 0 {
			n := min(maxOutputFrame, len(scrollback))
			trySend(sink, websocket.MessageBinary, scrollback[:n])
			scrollback = scrollback[n:]
		}
	}
	if payload, err := json.Marshal(controlState{
		Type:       "control",
		Controller: isController,
		Cols:       cols,
		Rows:       rows,
	}); err == nil {
		trySend(sink, websocket.MessageText, payload)
	}
	t.sinks[sink] = struct{}{}
	t.mu.Unlock()

	go t.pump(sink)

	defer func() {
		t.mu.Lock()
		delete(t.sinks, sink)
		if t.controller == sink {
			t.controller = nil
		}
		t.mu.Unlock()
		cancel()
	}()

	// Reapplying the current size makes the shell redraw on reconnect.
	_ = pty.Setsize(ptmx, &pty.Winsize{Cols: cols, Rows: rows})

	errCh := make(chan error, 2)

	go func() {
		for {
			typ, data, err := conn.Read(attachCtx)
			if err != nil {
				errCh <- err
				return
			}
			switch typ {
			case websocket.MessageText:
				var msg controlMsg
				if json.Unmarshal(data, &msg) != nil {
					continue
				}
				switch msg.Type {
				case "resize":
					t.resizeFrom(sink, msg.Cols, msg.Rows)
				case "input":
					t.writeInput(sink, []byte(msg.Data))
				case "take-control":
					t.takeControl(sink)
				}
			case websocket.MessageBinary:
				t.writeInput(sink, data)
			}
		}
	}()

	go func() {
		select {
		case <-exited:
		case <-attachCtx.Done():
		}
		errCh <- nil
	}()

	err := <-errCh
	// A bare socket close is ambiguous: it can mean the shell exited, the page
	// reloaded, or a consumer dropped off. Only the shell's own exit is
	// terminal, so announce it explicitly; the client keeps the pane on any
	// other close and can reconnect later.
	select {
	case <-exited:
		// Flush the exit frame through the pump before closing, so the client
		// always learns the shell exited (rather than seeing a bare close).
		done := make(chan struct{})
		trySendFrame(sink, outbound{typ: websocket.MessageText, data: exitMessage, done: done})
		select {
		case <-done:
		case <-time.After(2 * time.Second):
		}
	default:
	}
	cancel()
	_ = conn.Close(websocket.StatusNormalClosure, "")
	return err
}

// writeInput forwards bytes to the PTY, but only when sink controls the
// terminal; observers are read-only.
func (t *Terminal) writeInput(sink *attachSink, data []byte) {
	t.mu.Lock()
	ptmx := t.ptmx
	allowed := !t.closed && t.controller == sink
	t.mu.Unlock()
	if !allowed || ptmx == nil {
		return
	}
	_, _ = ptmx.Write(data)
}

// resizeFrom applies a resize only from the controller, then tells the other
// consumers the new size so their rendering matches the PTY.
func (t *Terminal) resizeFrom(sink *attachSink, cols, rows uint16) {
	if cols == 0 || rows == 0 {
		return
	}
	t.mu.Lock()
	allowed := !t.closed && t.controller == sink
	t.mu.Unlock()
	if !allowed {
		return
	}
	if err := t.resize(cols, rows); err != nil {
		return
	}
	t.broadcastControl(cols, rows)
}

// takeControl hands the keyboard to sink and tells every consumer about it, so
// the displaced controller can drop to read-only.
func (t *Terminal) takeControl(sink *attachSink) {
	t.mu.Lock()
	if t.closed || t.ptmx == nil {
		t.mu.Unlock()
		return
	}
	t.controller = sink
	cols, rows := t.cols, t.rows
	t.mu.Unlock()
	t.broadcastControl(cols, rows)
}

// broadcastControl tells every consumer its role and the PTY's current size.
func (t *Terminal) broadcastControl(cols, rows uint16) {
	t.mu.Lock()
	sinks := snapshotSinks(t.sinks)
	controller := t.controller
	t.mu.Unlock()

	for _, sink := range sinks {
		payload, err := json.Marshal(controlState{
			Type:       "control",
			Controller: sink == controller,
			Cols:       cols,
			Rows:       rows,
		})
		if err != nil {
			continue
		}
		trySend(sink, websocket.MessageText, payload)
	}
}

func newID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b[:])
}

func sortInfos(infos []Info) {
	sort.Slice(infos, func(i, j int) bool { return infos[i].CreatedAt.Before(infos[j].CreatedAt) })
}
