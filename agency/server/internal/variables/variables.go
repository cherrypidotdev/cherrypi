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

package variables

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"cherrypi.vladfaust.com/agency/internal/auth"
	"cherrypi.vladfaust.com/agency/internal/ids"
	"cherrypi.vladfaust.com/agency/internal/sealed"
	"cherrypi.vladfaust.com/agency/internal/store"
)

const (
	ScopeOrg     = "org"
	ScopeProject = "project"
	ScopeTeam    = "team"
	ScopeGroup   = "group"
	ScopeAgent   = "agent"
)

const MaxValueBytes = 64 << 10

const MaxNoteLength = 500

var (
	ErrInvalidName = errors.New(
		"a variable name must start with a letter or underscore and contain only " +
			"letters, digits and underscores (max 128 characters)",
	)

	ErrReservedName = errors.New("that name is reserved by the runtime")

	ErrInvalidValue = errors.New("a value must be 1 byte to 64 KiB and contain no NUL")

	ErrNoteTooLong = errors.New("a note must be at most 500 characters")

	ErrNotFound = errors.New("no such variable")

	ErrExists = errors.New("a variable with that name already exists in this scope")

	ErrAlreadySecret = errors.New("that variable is already a secret")

	ErrInvalidScope = errors.New("invalid variable scope")

	ErrForbidden = errors.New("only the scope's owners may manage its variables")
)

// namePattern is a portable environment variable name: no dots, dashes or

var namePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]{0,127}$`)

var reservedNames = map[string]bool{
	"PATH":                  true,
	"HOME":                  true,
	"USER":                  true,
	"LOGNAME":               true,
	"SHELL":                 true,
	"ENV":                   true,
	"BASH_ENV":              true,
	"IFS":                   true,
	"PWD":                   true,
	"OLDPWD":                true,
	"TMPDIR":                true,
	"LD_PRELOAD":            true,
	"LD_LIBRARY_PATH":       true,
	"DYLD_INSERT_LIBRARIES": true,
	"NODE_OPTIONS":          true,
	"GIT_ASKPASS":           true,
	"SSH_AUTH_SOCK":         true,
	"SSH_ASKPASS":           true,
}

var reservedPrefixes = []string{"CHERRYPI_", "PI_", "LD_", "DYLD_"}

type Scope struct {
	Kind string
	ID   string
}

func OrgScope() Scope { return Scope{Kind: ScopeOrg} }

func ProjectScope(projectID string) Scope {
	return Scope{Kind: ScopeProject, ID: projectID}
}

func TeamScope(teamID int64) Scope {
	return Scope{Kind: ScopeTeam, ID: strconv.FormatInt(teamID, 10)}
}

func GroupScope(groupID int64) Scope {
	return Scope{Kind: ScopeGroup, ID: strconv.FormatInt(groupID, 10)}
}

func AgentScope(actorID ids.ActorID) Scope {
	return Scope{Kind: ScopeAgent, ID: strconv.FormatInt(int64(actorID), 10)}
}

func (s Scope) Validate() error {
	switch s.Kind {
	case ScopeOrg:
		if s.ID != "" {
			return fmt.Errorf("%w: the organization scope has no id", ErrInvalidScope)
		}
		return nil
	case ScopeProject:
		if strings.TrimSpace(s.ID) == "" {
			return fmt.Errorf("%w: a project id is required", ErrInvalidScope)
		}
		return nil
	case ScopeTeam, ScopeGroup, ScopeAgent:
		if _, err := strconv.ParseInt(s.ID, 10, 64); err != nil {
			return fmt.Errorf("%w: %s needs a numeric id", ErrInvalidScope, s.Kind)
		}
		return nil
	default:
		return fmt.Errorf("%w: unknown kind %q", ErrInvalidScope, s.Kind)
	}
}

func (s Scope) String() string {
	if s.ID == "" {
		return s.Kind
	}
	return s.Kind + ":" + s.ID
}

type Variable struct {
	Scope     Scope
	Name      string
	Note      string
	Secret    bool
	Value     string
	CreatedAt time.Time
	UpdatedAt time.Time
	UpdatedBy ids.ActorID
	sealed    []byte
}

type Resolved struct {
	Name     string
	Note     string
	Secret   bool
	Value    string
	Source   Scope
	Shadowed []Scope
}

type Caller struct {
	AgentActorID int64

	ProjectID string
}

type ResolverStore interface {
	ListVariables(ctx context.Context) ([]store.Variable, error)
	ListTeamIDsForAgentActor(ctx context.Context, actorID int64) ([]int64, error)
	ListAgentGroupIDsForAgentActor(ctx context.Context, actorID int64) ([]int64, error)
}

type WriteStore interface {
	GetVariable(ctx context.Context, p store.GetVariableParams) (store.Variable, error)
	SaveVariable(ctx context.Context, p store.UpsertVariableParams, revision string) error
	DeleteVariable(
		ctx context.Context,
		p store.DeleteVariableParams,
		revision string,
	) (bool, error)
	DeleteVariablesForScope(ctx context.Context, p store.DeleteVariablesForScopeParams) error
	BumpAgentConfigRevision(ctx context.Context, revision string) error
}

type LogStore interface {
	InsertProjectLog(ctx context.Context, p store.InsertProjectLogParams) error
	InsertTeamLog(ctx context.Context, p store.InsertTeamLogParams) error
	InsertAgentGroupLog(ctx context.Context, p store.InsertAgentGroupLogParams) error
	InsertAgentDomainLog(ctx context.Context, p store.InsertAgentDomainLogParams) error
	InsertUserLog(ctx context.Context, p store.InsertUserLogParams) error
}

type Service struct {
	store  ResolverStore
	writes WriteStore
	logs   LogStore
	sealer *sealed.Sealer
	now    func() time.Time
}

func New(
	st ResolverStore,
	writes WriteStore,
	logs LogStore,
	secretKey string,
) (*Service, error) {
	seal, err := sealed.New(secretKey)
	if err != nil {
		return nil, err
	}
	return &Service{store: st, writes: writes, logs: logs, sealer: seal, now: time.Now}, nil
}

func (s *Service) SecretsEnabled() bool { return s.sealer.Enabled() }

func ValidateName(name string) error {
	if !namePattern.MatchString(name) {
		return ErrInvalidName
	}
	if reservedNames[name] {
		return fmt.Errorf("%w: %s", ErrReservedName, name)
	}
	for _, prefix := range reservedPrefixes {
		if strings.HasPrefix(name, prefix) {
			return fmt.Errorf(
				"%w: the %s namespace belongs to the runtime",
				ErrReservedName,
				prefix,
			)
		}
	}
	return nil
}

func ValidateValue(value string) error {
	if len(value) == 0 || len(value) > MaxValueBytes || strings.ContainsRune(value, 0) {
		return ErrInvalidValue
	}
	return nil
}

func ValidateNote(note string) error {
	if len([]rune(note)) > MaxNoteLength {
		return ErrNoteTooLong
	}
	return nil
}

func (s *Service) List(ctx context.Context, scope Scope) ([]Variable, error) {
	if err := scope.Validate(); err != nil {
		return nil, err
	}
	all, err := s.store.ListVariables(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Variable, 0, len(all))
	for _, row := range all {
		v := variableOf(row)
		if v.Scope != scope {
			continue
		}
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (s *Service) Get(ctx context.Context, scope Scope, name string) (Variable, error) {
	if err := scope.Validate(); err != nil {
		return Variable{}, err
	}
	row, err := s.writes.GetVariable(ctx, store.GetVariableParams{
		ScopeKind: scope.Kind,
		ScopeID:   scope.ID,
		Name:      name,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return Variable{}, ErrNotFound
	}
	if err != nil {
		return Variable{}, err
	}
	return variableOf(row), nil
}

func (s *Service) Create(
	ctx context.Context,
	scope Scope,
	name, value, note string,
	secret bool,
	actorID ids.ActorID,
) (Variable, error) {
	if err := scope.Validate(); err != nil {
		return Variable{}, err
	}
	name = strings.TrimSpace(name)
	if err := ValidateName(name); err != nil {
		return Variable{}, err
	}
	if err := ValidateValue(value); err != nil {
		return Variable{}, err
	}
	if err := ValidateNote(note); err != nil {
		return Variable{}, err
	}
	if _, err := s.Get(ctx, scope, name); err == nil {
		return Variable{}, ErrExists
	} else if !errors.Is(err, ErrNotFound) {
		return Variable{}, err
	}

	params, err := s.valueParams(scope, name, value, note, secret, actorID)
	if err != nil {
		return Variable{}, err
	}
	now := s.now()
	params.CreatedAt = now.UnixMilli()
	params.UpdatedAt = now.UnixMilli()
	params.CreatedByActorID = nullActor(actorID)
	if err := s.writes.SaveVariable(ctx, params, s.nextRevision()); err != nil {
		return Variable{}, err
	}
	s.log(ctx, scope, "created", name, "", secretLabel(secret),
		fmt.Sprintf("Created %s %s", kindLabel(secret), name), nil)
	return s.Get(ctx, scope, name)
}

func (s *Service) Update(
	ctx context.Context,
	scope Scope,
	name string,
	note *string,
	value *string,
	actorID ids.ActorID,
) (Variable, error) {
	current, err := s.Get(ctx, scope, name)
	if err != nil {
		return Variable{}, err
	}
	if note != nil {
		if err := ValidateNote(*note); err != nil {
			return Variable{}, err
		}
		current.Note = *note
	}
	plain := current.Value
	sealedValue := []byte(nil)
	if current.Secret {

		if value == nil {
			sealedValue, err = s.storedSealed(ctx, scope, name)
			if err != nil {
				return Variable{}, err
			}
		} else {
			if err := ValidateValue(*value); err != nil {
				return Variable{}, err
			}
			sealedValue, err = s.sealer.Seal(*value)
			if err != nil {
				return Variable{}, err
			}
		}
	} else {
		if value != nil {
			if err := ValidateValue(*value); err != nil {
				return Variable{}, err
			}
			plain = *value
		}
	}

	params := store.UpsertVariableParams{
		ScopeKind:        scope.Kind,
		ScopeID:          scope.ID,
		Name:             name,
		Note:             current.Note,
		Secret:           boolInt(current.Secret),
		ValueSealed:      sealedValue,
		UpdatedByActorID: nullActor(actorID),
		UpdatedAt:        s.now().UnixMilli(),
	}
	if current.Secret {
		params.ValueSealed = sealedValue
	} else {
		params.Value = sql.NullString{String: plain, Valid: true}
	}
	if err := s.writes.SaveVariable(ctx, params, s.nextRevision()); err != nil {
		return Variable{}, err
	}
	field, message := "note", "Updated the note of "+name
	if value != nil {
		field, message = "value", "Replaced the value of "+name
	}
	s.log(ctx, scope, "updated", name, "", field, message, nil)
	return s.Get(ctx, scope, name)
}

func (s *Service) Promote(
	ctx context.Context,
	scope Scope,
	name string,
	actorID ids.ActorID,
) (Variable, error) {
	current, err := s.Get(ctx, scope, name)
	if err != nil {
		return Variable{}, err
	}
	if current.Secret {
		return Variable{}, ErrAlreadySecret
	}
	sealedValue, err := s.sealer.Seal(current.Value)
	if err != nil {
		return Variable{}, err
	}
	params := store.UpsertVariableParams{
		ScopeKind:        scope.Kind,
		ScopeID:          scope.ID,
		Name:             name,
		Note:             current.Note,
		Secret:           1,
		ValueSealed:      sealedValue,
		UpdatedByActorID: nullActor(actorID),
		UpdatedAt:        s.now().UnixMilli(),
	}
	if err := s.writes.SaveVariable(ctx, params, s.nextRevision()); err != nil {
		return Variable{}, err
	}
	s.log(ctx, scope, "promoted", name, "plain", "secret",
		"Made "+name+" a secret", nil)
	return s.Get(ctx, scope, name)
}

func (s *Service) Delete(ctx context.Context, scope Scope, name string) error {
	if err := scope.Validate(); err != nil {
		return err
	}
	deleted, err := s.writes.DeleteVariable(ctx, store.DeleteVariableParams{
		ScopeKind: scope.Kind,
		ScopeID:   scope.ID,
		Name:      name,
	}, s.nextRevision())
	if err != nil {
		return err
	}
	if !deleted {
		return ErrNotFound
	}
	s.log(ctx, scope, "deleted", name, name, "", "Deleted "+name, nil)
	return nil
}

func (s *Service) DeleteScope(ctx context.Context, scope Scope) error {
	if err := scope.Validate(); err != nil {
		return err
	}
	if err := s.writes.DeleteVariablesForScope(ctx, store.DeleteVariablesForScopeParams{
		ScopeKind: scope.Kind,
		ScopeID:   scope.ID,
	}); err != nil {
		return err
	}
	return s.writes.BumpAgentConfigRevision(ctx, s.nextRevision())
}

func (s *Service) valueParams(
	scope Scope,
	name, value, note string,
	secret bool,
	actorID ids.ActorID,
) (store.UpsertVariableParams, error) {
	params := store.UpsertVariableParams{
		ScopeKind:        scope.Kind,
		ScopeID:          scope.ID,
		Name:             name,
		Note:             note,
		Secret:           boolInt(secret),
		UpdatedByActorID: nullActor(actorID),
	}
	if !secret {
		params.Value = sql.NullString{String: value, Valid: true}
		return params, nil
	}
	sealedValue, err := s.sealer.Seal(value)
	if err != nil {
		return store.UpsertVariableParams{}, err
	}
	params.ValueSealed = sealedValue
	return params, nil
}

func (s *Service) storedSealed(ctx context.Context, scope Scope, name string) ([]byte, error) {
	row, err := s.writes.GetVariable(ctx, store.GetVariableParams{
		ScopeKind: scope.Kind,
		ScopeID:   scope.ID,
		Name:      name,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return row.ValueSealed, nil
}

func (s *Service) nextRevision() string {
	return store.NewRevision()
}

func (s *Service) log(
	ctx context.Context,
	scope Scope,
	kind, name, oldValue, newValue, message string,
	data any,
) {
	if s.logs == nil {
		return
	}
	raw := ""
	if data != nil {
		if encoded, err := json.Marshal(data); err == nil {
			raw = string(encoded)
		}
	}
	at := s.now().UnixMilli()
	actor := nullActor(auditActorID(ctx))
	var err error
	switch scope.Kind {
	case ScopeProject:
		err = s.logs.InsertProjectLog(ctx, store.InsertProjectLogParams{
			ProjectID: scope.ID, At: at, ActorID: actor, Kind: kind,
			Field: name, OldValue: oldValue, NewValue: newValue,
			Message: message, DataJson: raw,
		})
	case ScopeTeam:
		teamID, _ := strconv.ParseInt(scope.ID, 10, 64)
		err = s.logs.InsertTeamLog(ctx, store.InsertTeamLogParams{
			TeamID: teamID, At: at, ActorID: actor, Kind: kind,
			Field: name, OldValue: oldValue, NewValue: newValue,
			Message: message, DataJson: raw,
		})
	case ScopeGroup:
		groupID, _ := strconv.ParseInt(scope.ID, 10, 64)
		err = s.logs.InsertAgentGroupLog(ctx, store.InsertAgentGroupLogParams{
			GroupID: groupID, At: at, ActorID: actor, Kind: kind,
			Field: name, OldValue: oldValue, NewValue: newValue,
			Message: message, DataJson: raw,
		})
	case ScopeAgent:
		agentActorID, _ := strconv.ParseInt(scope.ID, 10, 64)
		err = s.logs.InsertAgentDomainLog(ctx, store.InsertAgentDomainLogParams{
			AgentActorID: agentActorID, At: at, ActorID: actor, Kind: kind,
			Field: name, OldValue: oldValue, NewValue: newValue,
			Message: message, DataJson: raw,
		})
	case ScopeOrg:

		if actor.Valid {
			err = s.logs.InsertUserLog(ctx, store.InsertUserLogParams{
				UserActorID: actor.Int64, At: at, ActorID: actor, Kind: kind,
				Field: name, OldValue: oldValue, NewValue: newValue,
				Message: message, DataJson: raw,
			})
		}
	}
	if err != nil {
		log.Printf("failed to write %s variable log for %s: %v", scope.Kind, scope, err)
	}
}

func variableOf(row store.Variable) Variable {
	out := Variable{
		Scope:     Scope{Kind: row.ScopeKind, ID: row.ScopeID},
		Name:      row.Name,
		Note:      row.Note,
		Secret:    row.Secret != 0,
		CreatedAt: time.UnixMilli(row.CreatedAt),
		UpdatedAt: time.UnixMilli(row.UpdatedAt),
	}
	if row.Value.Valid {
		out.Value = row.Value.String
	}
	if row.Secret != 0 {
		out.sealed = row.ValueSealed
	}
	if row.UpdatedByActorID.Valid {
		out.UpdatedBy = ids.ActorID(row.UpdatedByActorID.Int64)
	}
	return out
}

func boolInt(value bool) int64 {
	if value {
		return 1
	}
	return 0
}

func nullActor(id ids.ActorID) sql.NullInt64 {
	if id == 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(id), Valid: true}
}

func auditActorID(ctx context.Context) ids.ActorID {
	if id, ok := auth.ActorID(ctx); ok {
		return id
	}
	return 0
}

func kindLabel(secret bool) string {
	if secret {
		return "secret"
	}
	return "variable"
}

func secretLabel(secret bool) string {
	if secret {
		return "secret"
	}
	return "plain"
}
