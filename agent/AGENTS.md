# Backend conventions

Backend-specific rules for `agent`. The repo-wide guidelines in
[`../AGENTS.md`](../AGENTS.md) still apply; this file only adds what is
specific to the Go agent.

## No HTTP listener, no local auth

- The agent has **no HTTP listener**: it serves its API (sessions, terminals,
  events) only over the agency [tunnel](../VOCABULARY.md#tunnel). There is no
  port, healthcheck, or basic-auth middleware.
- The agency authenticates the tunnel itself; the agent's request context
  carries the forwarded user as `X-CherryPi-User`
  (`internal/auth.WithUsername`/`Username`), used to attribute terminals.
- Do not add a listener or a local credential store back.

## Enums

- **Enums are integers.** Persisted or wire enums use integer values, never
  free-form strings.
- In protobuf, declare an `enum` (first value `*_UNSPECIFIED = 0`) and let the
  generated Go type carry the name.
- Outside protobuf, mirror the protobuf enum with a typed integer
  (`type Status int64`) whose constants hold the **same numbers**, so
  `agentv1.Status(s.Status)` is a safe cast. Never renumber once persisted.
- Store the integer in SQLite (`status INTEGER NOT NULL`), including in query
  literals (`WHERE status = 1 -- active`).

## Session workspaces

- A session's workspace defaults to **server-assigned**
  `$CHERRYPI_DIR/sessions/<session id>`.
- SDK callers may override it via `CreateSession`'s `workspace`; the server
  creates the directory if missing.
- A session clones whatever it needs into its own workspace and shares results
  as artifacts.

## Bundled Pi extensions

- The agent gives Pi tools through **bundled extensions**, never over the RPC
  protocol — `pi --mode rpc` has no tool-registration command. Extensions live
  in `internal/session/extension_assets`, are embedded with `//go:embed`, and
  are written out by `InstallExtensions` and passed as `pi --extension`.
- An extension must not start processes, sockets, or timers in its factory; it
  registers tools/commands and returns. It records state through
  `pi.appendEntry`, not local files, so the session's `handleRecord` can forward
  it to the agency over the event hub.
- The `setTaskUrls` extension is the reference for event-forwarding tools:
  the tool appends a `cherrypi_task_urls` custom entry,
  `handleEntryAppended` turns it into a `task_link` event, and the agency's
  events consumer stores the task's links. Keep the `customType` and the
  payload shape in sync across the two modules.
- Tools that need a response call the agency over HTTP with the agent's
  personal access token (`upload_attachment.ts`, `set_task_artifacts.ts`,
  `agent_tools.ts`). The agent passes `CHERRYPI_AGENCY_URL` and the token file
  path (`CHERRYPI_AGENT_TOKEN_FILE`) into `childEnv`; the extension reads the
  file on every call so a rotation needs no Pi respawn, and must treat a
  missing token as "not configured" rather than failing the run.

## Bundled Pi extensions

RPC mode cannot register a tool, so every CherryPi tool that is not built into
Pi lives in a TypeScript extension under
`agent/internal/session/extension_assets/`. `InstallExtensions` embeds each
file with `//go:embed`, writes it into `$CHERRYPI_DIR/extensions`, and the agent
passes the paths to Pi as `--extension`. Keep the list in `extensions.go` and
this one in sync:

- `set_task_urls.ts` — `setTaskUrls`
- `upload_attachment.ts` — `uploadAttachment`
- `set_task_artifacts.ts` — `setTaskArtifacts`
- `agent_tools.ts` — `listTasks`, `createTask`, `updateTask`, `listProjects`
- `ask_user_question.ts` — `ask_user_question` (the questionnaire wizard)

Extensions may import `typebox` and `@earendil-works/pi-coding-agent`; Pi's
loader aliases both. A file is re-written on every agent start, so an upgraded
agent ships the current extension.

Because command-line extensions load before package extensions and Pi keeps the
**first registration per tool name**, a bundled tool takes precedence over the
same-named tool from an installed package.
