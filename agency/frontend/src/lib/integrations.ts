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

export interface IntegrationScope {
  kind: "org" | "project" | "team" | "agent" | "group";
  id: string;
}

export interface IntegrationValue {
  value?: string;
  env?: string;
  secret?: string;
  hasSecret?: boolean;
}

export interface ToolFilter {
  include: string[];
  exclude: string[];
}

export interface IntegrationInstall {
  scope: IntegrationScope;
  enabled: boolean;
}

export interface McpServerEntry {
  id: number;
  name: string;
  description: string;
  owner: IntegrationScope;
  url: string;
  headers: Record<string, IntegrationValue>;
  connectTimeoutMs: number;
  requestTimeoutMs: number;
  tools: ToolFilter;
  directTools: boolean;
  enabled: boolean;
  installs: IntegrationInstall[];
  updatedAt: string;
}

export interface SkillFile {
  path: string;
  text?: string;
}

export interface SkillOrigin {
  sourceId: number;
  slug: string;
  path: string;
  commit: string;
  missing: boolean;
}

export interface SkillEntry {
  id: number;
  name: string;
  description: string;
  owner: IntegrationScope;
  body: string;
  files: SkillFile[];
  disableModelInvocation: boolean;
  enabled: boolean;
  installs: IntegrationInstall[];

  source?: SkillOrigin;
  managed: boolean;
  updatedAt: string;
}

export interface GitCLIStatus {
  path: string;
  version: string;
  ok: boolean;
}

export interface GitSourcesStatus {
  enabled: boolean;
  ready: boolean;
  git: GitCLIStatus;
  ssh: GitCLIStatus;
  message?: string;
}

export interface GitHostKey {
  type: string;
  fingerprint: string;
}

export interface GitCredential {
  id: number;
  name: string;
  kind: "token" | "ssh_key";
  host: string;
  port: number;
  username: string;
  hasSecret: boolean;
  knownHosts: string;
  publicKey: string;
  fingerprint: string;
  sources: number;
  updatedAt: string;
}

export interface GitCredentialInput {
  id?: number;
  name: string;
  kind: "token" | "ssh_key";
  host: string;
  port: number;
  username: string;
  secret: string;
  keepSecret: boolean;
  knownHosts: string;
}

export interface GitSourceSkill {
  path: string;
  rawName: string;
  name: string;
  description: string;
  body?: string;
  files?: SkillFile[];
  disableModelInvocation: boolean;
  status: "added" | "updated" | "unchanged" | "removed" | "conflict";
  entryId?: number;
  error?: string;
}

export interface GitSourceProblem {
  path: string;
  message: string;
}

export interface GitSource {
  id: number;
  slug: string;
  name: string;
  description: string;
  owner: IntegrationScope;
  repoUrl: string;
  ref: string;
  pathGlob: string;
  credentialId: number;
  credentialName: string;
  updateMode: "staged" | "auto";
  pollIntervalSeconds: number;
  paused: boolean;
  installs: IntegrationInstall[];
  syncState: "" | "ok" | "pending" | "error";
  syncError: string;
  syncedCommit: string;
  syncedAt: string;
  checkedAt: string;
  pendingCommit: string;
  pending: GitSourceSkill[];
  problems: GitSourceProblem[];
  skillCount: number;
  updatedAt: string;
}

export interface GitSourceInput {
  id?: number;
  slug: string;
  name: string;
  description: string;
  owner: IntegrationScope;
  repoUrl: string;
  ref: string;
  pathGlob: string;
  credentialId: number;
  updateMode: "staged" | "auto";
  pollIntervalSeconds: number;
  paused: boolean;
  installs: IntegrationInstall[];
}

export interface GitSourcePreview {
  commit: string;
  ref: string;
  skills: GitSourceSkill[];
  problems: GitSourceProblem[];
}

export interface IntegrationLibrary {
  servers: McpServerEntry[];
  skills: SkillEntry[];
  seal: boolean;
  git?: GitSourcesStatus;
  sources?: GitSource[];
  credentials?: GitCredential[];

  admin?: boolean;
}

export interface IntegrationInput {
  id?: number;
  name: string;
  description: string;
  owner: IntegrationScope;
  enabled: boolean;
  installs: IntegrationInstall[];
  url?: string;
  headers?: Record<string, IntegrationValue>;
  connectTimeoutMs?: number;
  requestTimeoutMs?: number;
  tools?: ToolFilter;
  directTools?: boolean;
  body?: string;
  files?: SkillFile[];
  disableModelInvocation?: boolean;
}

export interface ReportedSkill {
  name: string;
  description: string;
  origin: "agency" | "host";
  owner: string;
  disableModelInvocation: boolean;
}

export interface ReportedServer {
  name: string;
  origin: "agency" | "host";
  owner: string;
  transport: string;
  connected: boolean;
  toolCount: number;
  error: string;
}

export interface TaskIntegrationReport {
  taskId: string;
  report: { skills: ReportedSkill[]; servers: ReportedServer[] };
  configRevision: string;
  reportedAt: number;
  recorded: boolean;
}

const BASE = "/api/rest/integrations";

export const SCOPE_KINDS = [
  { value: "org", labelKey: "library.scope.org" },
  { value: "project", labelKey: "library.scope.project" },
  { value: "team", labelKey: "library.scope.team" },
  { value: "group", labelKey: "library.scope.group" },
  { value: "agent", labelKey: "library.scope.agent" },
] as const;

const SCOPE_NAMES: Record<IntegrationScope["kind"], string> = {
  org: "Org-wide",
  project: "Project",
  team: "Team",
  group: "Group",
  agent: "Agent",
};

export function scopeLabel(scope: IntegrationScope | string): string {
  const parsed = typeof scope === "string" ? parseScope(scope) : scope;
  if (!parsed) return "";
  if (parsed.kind === "org") return SCOPE_NAMES.org;
  return `${SCOPE_NAMES[parsed.kind]} ${parsed.id}`;
}

export function parseScope(raw: string): IntegrationScope | undefined {
  const [kind, ...rest] = raw.split(":");
  const id = rest.join(":");
  if (
    kind !== "org" &&
    kind !== "project" &&
    kind !== "team" &&
    kind !== "group" &&
    kind !== "agent"
  ) {
    return undefined;
  }
  return { kind, id };
}

export function emptyValues(): Record<string, IntegrationValue> {
  return {};
}

export const integrationsApi = {
  list(): Promise<IntegrationLibrary> {
    return request<IntegrationLibrary>("GET", BASE);
  },
  saveServer(input: IntegrationInput): Promise<McpServerEntry> {
    return request<McpServerEntry>("POST", `${BASE}/servers`, input);
  },
  saveSkill(input: IntegrationInput): Promise<SkillEntry> {
    return request<SkillEntry>("POST", `${BASE}/skills`, input);
  },
  deleteServer(id: number): Promise<void> {
    return request<void>("DELETE", `${BASE}/servers/${id}`);
  },
  deleteSkill(id: number): Promise<void> {
    return request<void>("DELETE", `${BASE}/skills/${id}`);
  },

  setInstall(
    kind: "mcp" | "skill",
    id: number,
    scope: IntegrationScope,
    enabled: boolean,
  ): Promise<void> {
    return request<void>(
      "PUT",
      `${BASE}/${kind === "mcp" ? "servers" : "skills"}/${id}/install`,
      {
        scope,
        enabled,
      },
    );
  },
  removeInstall(
    kind: "mcp" | "skill",
    id: number,
    scope: IntegrationScope,
  ): Promise<void> {
    return request<void>(
      "DELETE",
      `${BASE}/${kind === "mcp" ? "servers" : "skills"}/${id}/install`,
      { scope },
    );
  },

  taskReport(taskId: string): Promise<TaskIntegrationReport> {
    return request<TaskIntegrationReport>("GET", `${BASE}/task/${taskId}`);
  },

  setGitSources(enabled: boolean): Promise<GitSourcesStatus> {
    return request<GitSourcesStatus>("POST", `${BASE}/git`, { enabled });
  },
  saveCredential(input: GitCredentialInput): Promise<GitCredential> {
    return request<GitCredential>("POST", `${BASE}/git-credentials`, input);
  },
  deleteCredential(id: number): Promise<void> {
    return request<void>("DELETE", `${BASE}/git-credentials/${id}`);
  },

  scanHostKeys(
    host: string,
    port: number,
  ): Promise<{ keys: GitHostKey[]; knownHosts: string }> {
    return request<{ keys: GitHostKey[]; knownHosts: string }>(
      "POST",
      `${BASE}/git-credentials/scan`,
      { host, port },
    );
  },
  getSource(id: number): Promise<GitSource> {
    return request<GitSource>("GET", `${BASE}/sources/${id}`);
  },
  saveSource(input: GitSourceInput): Promise<GitSource> {
    return request<GitSource>("POST", `${BASE}/sources`, input);
  },

  previewSource(input: GitSourceInput): Promise<GitSourcePreview> {
    return request<GitSourcePreview>("POST", `${BASE}/sources/preview`, input);
  },
  deleteSource(id: number): Promise<void> {
    return request<void>("DELETE", `${BASE}/sources/${id}`);
  },

  syncSource(id: number): Promise<GitSource> {
    return request<GitSource>("POST", `${BASE}/sources/${id}/sync`);
  },

  applySource(id: number): Promise<GitSource> {
    return request<GitSource>("POST", `${BASE}/sources/${id}/apply`);
  },

  setSourceInstall(
    id: number,
    scope: IntegrationScope,
    enabled: boolean,
  ): Promise<void> {
    return request<void>("PUT", `${BASE}/sources/${id}/install`, {
      scope,
      enabled,
    });
  },
  removeSourceInstall(id: number, scope: IntegrationScope): Promise<void> {
    return request<void>("DELETE", `${BASE}/sources/${id}/install`, { scope });
  },

  detachSkill(id: number): Promise<SkillEntry> {
    return request<SkillEntry>("POST", `${BASE}/skills/${id}/detach`);
  },

  setSkillEnabled(id: number, enabled: boolean): Promise<void> {
    return request<void>("PUT", `${BASE}/skills/${id}/enabled`, { enabled });
  },
};
