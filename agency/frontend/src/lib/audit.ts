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

import { request } from "@/lib/account";

/** One of the five domain logs the explorer unions. */
export type AuditScope = "task" | "agent" | "project" | "team" | "user";

export const AUDIT_SCOPES: AuditScope[] = [
  "task",
  "agent",
  "project",
  "team",
  "user",
];

/** One domain-log entry, from any scope. */
export interface AuditEntry {
  id: number;
  scope: AuditScope;
  scope_id: string;
  scope_label: string;
  at: string;
  actor_id: string;
  actor_kind: string;
  actor_username: string;
  actor_name: string;
  kind: string;
  field: string;
  old_value: string;
  new_value: string;
  message: string;
  data_json: string;
}

/** The explorer's filters; empty values mean "no filter". */
export interface AuditFilter {
  actor?: string;
  scope?: AuditScope | "";
  kind?: string;
  since?: string;
  until?: string;
  q?: string;
  limit?: number;
  cursor?: string;
}

export interface AuditPage {
  logs: AuditEntry[];
  next_cursor: string;
}

function queryString(filter: AuditFilter): string {
  const params = new URLSearchParams();
  if (filter.actor) params.set("actor", filter.actor);
  if (filter.scope) params.set("scope", filter.scope);
  if (filter.kind) params.set("kind", filter.kind);
  if (filter.since) params.set("since", filter.since);
  if (filter.until) params.set("until", filter.until);
  if (filter.q) params.set("q", filter.q);
  if (filter.limit) params.set("limit", String(filter.limit));
  if (filter.cursor) params.set("cursor", filter.cursor);
  const encoded = params.toString();
  return encoded ? `?${encoded}` : "";
}

/** One page of the admin events explorer, newest first. */
export async function listEvents(filter: AuditFilter = {}): Promise<AuditPage> {
  return request<AuditPage>("GET", `/api/rest/events${queryString(filter)}`);
}

/** The export URL for a filter; the link is fetched with the session cookie. */
export function eventsExportUrl(
  format: "csv" | "json",
  filter: AuditFilter = {},
): string {
  return `/api/rest/events/export.${format}${queryString(filter)}`;
}

/** The configured audit retention in days. */
export async function getEventsSettings(): Promise<{ retention_days: number }> {
  return request<{ retention_days: number }>(
    "GET",
    "/api/rest/events/settings",
  );
}

/** Persists the audit retention in days. */
export async function setEventsSettings(
  retentionDays: number,
): Promise<{ retention_days: number }> {
  return request<{ retention_days: number }>(
    "PUT",
    "/api/rest/events/settings",
    { retention_days: retentionDays },
  );
}

/** The caller's own domain log: auth events and account mutations. */
export async function listAccountLogs(
  before = 0,
  limit = 50,
): Promise<{ logs: AuditEntry[] }> {
  const params = new URLSearchParams();
  if (before > 0) params.set("before", String(before));
  params.set("limit", String(limit));
  return request<{ logs: AuditEntry[] }>(
    "GET",
    `/api/rest/account/logs?${params.toString()}`,
  );
}

/** An agent's mutation domain log; owner or admin only. */
export async function listAgentDomainLogs(
  agentId: string | number,
  before = 0,
  limit = 50,
): Promise<{ logs: AuditEntry[] }> {
  const params = new URLSearchParams();
  if (before > 0) params.set("before", String(before));
  params.set("limit", String(limit));
  return request<{ logs: AuditEntry[] }>(
    "GET",
    `/api/rest/agents/${agentId}/logs?${params.toString()}`,
  );
}
