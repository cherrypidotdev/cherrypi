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

type Kind int64

const (
	KindUnspecified Kind = 0

	KindRoot Kind = 1
)

var (
	ErrNotFound = errors.New("terminal not found")

	ErrForbidden = errors.New("terminal owned by another user")

	ErrInvalidKind = errors.New("terminal kind is required")
)

type Config struct {
	Shell string

	DefaultCols uint16
	DefaultRows uint16
}

type Info struct {
	ID        string
	Kind      Kind
	Owner     string
	Cwd       string
	Cols      uint16
	Rows      uint16
	CreatedAt time.Time
}

type Stats struct {
	Root  int
	Total int
}

const outboundBuffer = 128

const sinkWriteTimeout = 10 * time.Second

const maxOutputFrame = 32 * 1024

const outputFlushInterval = 8 * time.Millisecond

const outputChunkQueue = 32

type outbound struct {
	typ  websocket.MessageType
	data []byte
	done chan struct{}
}

type attachSink struct {
	conn   *websocket.Conn
	ctx    context.Context
	cancel context.CancelFunc
	out    chan outbound
}

type Terminal struct {
	ID   string
	Kind Kind

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

	done   chan struct{}
	onExit func(id string)
	onEnd  func(event, reason, detail string)
}

type Manager struct {
	cfg   Config
	store store.Store

	mu   sync.Mutex
	byID map[string]*Terminal
}

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

func (m *Manager) Get(id string) (Info, error) {
	m.mu.Lock()
	t := m.byID[id]
	m.mu.Unlock()
	if t == nil {
		return Info{}, ErrNotFound
	}
	return t.info(), nil
}

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

	for _, sink := range sinks {
		sink.cancel()
	}
	if ptmx != nil {
		_ = ptmx.Close()
	}
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}

	select {
	case <-t.done:
	case <-time.After(2 * time.Second):
	}
}

const scrollbackCap = 256 * 1024

// retained line may render oddly.
func (t *Terminal) appendScrollbackLocked(chunk []byte) {
	t.scrollback = append(t.scrollback, chunk)
	t.scrollbackBytes += len(chunk)
	for len(t.scrollback) > 1 && t.scrollbackBytes > scrollbackCap {
		t.scrollbackBytes -= len(t.scrollback[0])
		t.scrollback = t.scrollback[1:]
	}
}

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

			chunks <- chunk
		}
		if err != nil {
			close(chunks)
			return
		}
	}
}

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

func sendBlocking(sink *attachSink, frame outbound) {
	select {
	case sink.out <- frame:
	case <-sink.ctx.Done():
		if frame.done != nil {
			close(frame.done)
		}
	}
}

func trySend(sink *attachSink, typ websocket.MessageType, data []byte) {
	trySendFrame(sink, outbound{typ: typ, data: data})
}

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

func snapshotSinks(sinks map[*attachSink]struct{}) []*attachSink {
	out := make([]*attachSink, 0, len(sinks))
	for sink := range sinks {
		out = append(out, sink)
	}
	return out
}

var exitMessage = []byte(`{"type":"exit"}`)

type controlMsg struct {
	Type string `json:"type"`
	Cols uint16 `json:"cols,omitempty"`
	Rows uint16 `json:"rows,omitempty"`
	Data string `json:"data,omitempty"`
}

type controlState struct {
	Type       string `json:"type"`
	Controller bool   `json:"controller"`
	Cols       uint16 `json:"cols,omitempty"`
	Rows       uint16 `json:"rows,omitempty"`
}

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

	if t.controller == nil {
		t.controller = sink
	}
	isController := t.controller == sink
	ptmx := t.ptmx
	cols, rows := t.cols, t.rows
	exited := t.exited

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

	select {
	case <-exited:

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
