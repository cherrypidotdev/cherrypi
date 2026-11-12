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

package archiver

import (
	"context"
	"log"
	"time"

	"cherrypi.vladfaust.com/agency/internal/settings"
	"cherrypi.vladfaust.com/agency/internal/tasks"
)

const DefaultInterval = time.Hour

type Service struct {
	Tasks    *tasks.Service
	Settings *settings.Service

	Interval time.Duration

	Now func() time.Time
}

func New(tasksSvc *tasks.Service, settingsSvc *settings.Service) *Service {
	return &Service{
		Tasks:    tasksSvc,
		Settings: settingsSvc,
		Interval: DefaultInterval,
		Now:      time.Now,
	}
}

func (s *Service) Run(ctx context.Context) {
	interval := s.Interval
	if interval <= 0 {
		interval = DefaultInterval
	}
	s.Sweep(ctx)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.Sweep(ctx)
		}
	}
}

func (s *Service) Sweep(ctx context.Context) int {
	policy, err := s.Settings.Get(ctx)
	if err != nil {
		log.Printf("[archiver] read the archive policy: %v", err)
		return 0
	}
	orgDays := 0
	if policy.TaskArchive.Enabled {
		orgDays = policy.TaskArchive.Days
	}
	due, err := s.Tasks.AutoArchivable(ctx, orgDays, s.now())
	if err != nil {
		log.Printf("[archiver] list due tasks: %v", err)
		return 0
	}

	archived := 0
	for _, task := range due {

		if _, err := s.Tasks.Archive(ctx, task.ID); err != nil {
			log.Printf("[archiver] archive %s: %v", task.ID, err)
			continue
		}
		archived++

		if err := s.Tasks.CleanupSession(ctx, task.ID); err != nil {
			log.Printf(
				"[archiver] cleanup session for %s deferred: %v", task.ID, err,
			)
		}
	}
	if archived > 0 {
		log.Printf("[archiver] archived %d closed task(s)", archived)
	}
	return archived
}

func (s *Service) now() time.Time {
	if s.Now == nil {
		return time.Now()
	}
	return s.Now()
}
