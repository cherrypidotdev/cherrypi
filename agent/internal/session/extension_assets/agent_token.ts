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

import { readFile } from "node:fs/promises";

const TOKEN_PREFIX = "pat_";

export type AgentTokenSource = "env" | "file";

export interface AgentToken {
  source: AgentTokenSource;

  label: string;

  value: string;
}

export interface AgentTokenAttempt {
  source: AgentTokenSource;
  label: string;

  status: number;
}

export interface AgentFetchOutcome {
  res?: Response;

  token?: AgentToken;

  attempts: AgentTokenAttempt[];

  error?: string;

  failedOver: boolean;
}

export async function agentTokens(): Promise<AgentToken[]> {
  const tokens: AgentToken[] = [];

  const env = normalize(process.env.CHERRYPI_AGENT_TOKEN);
  if (env) {
    tokens.push({
      source: "env",
      label: "CHERRYPI_AGENT_TOKEN",
      value: env,
    });
  }

  const file = (process.env.CHERRYPI_AGENT_TOKEN_FILE ?? "").trim();
  if (file) {
    let raw = "";
    try {
      raw = await readFile(file, "utf8");
    } catch {}
    const value = normalize(raw);
    if (value && value !== env) {
      tokens.push({
        source: "file",
        label: `CHERRYPI_AGENT_TOKEN_FILE ${file}`,
        value,
      });
    }
  }

  return tokens;
}

export async function fetchWithAgentToken(
  url: string,
  init: RequestInit = {},
  tokens?: AgentToken[],
): Promise<AgentFetchOutcome> {
  const candidates = tokens ?? (await agentTokens());
  const attempts: AgentTokenAttempt[] = [];
  let failedOver = false;

  for (const [index, token] of candidates.entries()) {
    const headers = new Headers(init.headers);
    headers.set("Authorization", `Bearer ${token.value}`);

    let res: Response;
    try {
      res = await fetch(url, { ...init, headers });
    } catch (err) {
      attempts.push({ source: token.source, label: token.label, status: 0 });
      return { attempts, error: String(err), failedOver };
    }

    attempts.push({
      source: token.source,
      label: token.label,
      status: res.status,
    });

    const next = candidates[index + 1];
    if (res.status === 401 && next) {
      failedOver = true;
      console.error(
        `cherrypi: ${token.label} was rejected (HTTP 401); retrying with ${next.label}`,
      );
      await discard(res);
      continue;
    }

    return { res, token, attempts, failedOver };
  }

  return { attempts, error: "no agent token is configured", failedOver };
}

export function credentialNote(outcome: AgentFetchOutcome): string {
  if (!outcome.failedOver) {
    return "";
  }
  const rejected = outcome.attempts.slice(0, -1);
  if (rejected.length === 0) {
    return "";
  }
  const accepted = outcome.attempts[outcome.attempts.length - 1];
  const failed = rejected
    .map((attempt) => `${attempt.label} rejected (HTTP ${attempt.status})`)
    .join("; ");
  const used = accepted ? `; retried with ${accepted.label}` : "";
  return `\n\n[credential: ${failed}${used}]`;
}

/**
 * Line listing every credential a failed call tried, when at least one was
 * rejected with 401, so a stale one is visible instead of the failure looking
 * like an outage. Empty for a plain outage or server error.
 */
export function credentialsLine(outcome: AgentFetchOutcome): string {
  if (!outcome.attempts.some((attempt) => attempt.status === 401)) {
    return "";
  }
  const parts = outcome.attempts.map((attempt) =>
    attempt.status === 0
      ? `${attempt.label}: request failed`
      : `${attempt.label}: HTTP ${attempt.status}`,
  );
  return `\n[credentials: ${parts.join("; ")}]`;
}

function normalize(raw: string | undefined): string {
  const token = (raw ?? "").trim();
  return token.startsWith(TOKEN_PREFIX) ? token : "";
}

async function discard(res: Response): Promise<void> {
  try {
    await res.body?.cancel();
  } catch {}
}
