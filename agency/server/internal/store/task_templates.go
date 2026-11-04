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

package store

import (
	"context"
	"database/sql"

	pgdb "cherrypi.vladfaust.com/agency/internal/gen/pg"
	sqlitedb "cherrypi.vladfaust.com/agency/internal/gen/sqlite"
)

type TaskTemplateSummary struct {
	TaskTemplate
	LatestVersion int64
}

type TaskTemplateApplicationParams struct {
	TaskID          string
	TemplateID      string
	TemplateVersion int64
	TemplateName    string
	ArgsJSON        string
	CreatedAt       int64
}

func (s *SqliteStore) CreateTaskTemplate(
	ctx context.Context,
	p CreateTaskTemplateParams,
) (TaskTemplate, error) {
	row, err := s.q.CreateTaskTemplate(ctx, sqlitedb.CreateTaskTemplateParams(p))
	if err != nil {
		return TaskTemplate{}, err
	}
	return fromSqliteTaskTemplate(row), nil
}

func (s *SqliteStore) GetTaskTemplate(
	ctx context.Context,
	id string,
) (TaskTemplateSummary, error) {
	row, err := s.q.GetTaskTemplate(ctx, id)
	if err != nil {
		return TaskTemplateSummary{}, err
	}
	return taskTemplateSummaryFromSqliteGet(row), nil
}

func (s *SqliteStore) GetTaskTemplateByProjectName(
	ctx context.Context,
	projectID, name string,
) (TaskTemplateSummary, error) {
	row, err := s.q.GetTaskTemplateByProjectName(
		ctx,
		sqlitedb.GetTaskTemplateByProjectNameParams{ProjectID: projectID, Name: name},
	)
	if err != nil {
		return TaskTemplateSummary{}, err
	}
	return taskTemplateSummaryFromSqliteByName(row), nil
}

func (s *SqliteStore) ListTaskTemplates(
	ctx context.Context,
	projectID string,
	includeArchived bool,
) ([]TaskTemplateSummary, error) {
	rows, err := s.q.ListTaskTemplates(ctx, sqlitedb.ListTaskTemplatesParams{
		ProjectID:       projectID,
		IncludeArchived: boolInt(includeArchived),
	})
	if err != nil {
		return nil, err
	}
	out := make([]TaskTemplateSummary, len(rows))
	for i, row := range rows {
		out[i] = taskTemplateSummaryFromSqliteList(row)
	}
	return out, nil
}

func (s *SqliteStore) UpdateTaskTemplate(
	ctx context.Context,
	p UpdateTaskTemplateParams,
) (TaskTemplate, error) {
	row, err := s.q.UpdateTaskTemplate(ctx, sqlitedb.UpdateTaskTemplateParams(p))
	if err != nil {
		return TaskTemplate{}, err
	}
	return fromSqliteTaskTemplate(row), nil
}

func (s *SqliteStore) SetTaskTemplateArchived(
	ctx context.Context,
	p SetTaskTemplateArchivedParams,
) (TaskTemplate, error) {
	row, err := s.q.SetTaskTemplateArchived(ctx, sqlitedb.SetTaskTemplateArchivedParams(p))
	if err != nil {
		return TaskTemplate{}, err
	}
	return fromSqliteTaskTemplate(row), nil
}

func (s *SqliteStore) CreateTaskTemplateVersion(
	ctx context.Context,
	p CreateTaskTemplateVersionParams,
) (TaskTemplateVersion, error) {
	row, err := s.q.CreateTaskTemplateVersion(
		ctx,
		sqlitedb.CreateTaskTemplateVersionParams(p),
	)
	if err != nil {
		return TaskTemplateVersion{}, err
	}
	return fromSqliteTaskTemplateVersion(row), nil
}

func (s *SqliteStore) GetTaskTemplateVersion(
	ctx context.Context,
	templateID string,
	version int64,
) (TaskTemplateVersion, error) {
	row, err := s.q.GetTaskTemplateVersion(ctx, sqlitedb.GetTaskTemplateVersionParams{
		TemplateID: templateID,
		Version:    version,
	})
	if err != nil {
		return TaskTemplateVersion{}, err
	}
	return fromSqliteTaskTemplateVersion(row), nil
}

func (s *SqliteStore) ListTaskTemplateVersions(
	ctx context.Context,
	templateID string,
) ([]TaskTemplateVersion, error) {
	rows, err := s.q.ListTaskTemplateVersions(ctx, templateID)
	if err != nil {
		return nil, err
	}
	return fromSqliteTaskTemplateVersionSlice(rows), nil
}

func (s *SqliteStore) CreateTaskTemplateApplication(
	ctx context.Context,
	p TaskTemplateApplicationParams,
) (int64, error) {
	return s.q.CreateTaskTemplateApplication(
		ctx,
		sqlitedb.CreateTaskTemplateApplicationParams{
			TemplateID:      nullString(p.TemplateID),
			TemplateVersion: p.TemplateVersion,
			TemplateName:    p.TemplateName,
			ArgsJson:        p.ArgsJSON,
			CreatedAt:       p.CreatedAt,
			TaskID:          p.TaskID,
		},
	)
}

func (s *SqliteStore) GetTaskTemplateApplication(
	ctx context.Context,
	taskID string,
) (TaskTemplateApplication, error) {
	row, err := s.q.GetTaskTemplateApplication(ctx, taskID)
	if err != nil {
		return TaskTemplateApplication{}, err
	}
	return fromSqliteTaskTemplateApplication(row), nil
}

func taskTemplateSummaryFromSqliteGet(row sqlitedb.GetTaskTemplateRow) TaskTemplateSummary {
	return TaskTemplateSummary{
		TaskTemplate: TaskTemplate{
			ID:               row.ID,
			ProjectID:        row.ProjectID,
			Name:             row.Name,
			Description:      row.Description,
			CreatedByActorID: row.CreatedByActorID,
			ArchivedAt:       row.ArchivedAt,
			CreatedAt:        row.CreatedAt,
			UpdatedAt:        row.UpdatedAt,
		},
		LatestVersion: row.LatestVersion,
	}
}

func taskTemplateSummaryFromSqliteByName(
	row sqlitedb.GetTaskTemplateByProjectNameRow,
) TaskTemplateSummary {
	return TaskTemplateSummary{
		TaskTemplate: TaskTemplate{
			ID:               row.ID,
			ProjectID:        row.ProjectID,
			Name:             row.Name,
			Description:      row.Description,
			CreatedByActorID: row.CreatedByActorID,
			ArchivedAt:       row.ArchivedAt,
			CreatedAt:        row.CreatedAt,
			UpdatedAt:        row.UpdatedAt,
		},
		LatestVersion: row.LatestVersion,
	}
}

func taskTemplateSummaryFromSqliteList(row sqlitedb.ListTaskTemplatesRow) TaskTemplateSummary {
	return TaskTemplateSummary{
		TaskTemplate: TaskTemplate{
			ID:               row.ID,
			ProjectID:        row.ProjectID,
			Name:             row.Name,
			Description:      row.Description,
			CreatedByActorID: row.CreatedByActorID,
			ArchivedAt:       row.ArchivedAt,
			CreatedAt:        row.CreatedAt,
			UpdatedAt:        row.UpdatedAt,
		},
		LatestVersion: row.LatestVersion,
	}
}

func (s *PgStore) CreateTaskTemplate(
	ctx context.Context,
	p CreateTaskTemplateParams,
) (TaskTemplate, error) {
	row, err := s.q.CreateTaskTemplate(ctx, pgdb.CreateTaskTemplateParams(p))
	if err != nil {
		return TaskTemplate{}, err
	}
	return fromPgTaskTemplate(row), nil
}

func (s *PgStore) GetTaskTemplate(
	ctx context.Context,
	id string,
) (TaskTemplateSummary, error) {
	row, err := s.q.GetTaskTemplate(ctx, id)
	if err != nil {
		return TaskTemplateSummary{}, err
	}
	return taskTemplateSummaryFromPgGet(row), nil
}

func (s *PgStore) GetTaskTemplateByProjectName(
	ctx context.Context,
	projectID, name string,
) (TaskTemplateSummary, error) {
	row, err := s.q.GetTaskTemplateByProjectName(
		ctx,
		pgdb.GetTaskTemplateByProjectNameParams{ProjectID: projectID, Name: name},
	)
	if err != nil {
		return TaskTemplateSummary{}, err
	}
	return taskTemplateSummaryFromPgByName(row), nil
}

func (s *PgStore) ListTaskTemplates(
	ctx context.Context,
	projectID string,
	includeArchived bool,
) ([]TaskTemplateSummary, error) {
	rows, err := s.q.ListTaskTemplates(ctx, pgdb.ListTaskTemplatesParams{
		ProjectID:       projectID,
		IncludeArchived: int32(boolInt(includeArchived)),
	})
	if err != nil {
		return nil, err
	}
	out := make([]TaskTemplateSummary, len(rows))
	for i, row := range rows {
		out[i] = taskTemplateSummaryFromPgList(row)
	}
	return out, nil
}

func (s *PgStore) UpdateTaskTemplate(
	ctx context.Context,
	p UpdateTaskTemplateParams,
) (TaskTemplate, error) {
	row, err := s.q.UpdateTaskTemplate(ctx, pgdb.UpdateTaskTemplateParams(p))
	if err != nil {
		return TaskTemplate{}, err
	}
	return fromPgTaskTemplate(row), nil
}

func (s *PgStore) SetTaskTemplateArchived(
	ctx context.Context,
	p SetTaskTemplateArchivedParams,
) (TaskTemplate, error) {
	row, err := s.q.SetTaskTemplateArchived(ctx, pgdb.SetTaskTemplateArchivedParams(p))
	if err != nil {
		return TaskTemplate{}, err
	}
	return fromPgTaskTemplate(row), nil
}

func (s *PgStore) CreateTaskTemplateVersion(
	ctx context.Context,
	p CreateTaskTemplateVersionParams,
) (TaskTemplateVersion, error) {
	row, err := s.q.CreateTaskTemplateVersion(ctx, pgdb.CreateTaskTemplateVersionParams(p))
	if err != nil {
		return TaskTemplateVersion{}, err
	}
	return fromPgTaskTemplateVersion(row), nil
}

func (s *PgStore) GetTaskTemplateVersion(
	ctx context.Context,
	templateID string,
	version int64,
) (TaskTemplateVersion, error) {
	row, err := s.q.GetTaskTemplateVersion(ctx, pgdb.GetTaskTemplateVersionParams{
		TemplateID: templateID,
		Version:    version,
	})
	if err != nil {
		return TaskTemplateVersion{}, err
	}
	return fromPgTaskTemplateVersion(row), nil
}

func (s *PgStore) ListTaskTemplateVersions(
	ctx context.Context,
	templateID string,
) ([]TaskTemplateVersion, error) {
	rows, err := s.q.ListTaskTemplateVersions(ctx, templateID)
	if err != nil {
		return nil, err
	}
	return fromPgTaskTemplateVersionSlice(rows), nil
}

func (s *PgStore) CreateTaskTemplateApplication(
	ctx context.Context,
	p TaskTemplateApplicationParams,
) (int64, error) {
	return s.q.CreateTaskTemplateApplication(
		ctx,
		pgdb.CreateTaskTemplateApplicationParams{
			TemplateID:      nullString(p.TemplateID),
			TemplateVersion: p.TemplateVersion,
			TemplateName:    p.TemplateName,
			ArgsJson:        p.ArgsJSON,
			CreatedAt:       p.CreatedAt,
			TaskID:          p.TaskID,
		},
	)
}

func (s *PgStore) GetTaskTemplateApplication(
	ctx context.Context,
	taskID string,
) (TaskTemplateApplication, error) {
	row, err := s.q.GetTaskTemplateApplication(ctx, taskID)
	if err != nil {
		return TaskTemplateApplication{}, err
	}
	return fromPgTaskTemplateApplication(row), nil
}

func taskTemplateSummaryFromPgGet(row pgdb.GetTaskTemplateRow) TaskTemplateSummary {
	return TaskTemplateSummary{
		TaskTemplate: TaskTemplate{
			ID:               row.ID,
			ProjectID:        row.ProjectID,
			Name:             row.Name,
			Description:      row.Description,
			CreatedByActorID: row.CreatedByActorID,
			ArchivedAt:       row.ArchivedAt,
			CreatedAt:        row.CreatedAt,
			UpdatedAt:        row.UpdatedAt,
		},
		LatestVersion: int64(row.LatestVersion),
	}
}

func taskTemplateSummaryFromPgByName(
	row pgdb.GetTaskTemplateByProjectNameRow,
) TaskTemplateSummary {
	return TaskTemplateSummary{
		TaskTemplate: TaskTemplate{
			ID:               row.ID,
			ProjectID:        row.ProjectID,
			Name:             row.Name,
			Description:      row.Description,
			CreatedByActorID: row.CreatedByActorID,
			ArchivedAt:       row.ArchivedAt,
			CreatedAt:        row.CreatedAt,
			UpdatedAt:        row.UpdatedAt,
		},
		LatestVersion: int64(row.LatestVersion),
	}
}

func taskTemplateSummaryFromPgList(row pgdb.ListTaskTemplatesRow) TaskTemplateSummary {
	return TaskTemplateSummary{
		TaskTemplate: TaskTemplate{
			ID:               row.ID,
			ProjectID:        row.ProjectID,
			Name:             row.Name,
			Description:      row.Description,
			CreatedByActorID: row.CreatedByActorID,
			ArchivedAt:       row.ArchivedAt,
			CreatedAt:        row.CreatedAt,
			UpdatedAt:        row.UpdatedAt,
		},
		LatestVersion: int64(row.LatestVersion),
	}
}

func nullString(value string) sql.NullString {
	return sql.NullString{String: value, Valid: value != ""}
}

func boolInt(value bool) int64 {
	if value {
		return 1
	}
	return 0
}
