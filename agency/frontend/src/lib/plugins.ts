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

import { t } from "@/i18n/translate";
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
  schedule?: { every_seconds?: number };
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

export const PLUGIN_SCOPE_KEYS: Record<PluginScope, string> = {
  org: "pluginScope.org",
  project: "pluginScope.project",
  agent: "pluginScope.agent",
  user: "pluginScope.user",
};

export function scopeLabelKey(scope: PluginScope): string {
  return PLUGIN_SCOPE_KEYS[scope];
}

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
    schedule: parsed.schedule,
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
      return t("pluginHandler.http");
    case "tool":
      return t("pluginHandler.tool");
    case "event":
      return t("pluginHandler.event");
    case "tick":
      return t("pluginHandler.tick");
    case "task_tools":
      return t("pluginHandler.taskTools");
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
        ? t("pluginPermission.networkAnyHost")
        : t("pluginPermission.network"),
      detail: manifest.network.join(", "),
      tone: allowsAnyHost(manifest) ? "danger" : "info",
    });
  }
  if (manifest.routes.length) {
    out.push({
      key: "routes",
      label: t("pluginPermission.routes"),
      detail: manifest.routes
        .map((route) => `${route.methods.join("/")} ${route.path}`)
        .join(", "),
      tone: "warning",
    });
  }
  if (manifest.events.length) {
    out.push({
      key: "events",
      label: t("pluginPermission.events"),
      detail: manifest.events.join(", "),
      tone: "info",
    });
  }
  if (manifest.tools.length) {
    out.push({
      key: "tools",
      label: t("pluginPermission.tools"),
      detail: manifest.tools.map((tool) => tool.name).join(", "),
      tone: "info",
    });
  }
  if (manifest.schedule?.every_seconds) {
    out.push({
      key: "schedule",
      label: t("pluginPermission.schedule"),
      detail: t("pluginPermission.scheduleDetail", {
        seconds: manifest.schedule.every_seconds,
      }),
      tone: "info",
    });
  }
  const surfaces = manifestSurfaces(manifest);
  if (surfaces.length) {
    out.push({
      key: "settings",
      label: t("pluginPermission.settings"),
      detail: surfaces.map((scope) => t(scopeLabelKey(scope))).join(", "),
      tone: "info",
    });
  }

  const handlers = new Set(manifest.handlers);
  if (manifest.tools.length) handlers.add("tool");
  if (manifest.events.length) handlers.add("event");
  if (manifest.routes.length) handlers.add("http");
  if (manifest.schedule?.every_seconds) handlers.add("tick");
  if (handlers.size > 0) {
    out.push({
      key: "tasks",
      label: t("pluginPermission.tasks"),
      detail: t("pluginPermission.tasksDetail"),
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

export const PLUGIN_LOG_LEVEL_KEYS: Record<PluginLogLevel, string> = {
  trace: "pluginLog.level.trace",
  debug: "pluginLog.level.debug",
  info: "pluginLog.level.info",
  warn: "pluginLog.level.warn",
  error: "pluginLog.level.error",
};

export interface PluginSettingsGroup {
  id: string;
  title: string;
  description?: string;
  fields: string[];
}

export interface PluginTemplateVariable {
  type?: string;
  description?: string;
  sample?: unknown;
  items?: PluginTemplateVariable;
  properties?: Record<string, PluginTemplateVariable>;
}

export interface PluginTemplateMeta {
  engine?: string;
  variables?: Record<string, PluginTemplateVariable>;
}

export function pluginSettingsGroups(
  schema: string | undefined,
): PluginSettingsGroup[] {
  const parsed = parseJSONSchema(schema) as Record<string, unknown> | null;
  const groups = parsed?.["x-groups"];
  if (!Array.isArray(groups)) return [];
  return groups
    .filter(
      (group): group is Record<string, unknown> =>
        !!group && typeof group === "object" && !Array.isArray(group),
    )
    .map((group) => ({
      id: typeof group.id === "string" ? group.id : "",
      title: typeof group.title === "string" ? group.title : "",
      description:
        typeof group.description === "string" ? group.description : undefined,
      fields: Array.isArray(group.fields)
        ? group.fields.filter(
            (field): field is string => typeof field === "string",
          )
        : [],
    }))
    .filter((group) => group.id && group.title);
}

export function pluginTemplateMeta(
  property: Record<string, unknown> | undefined,
): PluginTemplateMeta | null {
  if (!property || property.format !== "template") return null;
  const raw = property["x-template"];
  if (!raw || typeof raw !== "object" || Array.isArray(raw)) return null;
  const meta = raw as PluginTemplateMeta;
  return {
    engine: typeof meta.engine === "string" ? meta.engine : "jinja",
    variables:
      meta.variables && typeof meta.variables === "object"
        ? meta.variables
        : {},
  };
}

export function pluginTemplateDefault(
  property: Record<string, unknown> | undefined,
): string {
  return typeof property?.default === "string" ? property.default : "";
}

function templateVariableZero(variable: PluginTemplateVariable): unknown {
  switch (variable.type) {
    case "array":
      return [];
    case "object":
      return {};
    case "number":
    case "integer":
      return 0;
    case "boolean":
      return false;
    default:
      return "";
  }
}

export function pluginTemplateSamples(
  variables: Record<string, PluginTemplateVariable> | undefined,
): Record<string, unknown> {
  const out: Record<string, unknown> = {};
  for (const [path, variable] of Object.entries(variables ?? {})) {
    out[path] =
      variable.sample !== undefined
        ? variable.sample
        : templateVariableZero(variable);
  }
  return out;
}

export function pluginTemplateVariables(
  schema: string | undefined,
): Record<string, PluginTemplateVariable> {
  const parsed = parseJSONSchema(schema) as Record<string, unknown> | null;
  const properties = parsed?.properties;
  if (
    !properties ||
    typeof properties !== "object" ||
    Array.isArray(properties)
  ) {
    return {};
  }
  const out: Record<string, PluginTemplateVariable> = {};
  for (const property of Object.values(
    properties as Record<string, Record<string, unknown>>,
  )) {
    const meta = pluginTemplateMeta(property);
    for (const [path, variable] of Object.entries(meta?.variables ?? {})) {
      if (!(path in out)) out[path] = variable;
    }
  }
  return out;
}
