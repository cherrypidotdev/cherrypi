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

export interface ModelCost {
  input: number;
  output: number;
  cacheRead: number;
  cacheWrite: number;
}

export interface AgentModel {
  id: string;
  name: string;
  reasoning: boolean;
  input: string[];

  thinkingLevels: string[];

  forcePlan: boolean;
  contextWindow: number;
  maxTokens: number;
  cost: ModelCost;
  enabled: boolean;
}

export interface ModelScope {
  kind: "org" | "agent" | "team" | "project" | "group";
  id: string;
  modelId?: string;
}

export interface AgentProvider {
  slug: string;
  name: string;
  api: string;
  baseUrl: string;
  hasKey: boolean;
  headers: Record<string, string> | null;
  enabled: boolean;
  models: AgentModel[];
  grantees: ModelScope[];
}

export interface AgentConfig {
  revision: string;
  providers: AgentProvider[];

  sealing: boolean;
}

export interface DiscoveryInput {
  slug?: string;
  api: string;
  baseUrl: string;
  apiKey?: string;
  headers?: Record<string, string>;
}

export interface ProviderTestResult {
  ok: boolean;
  status?: number;
  latencyMs: number;
  message: string;
}

export interface ProviderInput {
  slug: string;
  name: string;
  api: string;
  baseUrl: string;
  apiKey?: string;
  clearKey?: boolean;
  headers?: Record<string, string>;
  enabled: boolean;
  models: AgentModel[];
  grantees: ModelScope[];
}

const BASE = "/api/rest/agent-config/admin";

export const WIRE_APIS = [
  { value: "openai-completions", label: "[OAI] Chat Completions" },
  { value: "openai-responses", label: "[OAI] Responses" },
  { value: "anthropic-messages", label: "Anthropic Messages" },
  { value: "google-generative-ai", label: "Google Gemini" },
  { value: "pi-messages", label: "Pi messages gateway" },
];

export const SCOPE_KINDS = [
  { value: "org", label: "Everyone" },
  { value: "agent", label: "One agent" },
  { value: "team", label: "One team" },
  { value: "project", label: "One project" },
  { value: "group", label: "One agent group" },
];

export const agentConfigApi = {
  get(): Promise<AgentConfig> {
    return request<AgentConfig>("GET", BASE);
  },
  saveProvider(input: ProviderInput): Promise<AgentConfig> {
    return request<AgentConfig>(
      "POST",
      `${BASE}/providers/${input.slug}`,
      input,
    );
  },
  deleteProvider(slug: string): Promise<void> {
    return request<void>("DELETE", `${BASE}/providers/${slug}`);
  },
  testProvider(slug: string): Promise<ProviderTestResult> {
    return request<ProviderTestResult>("POST", `${BASE}/test/${slug}`);
  },

  discoverModels(input: DiscoveryInput): Promise<AgentModel[]> {
    return request<{ models: AgentModel[] }>("POST", `${BASE}/discover`, {
      ...input,
    }).then((result) => result.models);
  },
};

export function emptyModel(): AgentModel {
  return {
    id: "",
    name: "",
    reasoning: false,
    input: ["text"],
    thinkingLevels: [],
    forcePlan: false,
    contextWindow: 128000,
    maxTokens: 8192,
    cost: { input: 0, output: 0, cacheRead: 0, cacheWrite: 0 },
    enabled: true,
  };
}

export function emptyProvider(): ProviderInput {
  return {
    slug: "",
    name: "",
    api: "openai-completions",
    baseUrl: "",
    apiKey: "",
    enabled: true,

    models: [],
    grantees: [{ kind: "org", id: "" }],
  };
}
