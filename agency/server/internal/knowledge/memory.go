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

package knowledge

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"cherrypi.vladfaust.com/agency/internal/accounts"
	"cherrypi.vladfaust.com/agency/internal/ids"
	"cherrypi.vladfaust.com/agency/internal/store"
	"github.com/google/uuid"
)

const (
	MemoryLongTerm = "long_term"

	MemoryDaily = "daily"

	MemoryScratchpad = "scratchpad"

	memoryTotalBudget   = 16000
	memoryFileBudget    = 4000
	memoryScratchBudget = 2000
	memoryDailyBudget   = 3000

	// Below it the log is dropped rather than rendered as a one-line stub that

	memoryDailyFloor = 200
)

type ScratchpadInput struct {
	Action string
	Text   string
}

type ForgetResult struct {
	RecoveryID string
	Files      int
	Entries    int
}

func memoryRootPath(projectID, target, date string) string {
	switch target {
	case MemoryDaily:
		if projectID != "" {
			return "memory/projects/" + projectID + "/daily/" + date + ".md"
		}
		return "memory/daily/" + date + ".md"
	case MemoryScratchpad:
		if projectID != "" {
			return "memory/projects/" + projectID + "/SCRATCHPAD.md"
		}
		return "memory/SCRATCHPAD.md"
	default:
		if projectID != "" {
			return "memory/projects/" + projectID + "/MEMORY.md"
		}
		return "memory/MEMORY.md"
	}
}

func (s *Service) memoryRef(actor Actor) ScopeRef {
	return ScopeRef{Scope: ScopeAgent, ScopeID: fmt.Sprint(actor.ID)}
}

func (s *Service) requireAgentMemory(actor Actor) error {
	if actor.Kind != accounts.KindAgent || actor.ID == 0 {
		return ErrAgentMemoryOnly
	}
	return nil
}

func (s *Service) WriteMemory(
	ctx context.Context,
	actor Actor,
	in MemoryWriteInput,
) (MemoryFile, bool, error) {
	if err := s.requireAgentMemory(actor); err != nil {
		return MemoryFile{}, false, err
	}
	target := in.Target
	if target == "" {
		target = MemoryLongTerm
	}
	if target != MemoryLongTerm && target != MemoryDaily && target != MemoryScratchpad {
		return MemoryFile{}, false, fmt.Errorf(
			"%w: unknown memory target %q",
			ErrInvalidInput,
			target,
		)
	}
	date := s.now().UTC().Format("2006-01-02")
	path := strings.TrimSpace(in.Path)
	if path == "" {
		path = memoryRootPath(in.ProjectID, target, date)
	}
	if err := s.validateMemoryPath(path); err != nil {
		return MemoryFile{}, false, err
	}
	if strings.TrimSpace(in.Content) == "" {
		return MemoryFile{}, false, fmt.Errorf("%w: memory content is empty", ErrInvalidInput)
	}

	ref := s.memoryRef(actor)
	concept, err := s.store.GetKnowledgeConcept(ctx, ref.Scope, ref.ScopeID, path)
	now := s.now().UTC()
	entry := in.Content
	mode := target
	if !in.Replace {
		entry = appendEntry(now, actor, in.Content)
	}
	if errors.Is(err, sql.ErrNoRows) {
		created, createErr := s.createMemoryFile(ctx, actor, ref, path, target, entry, now)
		if createErr != nil {
			return MemoryFile{}, false, createErr
		}
		return toMemoryFile(created), target == MemoryLongTerm, nil
	}
	if err != nil {
		return MemoryFile{}, false, err
	}
	if concept.Protected != 0 {
		return MemoryFile{}, false, ErrProtected
	}
	body := entry
	if !in.Replace {
		body = joinMemory(concept.Body, entry)
	}
	write := memoryWrite(concept, actor, now)
	write.Body = body
	write.Title = memoryTitle(path)
	rev := store.KnowledgeRevision{
		ActorID:   nullActor(actor.ID),
		ActorRef:  actor.Ref(),
		Summary:   "memory_write: " + mode,
		CreatedAt: now.UnixMilli(),
	}
	rev.Title, rev.Body, rev.FrontmatterJson, rev.Status = write.Title, write.Body, write.FrontmatterJson, write.Status
	updated, err := s.store.SaveKnowledgeConcept(ctx, concept.ID, concept.Revision, write, rev)
	if err != nil {
		if errors.Is(err, store.ErrKnowledgeConflict) {
			return MemoryFile{}, false, ErrConflict
		}
		return MemoryFile{}, false, err
	}
	return toMemoryFile(updated), target == MemoryLongTerm, nil
}

func (s *Service) createMemoryFile(
	ctx context.Context,
	actor Actor,
	ref ScopeRef,
	path, target, body string,
	now time.Time,
) (store.KnowledgeConcept, error) {
	write := store.KnowledgeConceptWrite{
		Type:               "Memory",
		Title:              memoryTitle(path),
		Body:               body,
		TagsJson:           "[]",
		Status:             StatusStable,
		GeneratedByActorID: nullActor(actor.ID),
		GeneratedBy:        actor.Ref(),
		GeneratedAt:        now.UnixMilli(),
		TrustTier:          TrustUnverified,
	}
	write.FrontmatterJson = memoryFrontmatter(write, ref, path)
	concept := store.KnowledgeConcept{
		ID:               uuid.NewString(),
		Scope:            ref.Scope,
		ScopeID:          ref.ScopeID,
		Path:             path,
		Revision:         1,
		CreatedByActorID: nullActor(actor.ID),
		CreatedAt:        now.UnixMilli(),
		UpdatedAt:        now.UnixMilli(),
	}
	created, err := s.store.CreateKnowledgeConcept(ctx, concept, write)
	if err != nil {
		return store.KnowledgeConcept{}, err
	}
	if _, err := s.store.CreateKnowledgeRevision(ctx, store.KnowledgeRevision{
		ConceptID:       created.ID,
		Revision:        created.Revision,
		Title:           created.Title,
		Body:            created.Body,
		FrontmatterJson: created.FrontmatterJson,
		Status:          created.Status,
		ActorID:         nullActor(actor.ID),
		ActorRef:        actor.Ref(),
		Summary:         "memory_write: " + target,
		CreatedAt:       created.CreatedAt,
	}); err != nil {
		return store.KnowledgeConcept{}, err
	}
	return created, nil
}

func (s *Service) validateMemoryPath(path string) error {
	if err := ValidatePath(path); err != nil {
		return err
	}
	if !strings.HasPrefix(path, MemoryRoot) {
		return fmt.Errorf("%w: memory paths live under %s", ErrInvalidPath, MemoryRoot)
	}
	return nil
}

func appendEntry(now time.Time, actor Actor, content string) string {
	header := fmt.Sprintf("<!-- %s [%s] -->", now.Format("2006-01-02 15:04:05"), actor.Ref())
	return header + "\n" + content
}

func joinMemory(previous, entry string) string {
	previous = strings.TrimRight(previous, "\n")
	if previous == "" {
		return entry
	}
	return previous + "\n\n" + entry
}

func memoryTitle(path string) string {
	switch {
	case strings.HasSuffix(path, "/SCRATCHPAD.md"):
		return "Agent scratchpad"
	case strings.HasSuffix(path, "/MEMORY.md"):
		return "Agent memory"
	case strings.Contains(path, "/daily/"):
		return "Daily log"
	default:
		return path
	}
}

func memoryWrite(c store.KnowledgeConcept, actor Actor, now time.Time) store.KnowledgeConceptWrite {
	write := conceptWriteFrom(c)
	write.GeneratedByActorID = nullActor(actor.ID)
	write.GeneratedBy = actor.Ref()
	write.GeneratedAt = now.UnixMilli()
	return write
}

func memoryFrontmatter(w store.KnowledgeConceptWrite, ref ScopeRef, path string) string {
	return renderFrontmatter(w, ref, path, "")
}

func toMemoryFile(c store.KnowledgeConcept) MemoryFile {
	return MemoryFile{
		Path:      c.Path,
		Body:      c.Body,
		Title:     c.Title,
		UpdatedAt: time.UnixMilli(c.UpdatedAt).UTC(),
		Revision:  c.Revision,
	}
}

func (s *Service) MemoryFiles(
	ctx context.Context,
	actor Actor,
	agentActorID ids.ActorID,
) ([]MemoryFile, error) {
	ref := ScopeRef{Scope: ScopeAgent, ScopeID: fmt.Sprint(agentActorID)}
	if err := s.requireRead(ctx, actor, ref); err != nil {
		return nil, err
	}
	rows, err := s.store.ListKnowledgeConcepts(ctx, store.KnowledgeConceptFilter{
		Scopes:     []string{ScopeAgent},
		ScopeID:    ref.ScopeID,
		PathPrefix: MemoryRoot,
		Limit:      500,
	})
	if err != nil {
		return nil, err
	}
	out := make([]MemoryFile, 0, len(rows))
	for _, row := range rows {
		out = append(out, toMemoryFile(row))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}

func (s *Service) MemoryFileAt(
	ctx context.Context,
	actor Actor,
	agentActorID ids.ActorID,
	path string,
) (MemoryFile, error) {
	ref := ScopeRef{Scope: ScopeAgent, ScopeID: fmt.Sprint(agentActorID)}
	if err := s.requireRead(ctx, actor, ref); err != nil {
		return MemoryFile{}, err
	}
	if !strings.HasPrefix(path, MemoryRoot) {
		return MemoryFile{}, ErrNotFound
	}
	concept, err := s.store.GetKnowledgeConcept(ctx, ref.Scope, ref.ScopeID, path)
	if errors.Is(err, sql.ErrNoRows) {
		return MemoryFile{}, ErrNotFound
	}
	if err != nil {
		return MemoryFile{}, err
	}
	return toMemoryFile(concept), nil
}

func (s *Service) SearchMemory(
	ctx context.Context,
	actor Actor,
	query string,
	limit int,
) ([]store.KnowledgeConcept, error) {
	if err := s.requireAgentMemory(actor); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	return s.store.ListKnowledgeConcepts(ctx, store.KnowledgeConceptFilter{
		Scopes:     []string{ScopeAgent},
		ScopeID:    fmt.Sprint(actor.ID),
		PathPrefix: MemoryRoot,
		Query:      strings.TrimSpace(query),
		Limit:      limit,
	})
}

func (s *Service) Forget(
	ctx context.Context,
	actor Actor,
	query string,
) (ForgetResult, error) {
	if err := s.requireAgentMemory(actor); err != nil {
		return ForgetResult{}, err
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return ForgetResult{}, fmt.Errorf("%w: forget needs a query", ErrInvalidInput)
	}
	rows, err := s.store.ListKnowledgeConcepts(ctx, store.KnowledgeConceptFilter{
		Scopes:     []string{ScopeAgent},
		ScopeID:    fmt.Sprint(actor.ID),
		PathPrefix: MemoryRoot,
		Limit:      500,
	})
	if err != nil {
		return ForgetResult{}, err
	}
	type filePayload struct {
		Path  string `json:"path"`
		Title string `json:"title"`
		Body  string `json:"body"`
	}
	payload := struct {
		Query string        `json:"query"`
		Files []filePayload `json:"files"`
	}{Query: query}
	entries := 0
	now := s.now().UTC()
	for _, row := range rows {
		titleMatch := strings.Contains(strings.ToLower(row.Title), strings.ToLower(query))
		body, removed := removeMatchingLines(row.Body, query)
		if !titleMatch && removed == 0 {
			continue
		}
		payload.Files = append(
			payload.Files,
			filePayload{Path: row.Path, Title: row.Title, Body: row.Body},
		)
		entries += removed
		if removed == 0 {
			continue
		}
		write := memoryWrite(row, actor, now)
		write.Body = body
		rev := store.KnowledgeRevision{
			ActorID:   nullActor(actor.ID),
			ActorRef:  actor.Ref(),
			Summary:   "memory_forget: " + query,
			CreatedAt: now.UnixMilli(),
		}
		rev.Title, rev.Body, rev.FrontmatterJson, rev.Status = write.Title, write.Body, write.FrontmatterJson, write.Status
		if _, err := s.store.SaveKnowledgeConcept(
			ctx,
			row.ID,
			row.Revision,
			write,
			rev,
		); err != nil {
			return ForgetResult{}, err
		}
	}
	if len(payload.Files) == 0 {
		return ForgetResult{}, ErrNotFound
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return ForgetResult{}, err
	}
	id := uuid.NewString()
	if err := s.store.CreateKnowledgeMemoryRecovery(ctx, store.KnowledgeMemoryRecovery{
		ID:           id,
		AgentActorID: int64(actor.ID),
		PayloadJson:  string(raw),
		CreatedAt:    now.UnixMilli(),
	}); err != nil {
		return ForgetResult{}, err
	}
	return ForgetResult{RecoveryID: id, Files: len(payload.Files), Entries: entries}, nil
}

func removeMatchingLines(body, query string) (string, int) {
	lines := strings.Split(body, "\n")
	kept := make([]string, 0, len(lines))
	removed := 0
	for _, line := range lines {
		if strings.Contains(strings.ToLower(line), strings.ToLower(query)) {
			removed++
			continue
		}
		kept = append(kept, line)
	}
	if removed == 0 {
		return body, 0
	}
	return strings.Join(kept, "\n"), removed
}

func (s *Service) RestoreMemory(
	ctx context.Context,
	actor Actor,
	recoveryID string,
) (int, error) {
	if err := s.requireAgentMemory(actor); err != nil {
		return 0, err
	}
	recovery, err := s.store.GetKnowledgeMemoryRecovery(ctx, recoveryID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrMemoryRecoveryNotFound
	}
	if err != nil {
		return 0, err
	}
	if recovery.AgentActorID != int64(actor.ID) {
		return 0, ErrForbidden
	}
	var payload struct {
		Query string `json:"query"`
		Files []struct {
			Path  string `json:"path"`
			Title string `json:"title"`
			Body  string `json:"body"`
		} `json:"files"`
	}
	if err := json.Unmarshal([]byte(recovery.PayloadJson), &payload); err != nil {
		return 0, err
	}
	now := s.now().UTC()
	ref := s.memoryRef(actor)
	restored := 0
	for _, file := range payload.Files {
		concept, err := s.store.GetKnowledgeConcept(ctx, ref.Scope, ref.ScopeID, file.Path)
		if errors.Is(err, sql.ErrNoRows) {
			if _, err := s.createMemoryFile(
				ctx,
				actor,
				ref,
				file.Path,
				MemoryLongTerm,
				file.Body,
				now,
			); err != nil {
				return 0, err
			}
			restored++
			continue
		}
		if err != nil {
			return 0, err
		}
		if concept.Body == file.Body {
			restored++
			continue
		}
		write := memoryWrite(concept, actor, now)
		write.Body = file.Body
		if file.Title != "" {
			write.Title = file.Title
		}
		rev := store.KnowledgeRevision{
			ActorID:   nullActor(actor.ID),
			ActorRef:  actor.Ref(),
			Summary:   "memory_restore: " + recoveryID,
			CreatedAt: now.UnixMilli(),
		}
		rev.Title, rev.Body, rev.FrontmatterJson, rev.Status = write.Title, write.Body, write.FrontmatterJson, write.Status
		if _, err := s.store.SaveKnowledgeConcept(
			ctx,
			concept.ID,
			concept.Revision,
			write,
			rev,
		); err != nil {
			return 0, err
		}
		restored++
	}
	_ = s.store.MarkKnowledgeMemoryRecoveryRestored(ctx, recoveryID, now.UnixMilli())
	return restored, nil
}

func (s *Service) Scratchpad(
	ctx context.Context,
	actor Actor,
	in ScratchpadInput,
) (string, error) {
	if err := s.requireAgentMemory(actor); err != nil {
		return "", err
	}
	path := memoryRootPath("", MemoryScratchpad, "")
	concept, err := s.store.GetKnowledgeConcept(ctx, ScopeAgent, fmt.Sprint(actor.ID), path)
	body := ""
	if err == nil {
		body = concept.Body
	} else if !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}
	action := strings.TrimSpace(in.Action)
	text := strings.TrimSpace(in.Text)
	switch action {
	case "add":
		if text == "" {
			return "", fmt.Errorf("%w: scratchpad add needs text", ErrInvalidInput)
		}
		body = addScratchItem(body, text)
	case "done", "undo":
		if text == "" {
			return "", fmt.Errorf("%w: scratchpad %s needs text", ErrInvalidInput, action)
		}
		body = setScratchItem(body, text, action == "done")
	case "clear_done":
		body = clearDoneScratch(body)
	case "list":
		return body, nil
	default:
		return "", fmt.Errorf("%w: unknown scratchpad action %q", ErrInvalidInput, action)
	}
	now := s.now().UTC()
	if concept.ID == "" {
		if _, err := s.createMemoryFile(
			ctx,
			actor,
			s.memoryRef(actor),
			path,
			MemoryScratchpad,
			body,
			now,
		); err != nil {
			return "", err
		}
		return body, nil
	}
	write := memoryWrite(concept, actor, now)
	write.Body = body
	write.Title = "Agent scratchpad"
	rev := store.KnowledgeRevision{
		ActorID:   nullActor(actor.ID),
		ActorRef:  actor.Ref(),
		Summary:   "scratchpad: " + action,
		CreatedAt: now.UnixMilli(),
	}
	rev.Title, rev.Body, rev.FrontmatterJson, rev.Status = write.Title, write.Body, write.FrontmatterJson, write.Status
	if _, err := s.store.SaveKnowledgeConcept(
		ctx,
		concept.ID,
		concept.Revision,
		write,
		rev,
	); err != nil {
		return "", err
	}
	return body, nil
}

func addScratchItem(body, text string) string {
	item := "- [ ] " + text
	if strings.Contains(body, item) {
		return body
	}
	body = strings.TrimRight(body, "\n")
	if body == "" {
		return item
	}
	return body + "\n" + item
}

// a malformed line (`- [`, `- [x`) must be skipped rather than sliced: indexing

func setScratchItem(body, text string, done bool) string {
	lines := strings.Split(body, "\n")
	want := "- [ ] "
	if done {
		want = "- [x] "
	}
	for i, line := range lines {
		rest, ok := cutScratchMarker(strings.TrimSpace(line))
		if !ok || !strings.Contains(rest, text) {
			continue
		}
		lines[i] = want + rest
		break
	}
	return strings.Join(lines, "\n")
}

// cutScratchMarker splits a `- [ ] item` / `- [x] item` line into its item
// text. A line that only starts like a marker is not an item.
func cutScratchMarker(line string) (string, bool) {
	for _, marker := range []string{"- [ ] ", "- [x] ", "- [X] "} {
		if rest, ok := strings.CutPrefix(line, marker); ok {
			return rest, true
		}
	}
	return "", false
}

func clearDoneScratch(body string) string {
	lines := strings.Split(body, "\n")
	kept := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "- [x]") {
			continue
		}
		kept = append(kept, line)
	}
	return strings.TrimSpace(strings.Join(kept, "\n"))
}

func (s *Service) Context(
	ctx context.Context,
	actor Actor,
	projectID string,
) (MemoryContext, error) {
	if err := s.requireAgentMemory(actor); err != nil {
		return MemoryContext{}, err
	}
	now := s.now().UTC()
	today := now.Format("2006-01-02")
	yesterday := now.AddDate(0, 0, -1).Format("2006-01-02")
	ref := s.memoryRef(actor)
	read := func(path string) string {
		concept, err := s.store.GetKnowledgeConcept(ctx, ref.Scope, ref.ScopeID, path)
		if err != nil {
			return ""
		}
		return concept.Body
	}

	type section struct {
		name   string
		path   string
		text   string
		budget int
		middle bool
	}
	durable := []section{
		{
			name:   "memory",
			path:   memoryRootPath("", MemoryLongTerm, ""),
			text:   read(memoryRootPath("", MemoryLongTerm, "")),
			budget: memoryFileBudget,
			middle: true,
		},
		{
			name:   "scratchpad",
			path:   memoryRootPath("", MemoryScratchpad, ""),
			text:   openScratchItems(read(memoryRootPath("", MemoryScratchpad, ""))),
			budget: memoryScratchBudget,
			middle: true,
		},
	}
	if projectID != "" {
		durable = append(durable, section{
			name:   "project memory",
			path:   memoryRootPath(projectID, MemoryLongTerm, ""),
			text:   read(memoryRootPath(projectID, MemoryLongTerm, "")),
			budget: memoryFileBudget,
			middle: true,
		})
	}
	daily := []section{
		{
			name:   "today's log",
			path:   memoryRootPath("", MemoryDaily, today),
			text:   read(memoryRootPath("", MemoryDaily, today)),
			budget: memoryDailyBudget,
		},
		{
			name:   "yesterday's log",
			path:   memoryRootPath("", MemoryDaily, yesterday),
			text:   read(memoryRootPath("", MemoryDaily, yesterday)),
			budget: memoryDailyBudget,
		},
	}
	if projectID != "" {
		daily = append(daily, section{
			name:   "project today's log",
			path:   memoryRootPath(projectID, MemoryDaily, today),
			text:   read(memoryRootPath(projectID, MemoryDaily, today)),
			budget: memoryDailyBudget,
		})
	}

	var b strings.Builder
	b.WriteString("## CherryPi knowledge\n\n")
	b.WriteString(
		"Recorded knowledge bits are the authoritative shared knowledge; agent memory is a lower-priority personal aid. ",
	)
	b.WriteString(
		"Search with `search_knowledge` or `list_knowledge`, read a bit with `read_knowledge`, and record one with `add_knowledge`; prefer an approved bit over a memory entry when they disagree.\n",
	)

	layers := make([]MemoryLayer, 0, len(durable)+len(daily))

	render := func(sec section, budget int) {
		text := strings.TrimSpace(sec.text)
		if text == "" {
			return
		}
		original := runeLen(text)
		if sec.middle {
			text = truncateMiddle(text, budget)
		} else {
			text = truncateTail(text, budget)
		}
		if strings.TrimSpace(text) == "" {
			return
		}
		b.WriteString("\n### ")
		b.WriteString(sec.name)
		b.WriteString("\n\n")
		b.WriteString(text)
		b.WriteString("\n")
		layers = append(layers, MemoryLayer{
			Name:      sec.name,
			Path:      sec.path,
			Chars:     runeLen(text),
			Truncated: runeLen(text) < original,
		})
	}

	for _, sec := range durable {
		render(sec, sec.budget)
	}
	for _, sec := range daily {
		remaining := memoryTotalBudget - runeLen(b.String()) - sectionOverhead(sec.name)
		if remaining < memoryDailyFloor {
			break
		}
		render(sec, min(sec.budget, remaining))
	}
	rendered := b.String()
	sum := sha256.Sum256([]byte(rendered))
	return MemoryContext{
		Text:       rendered,
		SnapshotAt: now,
		Hash:       hex.EncodeToString(sum[:]),
		Layers:     layers,
		ProjectID:  projectID,
	}, nil
}

func sectionOverhead(name string) int {
	return runeLen("\n### "+name+"\n\n") + 1
}

func (s *Service) Status(
	ctx context.Context,
	actor Actor,
	projectID string,
) (MemoryStatus, error) {
	files, err := s.MemoryFiles(ctx, actor, actor.ID)
	if err != nil {
		return MemoryStatus{}, err
	}
	snapshot, err := s.Context(ctx, actor, projectID)
	if err != nil {
		return MemoryStatus{}, err
	}
	total := 0
	for _, file := range files {
		total += len(file.Body)
	}
	return MemoryStatus{
		ProjectID:  projectID,
		Files:      files,
		TotalChars: total,
		TodayPath:  memoryRootPath("", MemoryDaily, s.now().UTC().Format("2006-01-02")),
		Context:    snapshot,
	}, nil
}

func openScratchItems(body string) string {
	lines := strings.Split(body, "\n")
	kept := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "- [ ]") {
			kept = append(kept, line)
		}
	}
	return strings.Join(kept, "\n")
}

// characters, so a Cyrillic or CJK line must not cost three to four times what

func runeLen(text string) int { return utf8.RuneCountInString(text) }

func truncateTail(text string, limit int) string {
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	if limit <= 1 {
		return "…"
	}
	return "…" + string(runes[len(runes)-(limit-1):])
}

func truncateMiddle(text string, limit int) string {
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	const marker = "\n…\n"
	markerRunes := utf8.RuneCountInString(marker)
	if limit <= markerRunes+2 {
		return truncateTail(text, limit)
	}
	keep := limit - markerRunes
	head := keep / 2
	tail := keep - head
	return string(runes[:head]) + marker + string(runes[len(runes)-tail:])
}
