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

import {
  PluginSettingErrorsSchema,
  type Plugin,
  type PluginPreview,
  type PluginSettingError as SettingErrorProto,
} from "@cherrypi/sdk";
import { ConnectError } from "@connectrpc/connect";

export interface PluginTool {
  name: string;
  description: string;
  input_schema?: unknown;
}

export interface PluginRoute {
  path: string;
  methods: string[];
  description?: string;
}

export type { PluginPreview };

export interface PluginManifest {
  id: string;
  name: string;
  version: string;
  description: string;
  runtime: string;
  entry: string;
  handlers: string[];
  settings: Record<string, unknown>;
  tools: PluginTool[];
  events: string[];
  network: string[];
  routes: PluginRoute[];
  requires_agency?: string;
  limits?: {
    timeout_seconds?: number;
    max_response_bytes?: number;
    max_request_bytes?: number;
    max_concurrent?: number;
  };
}

export interface SettingFieldError {
  path: string;
  errorText: string;
}

export function settingErrorsFrom(err: unknown): SettingFieldError[] {
  const connectErr =
    err instanceof ConnectError ? err : ConnectError.from(err as Error);
  const detail = connectErr.findDetails(PluginSettingErrorsSchema)[0];
  if (!detail) return [];
  return detail.errors.map((item: SettingErrorProto) => ({
    path: item.path,
    errorText: item.errorText,
  }));
}

export const PLUGIN_SCOPES = ["org", "project", "agent", "user"] as const;
export type PluginScope = (typeof PLUGIN_SCOPES)[number];

export const PLUGIN_SCOPE_LABELS: Record<PluginScope, string> = {
  org: "Organization",
  project: "Project",
  agent: "Agent",
  user: "User",
};

export function parseManifest(plugin: Plugin): PluginManifest {
  return parseManifestJSON(plugin.manifestJson, {
    id: plugin.id,
    name: plugin.name,
    version: plugin.version,
    description: plugin.description,
    runtime: plugin.runtime,
    entry: plugin.entry,
  });
}

export function parseManifestJSON(
  raw: string | undefined,
  fallback: Partial<PluginManifest> = {},
): PluginManifest {
  let parsed: Partial<PluginManifest> = {};
  try {
    parsed = JSON.parse(raw || "{}") as PluginManifest;
  } catch {
    parsed = {};
  }
  return {
    id: parsed.id ?? fallback.id ?? "",
    name: parsed.name ?? fallback.name ?? "",
    version: parsed.version ?? fallback.version ?? "",
    description: parsed.description ?? fallback.description ?? "",
    runtime: parsed.runtime ?? fallback.runtime ?? "lua",
    entry: parsed.entry ?? fallback.entry ?? "main.lua",
    handlers: parsed.handlers ?? [],
    settings: parsed.settings ?? {},
    tools: parsed.tools ?? [],
    events: parsed.events ?? [],
    network: parsed.network ?? [],
    routes: parsed.routes ?? [],
    requires_agency: parsed.requires_agency,
    limits: parsed.limits,
  };
}

export function parseJSONObject(
  value: string | undefined,
): Record<string, unknown> {
  if (!value) return {};
  try {
    const parsed = JSON.parse(value);
    if (parsed && typeof parsed === "object" && !Array.isArray(parsed)) {
      return parsed as Record<string, unknown>;
    }
  } catch {}
  return {};
}

export function parseJSONSchema(value: string | undefined): unknown {
  if (!value) return null;
  try {
    return JSON.parse(value);
  } catch {
    return null;
  }
}

export type PluginSchemaForm = "form" | "json" | "none";

export function pluginSchemaForm(value: string | undefined): PluginSchemaForm {
  const schema = parseJSONSchema(value) as Record<string, unknown> | null;
  if (!schema || typeof schema !== "object" || Array.isArray(schema)) {
    return "json";
  }
  if (!("properties" in schema)) return "json";
  const properties = schema.properties;
  if (
    !properties ||
    typeof properties !== "object" ||
    Array.isArray(properties)
  ) {
    return "none";
  }
  return Object.keys(properties as Record<string, unknown>).length
    ? "form"
    : "none";
}

export const NETWORK_ANY_HOST = "*";

export function allowsAnyHost(manifest: PluginManifest): boolean {
  return manifest.network.includes(NETWORK_ANY_HOST);
}

export function hasSurface(manifest: PluginManifest, scope: string): boolean {
  return Object.prototype.hasOwnProperty.call(manifest.settings, scope);
}

export function manifestSurfaces(manifest: PluginManifest): PluginScope[] {
  return PLUGIN_SCOPES.filter((scope) => hasSurface(manifest, scope));
}

export function pluginActorId(plugin: Plugin): string {
  return plugin.actorId || "";
}

export function handlerLabel(handler: string): string {
  switch (handler) {
    case "http":
      return "HTTP requests";
    case "tool":
      return "Agent tools";
    case "event":
      return "Agency events";
    default:
      return handler;
  }
}

export interface PluginPermission {
  key: string;
  label: string;
  detail: string;
  tone: "info" | "warning" | "danger";
}

export function pluginPermissions(
  manifest: PluginManifest,
): PluginPermission[] {
  const out: PluginPermission[] = [];
  if (manifest.network.length) {
    out.push({
      key: "network",
      label: allowsAnyHost(manifest)
        ? "Outbound network: any host"
        : "Outbound network",
      detail: manifest.network.join(", "),
      tone: allowsAnyHost(manifest) ? "danger" : "info",
    });
  }
  if (manifest.routes.length) {
    out.push({
      key: "routes",
      label: "Public HTTP routes",
      detail: manifest.routes
        .map((route) => `${route.methods.join("/")} ${route.path}`)
        .join(", "),
      tone: "warning",
    });
  }
  if (manifest.events.length) {
    out.push({
      key: "events",
      label: "Agency events",
      detail: manifest.events.join(", "),
      tone: "info",
    });
  }
  if (manifest.tools.length) {
    out.push({
      key: "tools",
      label: "Agent tools",
      detail: manifest.tools.map((tool) => tool.name).join(", "),
      tone: "info",
    });
  }
  const surfaces = manifestSurfaces(manifest);
  if (surfaces.length) {
    out.push({
      key: "settings",
      label: "Settings surfaces",
      detail: surfaces.map((scope) => PLUGIN_SCOPE_LABELS[scope]).join(", "),
      tone: "info",
    });
  }

  const handlers = new Set(manifest.handlers);
  if (manifest.tools.length) handlers.add("tool");
  if (manifest.events.length) handlers.add("event");
  if (manifest.routes.length) handlers.add("http");
  if (handlers.size > 0) {
    out.push({
      key: "tasks",
      label: "Task creation",
      detail:
        "May create tasks in projects where it is installed, capped by each project's switches.",
      tone: "warning",
    });
  }
  return out;
}

export const PLUGIN_LOG_LEVELS = [
  "trace",
  "debug",
  "info",
  "warn",
  "error",
] as const;
export type PluginLogLevel = (typeof PLUGIN_LOG_LEVELS)[number];

export const PLUGIN_LOG_LEVEL_LABELS: Record<PluginLogLevel, string> = {
  trace: "Trace",
  debug: "Debug",
  info: "Info",
  warn: "Warn",
  error: "Error",
};
