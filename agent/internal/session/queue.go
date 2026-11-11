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

package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"os"
	"time"

	sqlitedb "cherrypi.vladfaust.com/agent/internal/gen/sqlc"
)

type queueEntry struct {
	QueuedPrompt

	images []imageAttachment

	inPi bool

	reported bool
}

func queueID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b[:])
}

func queueContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 2*time.Second)
}

func decodeClearedQueue(data json.RawMessage) ([]string, error) {
	var handed struct {
		Steering *[]string `json:"steering"`
	}
	if err := json.Unmarshal(data, &handed); err != nil {
		return nil, err
	}
	if handed.Steering == nil {
		return nil, errors.New("reply has no steering list")
	}
	return *handed.Steering, nil
}

func (s *Session) queueEventLocked() QueueEvent {
	steering := make([]QueuedPrompt, len(s.queue))
	for i, entry := range s.queue {
		steering[i] = entry.QueuedPrompt
	}
	return QueueEvent{Steering: steering}
}

func (s *Session) publishQueue() {
	s.mu.Lock()
	queue := s.queueEventLocked()
	s.mu.Unlock()

	s.persistQueue()
	s.emit(Event{Kind: EventQueue, Queue: &queue})
}

func (s *Session) forgetQueued(id string) {
	s.mu.Lock()
	kept := make([]queueEntry, 0, len(s.queue))
	for _, entry := range s.queue {
		if entry.ID != id {
			kept = append(kept, entry)
		}
	}
	s.queue = kept
	s.mu.Unlock()

	s.publishQueue()
}

func (s *Session) mirrorQueued(entry QueuedPrompt, images []imageAttachment) QueuedPrompt {
	if entry.ID == "" {
		entry.ID = queueID()
	}
	s.mu.Lock()
	found := false
	for i := range s.queue {
		if s.queue[i].ID == entry.ID {
			s.queue[i].inPi = true
			s.queue[i].reported = false
			found = true
			break
		}
	}
	if !found {
		s.queue = append(s.queue, queueEntry{
			QueuedPrompt: entry,
			images:       images,
			inPi:         true,
		})
	}
	s.mu.Unlock()

	s.publishQueue()
	return entry
}

func (s *Session) setQueue(steering []string) {
	s.mu.Lock()
	changed := s.reconcileQueueLocked(steering)
	s.mu.Unlock()

	if !changed {
		return
	}
	s.publishQueue()
}

func (s *Session) reconcileQueueLocked(reported []string) bool {

	var held []queueEntry
	live := make(map[string][]queueEntry, len(s.queue))
	for _, entry := range s.queue {
		if !entry.inPi {
			held = append(held, entry)
			continue
		}
		live[entry.Text] = append(live[entry.Text], entry)
	}

	changed := false
	next := make([]queueEntry, 0, len(reported)+len(held))
	for _, text := range reported {
		entries := live[text]
		if len(entries) > 0 {

			entry := entries[0]
			entry.reported = true
			next = append(next, entry)
			live[text] = entries[1:]
			continue
		}
		next = append(next, queueEntry{
			QueuedPrompt: QueuedPrompt{ID: queueID(), Text: text},
			inPi:         true,
			reported:     true,
		})
		changed = true
	}
	for _, entries := range live {
		for _, entry := range entries {
			if entry.reported {

				changed = true
				continue
			}
			next = append(next, entry)
		}
	}
	s.queue = append(held, next...)
	return changed
}

func (s *Session) holdQueue() {
	s.mu.Lock()
	hasProcess := s.stdin != nil
	live := make(map[string]bool, len(s.queue))
	for _, entry := range s.queue {
		if entry.inPi {
			live[entry.ID] = true
		}
	}

	for i := range s.queue {
		s.queue[i].inPi = false
	}
	s.mu.Unlock()

	if !hasProcess {

		s.persistQueue()
		return
	}
	if len(live) == 0 {
		return
	}

	ctx, cancel := queueContext()
	data, err := s.send(ctx, "clear_queue", nil)
	cancel()
	if err != nil {

		log.Printf("[session %s] hold queue: %v", s.ID, err)
		s.persistQueue()
		return
	}

	reported, err := decodeClearedQueue(data)
	if err != nil {

		log.Printf("[session %s] decode cleared queue: %v", s.ID, err)
		s.persistQueue()
		return
	}

	s.mu.Lock()

	remaining := make(map[string]int, len(reported))
	for _, text := range reported {
		remaining[text]++
	}
	kept := make([]queueEntry, 0, len(s.queue)+len(reported))
	for _, entry := range s.queue {
		if !live[entry.ID] {
			kept = append(kept, entry)
			continue
		}
		if remaining[entry.Text] > 0 {
			remaining[entry.Text]--
			kept = append(kept, entry)
		}
	}

	for text, count := range remaining {
		for range count {
			kept = append(kept, queueEntry{
				QueuedPrompt: QueuedPrompt{ID: queueID(), Text: text},
			})
		}
	}
	s.queue = kept
	s.mu.Unlock()

	s.publishQueue()
}

func (s *Session) markQueueHeld() {
	s.mu.Lock()
	for i := range s.queue {
		s.queue[i].inPi = false
	}
	s.mu.Unlock()
}

func (s *Session) takeHeldQueue() []pendingDelivery {
	s.mu.Lock()
	defer s.mu.Unlock()
	var held []pendingDelivery
	kept := make([]queueEntry, 0, len(s.queue))
	for _, entry := range s.queue {
		if entry.inPi {
			kept = append(kept, entry)
			continue
		}
		held = append(held, pendingDelivery{
			entry:  entry.QueuedPrompt,
			images: entry.images,
		})
	}
	s.queue = kept
	return held
}

func (s *Session) restoreHeld(deliveries []pendingDelivery) {
	if len(deliveries) == 0 {
		return
	}
	s.mu.Lock()
	restored := make([]queueEntry, 0, len(deliveries)+len(s.queue))
	for _, delivery := range deliveries {
		entry := delivery.entry
		if entry.Text == "" {
			continue
		}
		if entry.ID == "" {
			entry.ID = queueID()
		}
		restored = append(restored, queueEntry{
			QueuedPrompt: entry,
			images:       delivery.images,
		})
	}
	s.queue = append(restored, s.queue...)
	s.mu.Unlock()

	s.publishQueue()
}

func (s *Session) steerQueued(
	ctx context.Context,
	entry QueuedPrompt,
	images []imageAttachment,
) error {

	images = reloadImages(images)
	entry = s.mirrorQueued(entry, images)
	if err := s.steer(ctx, entry.Text, images); err != nil {
		s.forgetQueued(entry.ID)
		return err
	}
	return nil
}

func (s *Session) dropQueued(id string) bool {
	s.mu.Lock()
	found := false
	live := false
	kept := make([]queueEntry, 0, len(s.queue))
	for _, entry := range s.queue {
		if entry.ID == id {
			found = true
			live = live || entry.inPi
			continue
		}
		live = live || entry.inPi
		kept = append(kept, entry)
	}
	if !found {
		s.mu.Unlock()
		return false
	}
	s.queue = kept
	s.mu.Unlock()

	s.publishQueue()
	if live {
		s.requeueLive()
	}
	return true
}

func (s *Session) clearQueued() {
	s.mu.Lock()
	empty := len(s.queue) == 0
	hasProcess := s.stdin != nil
	s.queue = nil
	s.mu.Unlock()

	if hasProcess {

		ctx, cancel := queueContext()
		if _, err := s.send(ctx, "clear_queue", nil); err != nil {
			log.Printf("[session %s] clear queue: %v", s.ID, err)
		}
		cancel()
	}
	if empty {
		return
	}
	s.publishQueue()
}

func (s *Session) requeueLive() {
	s.mu.Lock()
	hasProcess := s.stdin != nil
	live := make(map[string]bool, len(s.queue))
	for _, entry := range s.queue {
		if entry.inPi {
			live[entry.ID] = true
		}
	}

	for i := range s.queue {
		s.queue[i].inPi = false
	}
	s.mu.Unlock()

	if !hasProcess {
		s.publishQueue()
		return
	}

	ctx, cancel := queueContext()
	data, err := s.send(ctx, "clear_queue", nil)
	if err != nil {
		cancel()
		log.Printf("[session %s] requeue: clear queue: %v", s.ID, err)
		s.publishQueue()
		return
	}

	reported, err := decodeClearedQueue(data)
	if err != nil {
		cancel()

		log.Printf("[session %s] requeue: decode cleared queue: %v", s.ID, err)
		s.publishQueue()
		return
	}

	s.mu.Lock()
	remaining := make(map[string]int, len(reported))
	for _, text := range reported {
		remaining[text]++
	}
	kept := make([]queueEntry, 0, len(s.queue))
	var deliver []pendingDelivery
	for _, entry := range s.queue {
		if !live[entry.ID] {
			kept = append(kept, entry)
			continue
		}
		if remaining[entry.Text] == 0 {
			continue
		}
		remaining[entry.Text]--
		deliver = append(deliver, pendingDelivery{
			entry:  entry.QueuedPrompt,
			images: entry.images,
		})
	}

	s.queue = kept
	s.mu.Unlock()

	if len(deliver) == 0 {
		s.publishQueue()
	}
	for i, delivery := range deliver {
		if err := s.steerQueued(ctx, delivery.entry, delivery.images); err != nil {
			log.Printf("[session %s] requeue steer: %v", s.ID, err)
			s.restoreHeld(deliver[i:])
			break
		}
	}
	cancel()
	s.persistQueue()
}

func (s *Session) steer(ctx context.Context, text string, images []imageAttachment) error {
	fields := map[string]any{"message": text}
	if len(images) > 0 {
		fields["images"] = imageParts(images)
	}
	_, err := s.send(ctx, "steer", fields)
	return err
}

type persistedEntry struct {
	QueuedPrompt
	Images []persistedImage `json:"images,omitempty"`
}

type persistedImage struct {
	Path     string `json:"path"`
	MimeType string `json:"mimeType"`
}

func (s *Session) persistedEntries() []persistedEntry {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]persistedEntry, len(s.queue))
	for i, entry := range s.queue {
		out[i] = persistedEntry{QueuedPrompt: entry.QueuedPrompt}
		for _, image := range entry.images {
			if image.path == "" {
				continue
			}
			out[i].Images = append(out[i].Images, persistedImage{
				Path:     image.path,
				MimeType: image.mimeType,
			})
		}
	}
	return out
}

func (s *Session) persistQueue() {
	if s.probe {
		return
	}

	raw, err := json.Marshal(s.persistedEntries())
	if err != nil {
		log.Printf("[session %s] encode pending queue: %v", s.ID, err)
		return
	}
	ctx, cancel := queueContext()
	defer cancel()
	if err := s.store.SetSessionPendingQueue(ctx, sqlitedb.SetSessionPendingQueueParams{
		PendingQueue: string(raw),
		UpdatedAt:    time.Now().UnixMilli(),
		ID:           s.ID,
	}); err != nil {
		log.Printf("[session %s] persist pending queue: %v", s.ID, err)
	}
}

func (s *Session) loadQueue(raw string) {
	if raw == "" || raw == "[]" {
		return
	}
	var entries []persistedEntry
	if err := json.Unmarshal([]byte(raw), &entries); err != nil {
		log.Printf("[session %s] decode pending queue: %v", s.ID, err)
		return
	}
	s.mu.Lock()
	for _, entry := range entries {
		if entry.Text == "" {
			continue
		}
		if entry.ID == "" {
			entry.ID = queueID()
		}
		queued := queueEntry{QueuedPrompt: entry.QueuedPrompt}
		for _, image := range entry.Images {
			queued.images = append(queued.images, imageAttachment{
				mimeType: image.MimeType,
				path:     image.Path,
			})
		}
		s.queue = append(s.queue, queued)
	}
	s.mu.Unlock()
}

func reloadImages(images []imageAttachment) []imageAttachment {
	if len(images) == 0 {
		return nil
	}
	out := make([]imageAttachment, 0, len(images))
	for _, image := range images {
		if image.data == nil && image.path != "" {
			data, err := os.ReadFile(image.path)
			if err != nil {
				log.Printf("queued image %s: %v", image.path, err)
				continue
			}
			image.data = data
		}
		if image.data == nil {
			continue
		}
		out = append(out, image)
	}
	return out
}
