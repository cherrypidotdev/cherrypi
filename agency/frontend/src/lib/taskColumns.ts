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

export type TaskColumnKey =
  | "state"
  | "project"
  | "title"
  | "progress"
  | "links"
  | "tags"
  | "badges"
  | "references"
  | "priority"
  | "size"
  | "assignee"
  | "cost";

export interface TaskColumn {
  key: TaskColumnKey;
  labelKey: string;

  widthClass?: string;

  sortKey?: string;

  hintKey?: string;

  skeleton: string;
}

export interface TaskColumnPreference {
  key: TaskColumnKey;
  visible: boolean;
}

export const TASK_COLUMNS: TaskColumn[] = [
  {
    key: "state",
    labelKey: "taskColumn.state",
    widthClass: "w-56",
    sortKey: "state",
    skeleton: "h-5 w-20",
  },
  {
    key: "project",
    labelKey: "taskColumn.project",
    widthClass: "w-32",
    sortKey: "project",
    skeleton: "w-20",
  },
  {
    key: "title",
    labelKey: "taskColumn.title",
    sortKey: "task",
    skeleton: "w-48",
  },
  {
    key: "progress",
    labelKey: "taskColumn.progress",
    widthClass: "w-32",
    sortKey: "progress",
    hintKey: "taskColumn.progressHint",
    skeleton: "w-16",
  },
  {
    key: "badges",
    labelKey: "taskColumn.badges",
    widthClass: "w-40",
    hintKey: "taskColumn.badgesHint",
    skeleton: "w-20",
  },
  {
    key: "links",
    labelKey: "taskColumn.links",
    widthClass: "w-28",
    skeleton: "w-12",
  },
  {
    key: "tags",
    labelKey: "taskColumn.tags",
    widthClass: "w-40",
    skeleton: "w-24",
  },
  {
    key: "references",
    labelKey: "taskColumn.references",
    widthClass: "w-40",
    skeleton: "w-24",
  },
  {
    key: "priority",
    labelKey: "taskColumn.priority",
    widthClass: "w-24",
    sortKey: "priority",
    skeleton: "w-12",
  },
  {
    key: "size",
    labelKey: "taskColumn.size",
    widthClass: "w-24",
    sortKey: "size",
    skeleton: "w-12",
  },
  {
    key: "assignee",
    labelKey: "taskColumn.assignee",
    widthClass: "w-40",
    skeleton: "w-20",
  },
  {
    key: "cost",
    labelKey: "taskColumn.cost",
    widthClass: "w-24",
    sortKey: "cost",
    hintKey: "taskColumn.costHint",
    skeleton: "w-12",
  },
];

export const TASK_COLUMN_KEYS = TASK_COLUMNS.map((column) => column.key);

export function defaultTaskColumnPreferences(): TaskColumnPreference[] {
  return TASK_COLUMNS.map((column) => ({ key: column.key, visible: true }));
}
