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

package tasks

import (
	"cherrypi.vladfaust.com/agency/internal/ids"
	"cherrypi.vladfaust.com/agency/internal/store"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"cherrypi.vladfaust.com/agency/internal/actorid"
)

type Actor string

const (
	ActorUser Actor = "user"

	ActorAgent Actor = "agent"

	ActorSystem Actor = "system"
)

const (
	LogCreated    = "created"
	LogState      = "state"
	LogName       = "name"
	LogPrompt     = "prompt"
	LogPriority   = "priority"
	LogSize       = "size"
	LogAssignee   = "assignee"
	LogTags       = "tags"
	LogReferences = "references"
	LogLinks      = "links"
	LogProject    = "project"
	LogModel      = "model"
	LogSession    = "session"

	LogTeams = "teams"

	LogLeads = "leads"
)

type actorContextKey struct{}

func WithActor(ctx context.Context, actor Actor) context.Context {
	return context.WithValue(ctx, actorContextKey{}, actor)
}

func actorFrom(ctx context.Context) Actor {
	if actor, ok := ctx.Value(actorContextKey{}).(Actor); ok && actor != "" {
		return actor
	}
	return ActorSystem
}

func ActorFrom(ctx context.Context) Actor { return actorFrom(ctx) }

type actorIDContextKey struct{}

func WithActorID(ctx context.Context, actorID ids.ActorID) context.Context {
	return context.WithValue(ctx, actorIDContextKey{}, actorID)
}

func actorIDFrom(ctx context.Context) ids.ActorID {
	if id, ok := ctx.Value(actorIDContextKey{}).(ids.ActorID); ok {
		return id
	}
	return 0
}

type suppressLogKey struct{}

func withoutLogging(ctx context.Context) context.Context {
	return context.WithValue(ctx, suppressLogKey{}, true)
}

func loggingSuppressed(ctx context.Context) bool {
	suppressed, _ := ctx.Value(suppressLogKey{}).(bool)
	return suppressed
}

type Log struct {
	ID       int64
	At       time.Time
	Actor    string
	ActorID  ids.ActorID
	Kind     string
	Field    string
	OldValue string
	NewValue string
	Message  string

	DataJSON string
}

func (s *Service) Logs(ctx context.Context, taskID string) ([]Log, error) {
	rows, err := s.store.ListTaskLogsByTask(ctx, taskID)
	if err != nil {
		return nil, err
	}
	out := make([]Log, len(rows))
	for i, row := range rows {
		out[i] = Log{
			ID:       row.ID,
			At:       time.UnixMilli(row.At),
			Actor:    row.Actor,
			ActorID:  ids.ActorID(row.ActorID.Int64),
			Kind:     row.Kind,
			Field:    row.Field,
			OldValue: row.OldValue,
			NewValue: row.NewValue,
			Message:  row.Message,
			DataJSON: row.DataJson,
		}
	}
	return out, nil
}

func (s *Service) recordLog(
	ctx context.Context,
	actor Actor,
	at int64,
	taskID, kind, field, oldValue, newValue, message string,
	data any,
) {
	if taskID == "" || loggingSuppressed(ctx) {
		return
	}
	if err := s.store.InsertTaskLog(ctx, store.InsertTaskLogParams{
		TaskID:   taskID,
		At:       at,
		Actor:    string(actor),
		ActorID:  nullID(actorIDFrom(ctx)),
		Kind:     kind,
		Field:    field,
		OldValue: oldValue,
		NewValue: newValue,
		Message:  message,
		DataJson: marshalData(data),
	}); err != nil {
		log.Printf("[tasks] failed to record %s log for %s: %v", kind, taskID, err)
		return
	}
	s.changed(ctx, taskID)
}

func marshalData(data any) string {
	if data == nil {
		return ""
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return ""
	}
	return string(raw)
}

func (s *Service) logCreated(ctx context.Context, t Task) {
	s.recordLog(
		ctx, actorFrom(ctx), time.Now().UnixMilli(),
		t.ID, LogCreated, "", "", stateLabel(t.State), "Task created", nil,
	)
}

func (s *Service) logOwnership(ctx context.Context, before, after Task) {
	actor := actorFrom(ctx)
	at := time.Now().UnixMilli()
	if !equalInt64(before.TeamIDs, after.TeamIDs) {
		s.recordLog(
			ctx, actor, at,
			after.ID, LogTeams, "teams",
			labelWithCount("teams", len(before.TeamIDs)),
			labelWithCount("teams", len(after.TeamIDs)),
			"Owning teams changed to "+labelWithCount("teams", len(after.TeamIDs)),
			map[string]any{"team_ids": after.TeamIDs},
		)
	}
	if !equalActorIDs(before.LeadActorIDs, after.LeadActorIDs) {
		s.recordLog(
			ctx, actor, at,
			after.ID, LogLeads, "leads",
			labelWithCount("leads", len(before.LeadActorIDs)),
			labelWithCount("leads", len(after.LeadActorIDs)),
			"Leads changed to "+labelWithCount("leads", len(after.LeadActorIDs)),
			map[string]any{"actor_ids": after.LeadActorIDs},
		)
	}
}

func labelWithCount(noun string, count int) string {
	if count == 0 {
		return "no " + noun
	}
	return noun + " (" + strconv.Itoa(count) + ")"
}

func equalActorIDs(a, b []ids.ActorID) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalInt64(a, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (s *Service) logChanges(ctx context.Context, before, after Task) {
	actor := actorFrom(ctx)
	at := time.Now().UnixMilli()

	id := after.ID
	if id == "" {
		id = before.ID
	}

	if before.State != after.State {
		s.recordLog(
			ctx, actor, at, id, LogState, "state",
			stateLabel(before.State), stateLabel(after.State),
			fmt.Sprintf(
				"State changed from %s to %s",
				stateLabel(before.State), stateLabel(after.State),
			),
			enumChange(string(before.State), string(after.State)),
		)
	}
	if before.Name != after.Name {
		s.recordLog(
			ctx, actor, at, id, LogName, "name",
			displayName(before.Name), displayName(after.Name),
			fmt.Sprintf("Title changed to %q", displayName(after.Name)),
			valueChange(before.Name, after.Name),
		)
	}
	if before.Prompt != after.Prompt {
		s.recordLog(
			ctx, actor, at, id, LogPrompt, "prompt", "", "",
			"Prompt edited",
			nil,
		)
	}
	if before.ProjectID != after.ProjectID {
		s.recordLog(
			ctx, actor, at, id, LogProject, "project",
			s.projectLabel(ctx, before.ProjectID), s.projectLabel(ctx, after.ProjectID),
			fmt.Sprintf(
				"Project changed from %s to %s",
				s.projectLabel(ctx, before.ProjectID), s.projectLabel(ctx, after.ProjectID),
			),
			referenceChange(
				before.ProjectID, s.projectLabel(ctx, before.ProjectID),
				after.ProjectID, s.projectLabel(ctx, after.ProjectID),
			),
		)
	}
	if before.AssigneeActorID != after.AssigneeActorID {
		s.recordLog(
			ctx, actor, at, id, LogAssignee, "assignee",
			s.agentLabel(ctx, before.AssigneeActorID), s.agentLabel(ctx, after.AssigneeActorID),
			fmt.Sprintf(
				"Assignee changed from %s to %s",
				s.agentLabel(ctx, before.AssigneeActorID), s.agentLabel(ctx, after.AssigneeActorID),
			),
			referenceChange(
				actorid.Format(before.AssigneeActorID), s.agentLabel(ctx, before.AssigneeActorID),
				actorid.Format(after.AssigneeActorID), s.agentLabel(ctx, after.AssigneeActorID),
			),
		)
	}
	if before.Priority != after.Priority {
		s.recordLog(
			ctx, actor, at, id, LogPriority, "priority",
			priorityLabel(before.Priority), priorityLabel(after.Priority),
			fmt.Sprintf(
				"Priority changed from %s to %s",
				priorityLabel(before.Priority), priorityLabel(after.Priority),
			),
			numberChange(before.Priority, after.Priority),
		)
	}
	if before.Size != after.Size {
		s.recordLog(
			ctx, actor, at, id, LogSize, "size",
			sizeLabel(before.Size), sizeLabel(after.Size),
			fmt.Sprintf(
				"Size changed from %s to %s",
				sizeLabel(before.Size), sizeLabel(after.Size),
			),
			enumChange(before.Size, after.Size),
		)
	}
	if before.ModelProvider != after.ModelProvider || before.ModelID != after.ModelID {
		s.recordLog(
			ctx, actor, at, id, LogModel, "model",
			modelLabel(before), modelLabel(after),
			fmt.Sprintf("Model changed to %s", modelLabel(after)),
			valueChange(
				modelLabel(before),
				modelLabel(after),
			),
		)
	}
	if before.ThinkingLevel != after.ThinkingLevel {
		s.recordLog(
			ctx, actor, at, id, LogModel, "thinking",
			before.ThinkingLevel, after.ThinkingLevel,
			fmt.Sprintf(
				"Thinking level changed from %s to %s",
				displayName(before.ThinkingLevel), displayName(after.ThinkingLevel),
			),
			valueChange(before.ThinkingLevel, after.ThinkingLevel),
		)
	}
	if added, removed := diffTags(before.Tags, after.Tags); len(added)+len(removed) > 0 {
		s.recordLog(
			ctx, actor, at, id, LogTags, "tags", "", "",
			tagMessage(added, removed),
			map[string]any{"added": added, "removed": removed},
		)
	}
	if added, removed, changed := diffReferences(
		before.References,
		after.References,
	); len(
		added,
	)+len(
		removed,
	)+len(
		changed,
	) > 0 {
		s.recordLog(
			ctx, actor, at, id, LogReferences, "references", "", "",
			referenceMessage(added, removed, changed),
			map[string]any{"added": added, "removed": removed, "changed": changed},
		)
	}
	if added, removed := diffLinks(before.Links, after.Links); len(added)+len(removed) > 0 {
		s.recordLog(
			ctx, actor, at, id, LogLinks, "links", "", "",
			linkMessage(added, removed),
			map[string]any{"added": added, "removed": removed},
		)
	}
}

func enumChange(oldValue, newValue string) map[string]any {
	return map[string]any{"old": oldValue, "new": newValue}
}

func numberChange(oldValue, newValue int64) map[string]any {
	return map[string]any{"old": oldValue, "new": newValue}
}

func valueChange(oldValue, newValue string) map[string]any {
	return map[string]any{"old": oldValue, "new": newValue}
}

func referenceChange(oldID, oldName, newID, newName string) map[string]any {
	return map[string]any{
		"oldId":   oldID,
		"oldName": oldName,
		"newId":   newID,
		"newName": newName,
	}
}

type logTag struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

type logReference struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	State string `json:"state"`
	Kind  string `json:"kind"`
}

type logLink struct {
	URL   string `json:"url"`
	Label string `json:"label"`
}

func (s *Service) agentLabel(ctx context.Context, id ids.ActorID) string {
	if id == 0 {
		return "Unassigned"
	}
	row, err := s.store.GetAgentByActor(ctx, id)
	if err != nil {
		return actorid.Format(id)
	}
	if row.Name != "" {
		return row.Name
	}
	return row.Username
}

func (s *Service) projectLabel(ctx context.Context, id string) string {
	if id == "" {
		return "No project"
	}
	project, err := s.projects.Get(ctx, id)
	if err != nil {
		return id
	}
	return project.Name
}

func stateLabel(state State) string {
	switch state {
	case StateDraft:
		return "Draft"
	case StateBacklog:
		return "Backlog"
	case StateQueued:
		return "Queued"
	case StateRunning:
		return "Running"
	case StateInput:
		return "Input"
	case StateSettled:
		return "Settled"
	case StateFailed:
		return "Failed"
	case StateCompleted:
		return "Completed"
	case StateCancelled:
		return "Cancelled"
	default:
		return string(state)
	}
}

func priorityLabel(priority int64) string {
	switch priority {
	case PriorityLow:
		return "Low"
	case PriorityNormal:
		return "Normal"
	case PriorityHigh:
		return "High"
	case PriorityCritical:
		return "Critical"
	default:
		return "Normal"
	}
}

func sizeLabel(size string) string {
	switch size {
	case SizeSmall:
		return "Small"
	case SizeMedium:
		return "Medium"
	case SizeBig:
		return "Big"
	case SizeEpic:
		return "Epic"
	default:
		return displayName(size)
	}
}

func modelLabel(t Task) string {
	if t.ModelProvider == "" && t.ModelID == "" {
		return "Pi default"
	}
	if t.ModelProvider == "" {
		return t.ModelID
	}
	return t.ModelProvider + "/" + t.ModelID
}

func displayName(value string) string {
	if strings.TrimSpace(value) == "" {
		return "—"
	}
	return value
}

func diffTags(before, after []Tag) (added, removed []logTag) {
	beforeIDs := make(map[string]bool, len(before))
	for _, tag := range before {
		beforeIDs[tag.ID] = true
	}
	afterIDs := make(map[string]bool, len(after))
	for _, tag := range after {
		afterIDs[tag.ID] = true
	}
	for _, tag := range after {
		if !beforeIDs[tag.ID] {
			added = append(added, logTag{ID: tag.ID, Name: tag.Name, Color: tag.Color})
		}
	}
	for _, tag := range before {
		if !afterIDs[tag.ID] {
			removed = append(removed, logTag{ID: tag.ID, Name: tag.Name, Color: tag.Color})
		}
	}
	sort.Slice(added, func(i, j int) bool { return added[i].Name < added[j].Name })
	sort.Slice(removed, func(i, j int) bool { return removed[i].Name < removed[j].Name })
	return added, removed
}

func diffReferences(before, after []Reference) (added, removed, changed []logReference) {
	beforeByID := make(map[string]Reference, len(before))
	for _, ref := range before {
		beforeByID[ref.ID] = ref
	}
	afterByID := make(map[string]Reference, len(after))
	for _, ref := range after {
		afterByID[ref.ID] = ref
	}
	log := func(ref Reference) logReference {
		return logReference{
			ID: ref.ID, Name: ref.Name, State: string(ref.State), Kind: string(ref.Kind),
		}
	}
	for _, ref := range after {
		prior, ok := beforeByID[ref.ID]
		switch {
		case !ok:
			added = append(added, log(ref))
		case prior.Kind != ref.Kind:
			changed = append(changed, log(ref))
		}
	}
	for _, ref := range before {
		if _, ok := afterByID[ref.ID]; !ok {
			removed = append(removed, log(ref))
		}
	}
	sortByID := func(list []logReference) {
		sort.Slice(list, func(i, j int) bool { return list[i].ID < list[j].ID })
	}
	sortByID(added)
	sortByID(removed)
	sortByID(changed)
	return added, removed, changed
}

func diffLinks(before, after []Link) (added, removed []logLink) {
	beforeURLs := make(map[string]bool, len(before))
	for _, link := range before {
		beforeURLs[link.URL] = true
	}
	afterURLs := make(map[string]bool, len(after))
	for _, link := range after {
		afterURLs[link.URL] = true
	}
	for _, link := range after {
		if !beforeURLs[link.URL] {
			added = append(added, logLink(link))
		}
	}
	for _, link := range before {
		if !afterURLs[link.URL] {
			removed = append(removed, logLink(link))
		}
	}
	sort.Slice(added, func(i, j int) bool { return added[i].URL < added[j].URL })
	sort.Slice(removed, func(i, j int) bool { return removed[i].URL < removed[j].URL })
	return added, removed
}

func tagNames(tags []logTag) []string {
	names := make([]string, len(tags))
	for i, tag := range tags {
		names[i] = tag.Name
	}
	return names
}

func referenceNames(refs []logReference) []string {
	names := make([]string, len(refs))
	for i, ref := range refs {
		names[i] = referenceLabel(ref.ID, ref.Name, ref.Kind)
	}
	return names
}

func referenceLabel(id, name, kind string) string {
	label := id
	if name != "" {
		label = id + " " + name
	}
	switch kind {
	case "blocking":
		label += " (blocking)"
	case "supersedes":
		label += " (supersedes)"
	}
	return label
}

func linkNames(links []logLink) []string {
	names := make([]string, len(links))
	for i, link := range links {
		if link.Label != "" {
			names[i] = link.Label
		} else {
			names[i] = link.URL
		}
	}
	return names
}

func tagMessage(added, removed []logTag) string {
	return changeMessage(
		joinNamed("Tags added: ", tagNames(added)),
		joinNamed("Tags removed: ", tagNames(removed)),
	)
}

func referenceMessage(added, removed, changed []logReference) string {
	addRemove := changeMessage(
		joinNamed("References added: ", referenceNames(added)),
		joinNamed("References removed: ", referenceNames(removed)),
	)
	return changeMessage(
		addRemove,
		joinNamed("Blocking changed: ", referenceNames(changed)),
	)
}

func linkMessage(added, removed []logLink) string {
	return changeMessage(
		joinNamed("Links added: ", linkNames(added)),
		joinNamed("Links removed: ", linkNames(removed)),
	)
}

func joinNamed(prefix string, names []string) string {
	if len(names) == 0 {
		return ""
	}
	return prefix + strings.Join(names, ", ")
}

func changeMessage(added, removed string) string {
	parts := make([]string, 0, 2)
	if added != "" {
		parts = append(parts, added)
	}
	if removed != "" {
		parts = append(parts, removed)
	}
	return strings.Join(parts, "; ")
}
