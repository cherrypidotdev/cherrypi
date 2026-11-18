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
	"strings"
	"time"

	"github.com/sky-valley/pi/ai"

	sqlitedb "cherrypi.vladfaust.com/agent/internal/gen/sqlc"
)

type queuedDelivery struct {
	entry  QueuedPrompt
	images []imageAttachment
}

type queueEntry struct {
	QueuedPrompt

	images []imageAttachment
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

func (s *Session) removeQueuedLocked(id string) bool {
	for i, entry := range s.queue {
		if entry.ID == id {
			s.queue = append(s.queue[:i], s.queue[i+1:]...)
			return true
		}
	}
	return false
}

func (s *Session) forgetQueued(id string) {
	s.mu.Lock()
	removed := s.removeQueuedLocked(id)
	s.mu.Unlock()
	if removed {
		s.publishQueue()
	}
}

func (s *Session) enqueueQueued(
	entry QueuedPrompt,
	images []imageAttachment,
) (QueuedPrompt, bool) {
	if entry.ID == "" {
		entry.ID = queueID()
	}
	s.mu.Lock()
	for i := range s.queue {
		if s.queue[i].ID == entry.ID {
			s.queue[i].images = images
			s.mu.Unlock()
			s.publishQueue()
			return entry, false
		}
	}
	s.queue = append(s.queue, queueEntry{QueuedPrompt: entry, images: images})
	s.mu.Unlock()

	s.publishQueue()
	return entry, true
}

func (s *Session) consumeQueued(text string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return false
	}
	s.mu.Lock()
	index := -1
	for i, entry := range s.queue {
		if entry.Text == text {
			index = i
			break
		}
	}
	if index < 0 {
		s.mu.Unlock()
		return false
	}
	s.queue = append(s.queue[:index], s.queue[index+1:]...)
	s.mu.Unlock()

	s.publishQueue()
	return true
}

func (s *Session) holdQueue() {
	s.mu.Lock()
	eng := s.eng
	empty := len(s.queue) == 0
	s.mu.Unlock()

	if eng != nil {
		eng.sess.Agent.ClearSteeringQueue()
	}
	if empty {
		return
	}
	s.publishQueue()
}

func (s *Session) queuedDeliveriesLocked() []queuedDelivery {
	out := make([]queuedDelivery, 0, len(s.queue))
	for _, entry := range s.queue {
		out = append(out, queuedDelivery{
			entry:  entry.QueuedPrompt,
			images: entry.images,
		})
	}
	return out
}

func (s *Session) takeQueue() []queuedDelivery {
	s.mu.Lock()
	defer s.mu.Unlock()
	taken := s.queuedDeliveriesLocked()
	s.queue = nil
	return taken
}

func (s *Session) restoreHeld(deliveries []queuedDelivery) {
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
	entry, added := s.enqueueQueued(entry, images)
	if err := s.steer(entry.Text, images); err != nil {
		if added {
			s.forgetQueued(entry.ID)
		}
		return err
	}
	return nil
}

func (s *Session) dropQueued(id string) bool {
	s.mu.Lock()
	removed := s.removeQueuedLocked(id)
	s.mu.Unlock()
	if !removed {
		return false
	}
	s.publishQueue()
	s.requeueLive()
	return true
}

func (s *Session) clearQueued() {
	s.mu.Lock()
	empty := len(s.queue) == 0
	eng := s.eng
	s.queue = nil
	s.mu.Unlock()

	if eng != nil {
		eng.sess.Agent.ClearAllQueues()
	}
	if empty {
		return
	}
	s.publishQueue()
}

func (s *Session) requeueLive() {
	s.mu.Lock()
	eng := s.eng
	live := s.queuedDeliveriesLocked()
	s.mu.Unlock()

	if eng == nil || len(live) == 0 {
		return
	}

	eng.sess.Agent.ClearSteeringQueue()
	for _, delivery := range live {
		images := reloadImages(delivery.images)
		if err := s.steer(delivery.entry.Text, images); err != nil {
			log.Printf("[session %s] requeue steer: %v", s.ID, err)
			break
		}
	}
}

func (s *Session) steer(text string, images []imageAttachment) error {
	s.mu.Lock()
	eng := s.eng
	s.mu.Unlock()
	if eng == nil {
		return errors.New("session is not running")
	}
	eng.sess.Steer(ai.UserMessage{
		Content:   userContent(text, images),
		Timestamp: time.Now().UnixMilli(),
	})
	return nil
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
