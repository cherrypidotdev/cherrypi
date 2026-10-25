-- CherryPi: A multi-agent-multi-human self-hosted agency
-- Copyright (C) 2026 CherryPi authors <cherrypi.dev>
--
-- This program is free software: you can redistribute it and/or modify
-- it under the terms of the GNU Affero General Public License as published
-- by the Free Software Foundation, either version 3 of the License, or
-- (at your option) any later version.
--
-- This program is distributed in the hope that it will be useful,
-- but WITHOUT ANY WARRANTY; without even the implied warranty of
-- MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
-- GNU Affero General Public License for more details.
--
-- You should have received a copy of the GNU Affero General Public License
-- along with this program.  If not, see <https://www.gnu.org/licenses/>.




-- name: ListTaskArtifacts :many
SELECT * FROM task_artifacts
WHERE task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id))
ORDER BY position, created_at, artifact_id;

-- name: ListArtifactParts :many
SELECT * FROM artifact_parts
WHERE task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id))
ORDER BY artifact_id, position, id;

-- name: GetTaskArtifact :one
SELECT * FROM task_artifacts
WHERE task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id))
  AND artifact_id = sqlc.arg(artifact_id);

-- name: MaxTaskArtifactPosition :one
SELECT CAST(COALESCE(MAX(position), -1) AS BIGINT) FROM task_artifacts
WHERE task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id));

-- name: MaxArtifactPartPosition :one
SELECT CAST(COALESCE(MAX(position), -1) AS BIGINT) FROM artifact_parts
WHERE task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id))
  AND artifact_id = sqlc.arg(artifact_id);

-- name: UpsertTaskArtifact :exec
INSERT INTO task_artifacts (
  task_rowid, artifact_id, name, description, metadata_json, streaming,
  position, created_at, updated_at
)
VALUES (
  (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id)),
  sqlc.arg(artifact_id), sqlc.arg(name), sqlc.arg(description),
  sqlc.arg(metadata_json), sqlc.arg(streaming),
  sqlc.arg(position), sqlc.arg(created_at), sqlc.arg(updated_at)
)
ON CONFLICT (task_rowid, artifact_id) DO UPDATE SET
  name = excluded.name,
  description = excluded.description,
  metadata_json = excluded.metadata_json,
  streaming = excluded.streaming,
  updated_at = excluded.updated_at;

-- name: SetTaskArtifactStreaming :exec
UPDATE task_artifacts
SET streaming = sqlc.arg(streaming), updated_at = sqlc.arg(updated_at)
WHERE task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id))
  AND artifact_id = sqlc.arg(artifact_id);

-- name: InsertArtifactPart :exec
INSERT INTO artifact_parts (
  task_rowid, artifact_id, position, kind, text, data_json, url,
  attachment_id, filename, media_type, metadata_json
)
VALUES (
  (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id)),
  sqlc.arg(artifact_id), sqlc.arg(position), sqlc.arg(kind), sqlc.arg(text),
  sqlc.arg(data_json), sqlc.arg(url), sqlc.arg(attachment_id),
  sqlc.arg(filename), sqlc.arg(media_type), sqlc.arg(metadata_json)
);

-- name: DeleteArtifactParts :exec
DELETE FROM artifact_parts
WHERE task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id))
  AND artifact_id = sqlc.arg(artifact_id);

-- name: DeleteTaskArtifact :exec
DELETE FROM task_artifacts
WHERE task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id))
  AND artifact_id = sqlc.arg(artifact_id);

-- name: DeleteTaskArtifacts :exec
DELETE FROM task_artifacts
WHERE task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id));

-- name: DeleteArtifactPartsByTask :exec
DELETE FROM artifact_parts
WHERE task_rowid = (SELECT t.rowid FROM tasks t WHERE t.id = sqlc.arg(task_id));

-- name: DeleteTaskArtifactsByProjectAgent :exec
DELETE FROM task_artifacts WHERE task_rowid IN (
  SELECT rowid FROM tasks
  WHERE project_id = sqlc.arg(project_id) AND assignee_actor_id = (SELECT actor_id FROM agents WHERE agents.id = sqlc.arg(agent_id))
);

-- name: DeleteArtifactPartsByProjectAgent :exec
DELETE FROM artifact_parts WHERE task_rowid IN (
  SELECT rowid FROM tasks
  WHERE project_id = sqlc.arg(project_id) AND assignee_actor_id = (SELECT actor_id FROM agents WHERE agents.id = sqlc.arg(agent_id))
);

-- name: DeleteArtifactPartsByAttachment :exec
DELETE FROM artifact_parts WHERE attachment_id = sqlc.arg(attachment_id);

-- name: DeleteEmptyTaskArtifacts :exec


DELETE FROM task_artifacts
WHERE NOT EXISTS (
  SELECT 1 FROM artifact_parts p
  WHERE p.task_rowid = task_artifacts.task_rowid
    AND p.artifact_id = task_artifacts.artifact_id
);
