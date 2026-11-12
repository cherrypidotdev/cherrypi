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

import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";

const MODEL_REQUEST = 1;
const RETRY = 2;
const COMPACTION = 3;

export default function (pi: ExtensionAPI) {
  let requestStartedAt = 0;
  let requestModel = "";
  let requestProvider = "";

  let retryStartedAt = 0;
  let retryError = "";
  let requestAttempt = 0;

  let compactionStartedAt = 0;
  let compactionReason = "";

  let thinkingStartedAt = 0;
  let thinkingChars = 0;

  function record(
    kind: number,
    startedAt: number,
    durationMs: number,
    extra: Record<string, unknown>,
  ) {
    if (startedAt <= 0 || durationMs <= 0) return;
    pi.appendEntry("cherrypi_model_wait", { kind, durationMs, ...extra });
  }

  function closeRequest(answered: boolean) {
    if (requestStartedAt <= 0) return;
    if (answered) requestAttempt = 0;
    record(MODEL_REQUEST, requestStartedAt, Date.now() - requestStartedAt, {
      model: requestModel,
      provider: requestProvider,
    });
    requestStartedAt = 0;
  }

  function closeRetry() {
    if (retryStartedAt <= 0) return;
    record(RETRY, retryStartedAt, Date.now() - retryStartedAt, {
      model: requestModel,
      provider: requestProvider,
      error: retryError,
      attempt: requestAttempt,
    });
    retryStartedAt = 0;
    retryError = "";
  }

  function closeCompaction() {
    if (compactionStartedAt <= 0) return;
    record(COMPACTION, compactionStartedAt, Date.now() - compactionStartedAt, {
      model: requestModel,
      provider: requestProvider,
      reason: compactionReason,
    });
    compactionStartedAt = 0;
    compactionReason = "";
  }

  function closeThinking(content?: string) {
    if (thinkingStartedAt <= 0) return;
    const chars = Array.from(content ?? "").length || thinkingChars;
    pi.appendEntry("cherrypi_thinking_span", {
      durationMs: Date.now() - thinkingStartedAt,
      chars,
    });
    thinkingStartedAt = 0;
    thinkingChars = 0;
  }

  function resetRun() {
    requestStartedAt = 0;
    requestModel = "";
    requestProvider = "";
    retryStartedAt = 0;
    retryError = "";
    requestAttempt = 0;
    compactionStartedAt = 0;
    compactionReason = "";
  }

  pi.on("session_start", async (_event, ctx) => {
    resetRun();
    if (ctx.model) {
      requestModel = ctx.model.id;
      requestProvider = ctx.model.provider;
    }
  });

  pi.on("model_select", async (event) => {
    if (event.model) {
      requestModel = event.model.id;
      requestProvider = event.model.provider;
    }
  });

  pi.on("turn_start", async (_event, ctx) => {
    closeRetry();
    closeRequest(false);
    requestStartedAt = Date.now();
    requestModel = ctx.model?.id ?? requestModel;
    requestProvider = ctx.model?.provider ?? requestProvider;
  });

  pi.on("message_end", async (event) => {
    const message = event.message;
    if (message?.role !== "assistant") return;
    if (message.stopReason !== "error") return;
    closeRequest(false);
    if (retryStartedAt <= 0) {
      requestAttempt += 1;
      retryError = message.errorMessage ?? "";
      retryStartedAt = Date.now();
    }
  });

  pi.on("message_start", async (event) => {
    if (event.message?.role !== "assistant") return;
    closeRetry();
  });

  pi.on("message_update", async (event) => {
    const update = event.assistantMessageEvent;
    const type = update?.type;
    if (type === "start" || type === "done" || type === "error") return;
    closeRequest(true);

    switch (type) {
      case "thinking_start":
        thinkingStartedAt = Date.now();
        thinkingChars = 0;
        break;
      case "thinking_delta":
        thinkingStartedAt ||= Date.now();
        thinkingChars += Array.from(update?.delta ?? "").length;
        break;
      case "thinking_end":
        closeThinking(update?.content);
        break;
    }
  });

  pi.on("session_before_compact", async (event) => {
    if (compactionStartedAt > 0) return;
    compactionStartedAt = Date.now();
    compactionReason = event.reason ?? "";
  });

  pi.on("session_compact", async () => {
    closeCompaction();
  });

  pi.on("session_compact_failed", async () => {
    closeCompaction();
  });

  pi.on("agent_settled", async () => {
    closeRetry();
    closeRequest(false);
    closeCompaction();
    closeThinking();
    resetRun();
  });
}
