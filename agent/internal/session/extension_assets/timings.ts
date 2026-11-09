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

  let runFirstToken = false;

  let retryStartedAt = 0;
  let retryError = "";
  let requestAttempt = 0;

  let compactionStartedAt = 0;
  let compactionReason = "";

  function record(
    kind: number,
    startedAt: number,
    durationMs: number,
    extra: Record<string, unknown>,
  ) {
    if (startedAt <= 0 || durationMs <= 0) return;
    pi.appendEntry("cherrypi_model_wait", { kind, durationMs, ...extra });
  }

  function closeRequest(produced: boolean) {
    if (requestStartedAt <= 0) return;
    const firstToken = produced && !runFirstToken;
    if (firstToken) runFirstToken = true;

    if (produced) requestAttempt = 0;
    record(MODEL_REQUEST, requestStartedAt, Date.now() - requestStartedAt, {
      model: requestModel,
      firstToken,
    });
    requestStartedAt = 0;
    requestModel = "";
  }

  function closeRetry() {
    if (retryStartedAt <= 0) return;
    record(RETRY, retryStartedAt, Date.now() - retryStartedAt, {
      error: retryError,
      attempt: requestAttempt,
    });
    retryStartedAt = 0;
    retryError = "";
  }

  function closeCompaction() {
    if (compactionStartedAt <= 0) return;
    record(COMPACTION, compactionStartedAt, Date.now() - compactionStartedAt, {
      reason: compactionReason,
    });
    compactionStartedAt = 0;
    compactionReason = "";
  }

  function resetRun() {
    requestStartedAt = 0;
    requestModel = "";
    runFirstToken = false;
    retryStartedAt = 0;
    retryError = "";
    requestAttempt = 0;
    compactionStartedAt = 0;
    compactionReason = "";
  }

  pi.on("session_start", async (_event, ctx) => {
    resetRun();
    if (ctx.model) requestModel = ctx.model.id;
  });

  pi.on("model_select", async (event) => {
    if (event.model) requestModel = event.model.id;
  });

  pi.on("turn_start", async (_event, ctx) => {
    closeRetry();
    closeRequest(false);
    requestStartedAt = Date.now();
    requestModel = ctx.model?.id ?? requestModel;
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
    const type = event.assistantMessageEvent?.type;
    if (type === "start" || type === "done" || type === "error") return;
    closeRequest(true);
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
    resetRun();
  });
}
