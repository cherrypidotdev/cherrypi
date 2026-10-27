#!/usr/bin/env python3
"""Seed a throwaway CherryPi agency for the documentation screenshots.

Run it against a `-tags dev` agency backed by a fresh SQLite database:

    CHERRYPI_AGENCY_DIR=/tmp/chpi-demo/agency \\
    CHERRYPI_ADMIN_PASSWORD=demodemo \\
    CHERRYPI_AGENCY_SEED_AGENT=builder:devagent \\
    CHERRYPI_FRONTEND_PROXY_URL=http://127.0.0.1:5173 \\
    go run -tags dev ./cmd/agency

    python3 docs-site/scripts/seed-demo.py --url http://127.0.0.1:8080 \\
      --password demodemo --db /tmp/chpi-demo/agency/agency.sqlite3

The script creates teams, projects, agents and a set of tasks in different
states. When --db is given it also writes representative Pi session history and
task messages directly to SQLite, so the task console has something to show
without a live Pi session. Do not run it against a real agency.
"""

from __future__ import annotations

import argparse
import http.cookiejar
import json
import sqlite3
import sys
import time
import urllib.error
import urllib.request
import uuid


class Agency:
    def __init__(self, base_url: str, username: str, password: str) -> None:
        self.base_url = base_url.rstrip("/")
        self.cookies = http.cookiejar.CookieJar()
        self.opener = urllib.request.build_opener(
            urllib.request.HTTPCookieProcessor(self.cookies)
        )
        self.csrf = ""
        self.login(username, password)

    def raw(self, method: str, path: str, payload: dict | None, grpc: bool) -> dict:
        prefix = "/api/grpc" if grpc else ""
        url = f"{self.base_url}{prefix}{path}"
        data = None
        headers = {"Accept": "application/json"}
        if payload is not None:
            data = json.dumps(payload).encode()
            headers["Content-Type"] = "application/json"
        if method not in ("GET", "HEAD") and self.csrf:
            headers["X-CSRF-Token"] = self.csrf
        request = urllib.request.Request(
            url, data=data, headers=headers, method=method
        )
        try:
            with self.opener.open(request) as response:
                body = response.read()
                return json.loads(body) if body else {}
        except urllib.error.HTTPError as error:
            detail = error.read().decode(errors="replace")
            raise SystemExit(f"{method} {url} -> {error.code}: {detail}") from error

    def rest(self, method: str, path: str, payload: dict | None = None) -> dict:
        return self.raw(method, path, payload, grpc=False)

    def rpc(self, method: str, path: str, payload: dict | None = None) -> dict:
        return self.raw(method, path, payload, grpc=True)

    def login(self, username: str, password: str) -> None:
        result = self.rest(
            "POST",
            "/api/rest/auth/login",
            {"identifier": username, "password": password},
        )
        self.csrf = result.get("csrf_token", "")


def ensure_team(agency: Agency, name: str, description: str) -> dict:
    teams = agency.rest("GET", "/api/rest/teams").get("teams", [])
    for team in teams:
        if team.get("name") == name:
            return team
    return agency.rest(
        "POST", "/api/rest/teams", {"name": name, "description": description}
    )["team"]


def ensure_project(agency: Agency, name: str, slug: str, prefix: str, description: str) -> dict:
    projects = agency.rpc("POST", "/agency.v1.ProjectService/ListProjects", {}).get(
        "projects", []
    )
    for project in projects:
        if project.get("slug") == slug:
            return project
    return agency.rpc(
        "POST",
        "/agency.v1.ProjectService/CreateProject",
        {"name": name, "slug": slug, "prefix": prefix, "description": description},
    )["project"]


def attach_team(agency: Agency, project_id: str, team_id: int, role: str = "owner") -> None:
    agency.rest(
        "PUT",
        f"/api/rest/projects/{project_id}/teams/{team_id}",
        {"role": role},
    )


def ensure_agent(agency: Agency, username: str, display_name: str, description: str) -> dict:
    agents = agency.rpc("POST", "/agency.v1.AgentService/ListAgents", {}).get(
        "agents", []
    )
    for agent in agents:
        if agent.get("username") == username:
            return agent
    return agency.rpc(
        "POST",
        "/agency.v1.AgentService/CreateAgent",
        {
            "username": username,
            "displayName": display_name,
            "description": description,
            "prompt": "Prefer small, reviewable diffs. Run the test suite before finishing.",
        },
    )["agent"]


TASK_STATE_ALIASES = {
    "draft": "TASK_STATE_DRAFT",
    "backlog": "TASK_STATE_BACKLOG",
    "queued": "TASK_STATE_QUEUED",
}


def add_agent(agency: Agency, project_id: str, agent_id: str) -> None:
    agency.rpc(
        "POST",
        "/agency.v1.ProjectService/AddProjectAgent",
        {"projectId": project_id, "agentId": agent_id},
    )


def create_task(
    agency: Agency,
    project_id: str,
    team_id: int,
    agent_actor_id: str,
    name: str,
    prompt: str,
    state: str,
    priority: str,
    size: str,
) -> dict:
    return agency.rpc(
        "POST",
        "/agency.v1.TaskService/CreateTask",
        {
            "projectId": project_id,
            "assigneeActorId": agent_actor_id,
            "teamIds": [team_id],
            "name": name,
            "prompt": prompt,
            "state": TASK_STATE_ALIASES.get(state, "TASK_STATE_QUEUED"),
            "priority": priority,
            "size": size,
        },
    )["task"]


def entry(entry_id: str, parent: str | None, timestamp: str, message: dict) -> str:
    return json.dumps(
        {
            "type": "message",
            "id": entry_id,
            "parentId": parent,
            "timestamp": timestamp,
            "message": message,
        }
    )


def user_message(text: str, timestamp: str, entry_id: str, parent: str | None) -> str:
    return entry(
        entry_id,
        parent,
        timestamp,
        {"role": "user", "content": [{"type": "text", "text": text}]},
    )


def assistant_text(text: str, timestamp: str, entry_id: str, parent: str) -> str:
    return entry(
        entry_id,
        parent,
        timestamp,
        {"role": "assistant", "content": [{"type": "text", "text": text}]},
    )


def assistant_tool(call_id: str, name: str, arguments: dict, timestamp: str, entry_id: str, parent: str) -> str:
    return entry(
        entry_id,
        parent,
        timestamp,
        {
            "role": "assistant",
            "content": [
                {
                    "type": "toolCall",
                    "id": call_id,
                    "name": name,
                    "arguments": arguments,
                }
            ],
        },
    )


def tool_result(
    call_id: str,
    name: str,
    text: str,
    timestamp: str,
    entry_id: str,
    parent: str,
) -> str:
    return entry(
        entry_id,
        parent,
        timestamp,
        {
            "role": "toolResult",
            "toolCallId": call_id,
            "toolName": name,
            "content": [{"type": "text", "text": text}],
            "isError": False,
        },
    )


def history_for(task_name: str) -> list[tuple[str, str, str, str, str]]:
    """Return (entry_id, parent_id, type, raw, created_at) rows for a demo task."""
    now = int(time.time() * 1000)
    rows: list[tuple[str, str, str, str, str]] = []

    def add(entry_id: str, parent: str, raw: str, minutes_ago: int) -> None:
        rows.append(
            (
                entry_id,
                parent,
                "message",
                raw,
                str(now - minutes_ago * 60_000),
            )
        )

    if "onboarding" in task_name.lower():
        add(
            "u1",
            "",
            user_message(
                "The onboarding flow drops invitees on a blank page after they accept an invitation. Reproduce it, find the cause and fix it. Add a regression test.",
                "2026-10-26T09:02:00Z",
                "u1",
                None,
            ),
            42,
        )
        add(
            "a1",
            "u1",
            assistant_text(
                "I reproduced it locally. The accept-invitation route redirects to `/tasks` before the session cookie is committed, so the first request is unauthenticated and the SPA renders an empty shell. I'll check the redirect and the cookie flags.",
                "2026-10-26T09:02:08Z",
                "a1",
                "u1",
            ),
            41,
        )
        add(
            "a2",
            "a1",
            assistant_tool(
                "call-1",
                "bash",
                {"command": "rg -n \"redirect.*tasks|accept.*invitation\" src/views src/lib"},
                "2026-10-26T09:02:12Z",
                "a2",
                "a1",
            ),
            40,
        )
        add(
            "r1",
            "a2",
            tool_result(
                "call-1",
                "bash",
                "src/views/InviteView.vue:84:  await router.replace(redirectTarget());\nsrc/lib/account.ts:287:  await reloadSession();",
                "2026-10-26T09:02:13Z",
                "r1",
                "a2",
            ),
            39,
        )
        add(
            "a3",
            "r1",
            assistant_text(
                "Found it: `InviteView` calls `resetSession()` then routes immediately, before `reloadSession()` resolves. The fix is to await the session reload and only redirect when it succeeds; otherwise show the sign-in form with a notice. I've added a Vitest regression that asserts the redirect happens after the session resolves.",
                "2026-10-26T09:03:40Z",
                "a3",
                "r1",
            ),
            30,
        )
        add(
            "a4",
            "a3",
            assistant_tool(
                "call-2",
                "bash",
                {"command": "pnpm --filter dashboard test --run InviteView"},
                "2026-10-26T09:03:48Z",
                "a4",
                "a3",
            ),
            29,
        )
        add(
            "r2",
            "a4",
            tool_result(
                "call-2",
                "bash",
                "✓ src/views/InviteView.spec.ts (9 tests)\n\nTest Files  1 passed (1)\nTests  9 passed (9)",
                "2026-10-26T09:03:55Z",
                "r2",
                "a4",
            ),
            28,
        )
        add(
            "a5",
            "r2",
            assistant_text(
                "Done. The invitation flow now waits for the session before navigating, and the regression covers the blank-page case.",
                "2026-10-26T09:04:00Z",
                "a5",
                "r2",
            ),
            27,
        )
    else:
        add(
            "u1",
            "",
            user_message(
                f"Work on: {task_name}.",
                "2026-10-26T10:10:00Z",
                "u1",
                None,
            ),
            15,
        )
        add(
            "a1",
            "u1",
            assistant_text(
                "I'll start by reading the relevant files and the current tests, then make the smallest change that covers the request.",
                "2026-10-26T10:10:04Z",
                "a1",
                "u1",
            ),
            14,
        )
    return rows


def seed_history(
    db_path: str,
    task_id: str,
    task_name: str,
    agent_id: int,
    state: str,
    error: str = "",
) -> None:
    connection = sqlite3.connect(db_path)
    try:
        now = int(time.time() * 1000)
        started = now - 45 * 60_000
        settled = now - 25 * 60_000
        connection.execute(
            """
            UPDATE tasks
               SET state = ?, session_id = ?, session_agent_id = ?,
                   started_at = ?, settled_at = ?, updated_at = ?,
                   state_updated_at = ?, error = ?
             WHERE id = ?
            """,
            (
                state,
                f"demo-session-{task_id}",
                agent_id,
                started,
                settled if state in ("settled", "completed", "failed") else None,
                now,
                now,
                error,
                task_id,
            ),
        )
        connection.execute(
            """
            INSERT INTO task_messages (id, task_rowid, text, created_at, actor_id)
            SELECT ?, rowid, prompt, ?, creator_actor_id FROM tasks WHERE id = ?
            """,
            (str(uuid.uuid4()), started, task_id),
        )
        connection.executemany(
            """
            INSERT OR REPLACE INTO task_session_entries
                (task_id, seq, entry_id, parent_id, type, raw, created_at)
            VALUES (?, ?, ?, ?, ?, ?, ?)
            """,
            [
                (task_id, index, entry_id, parent, kind, raw, int(created_at))
                for index, (entry_id, parent, kind, raw, created_at) in enumerate(
                    history_for(task_name), start=1
                )
            ],
        )
        connection.commit()
    finally:
        connection.close()


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--url", default="http://127.0.0.1:8080")
    parser.add_argument("--user", default="admin")
    parser.add_argument("--password", default="demodemo")
    parser.add_argument(
        "--db",
        help="SQLite database path; enables task history and state seeding",
    )
    args = parser.parse_args()

    agency = Agency(args.url, args.user, args.password)

    team = ensure_team(
        agency,
        "Website Crew",
        "Design, frontend and release engineering for the public site.",
    )
    platform = ensure_team(
        agency,
        "Platform",
        "Shared infrastructure, CI and internal developer tooling.",
    )

    projects = [
        ensure_project(
            agency,
            "Website Relaunch",
            "website-relaunch",
            "WEB",
            "The new marketing site: onboarding, pricing and a faster docs experience.",
        ),
        ensure_project(
            agency,
            "Mobile App",
            "mobile-app",
            "APP",
            "React Native client for iOS and Android.",
        ),
        ensure_project(
            agency,
            "Infrastructure",
            "infrastructure",
            "INFRA",
            "CI runners, deployments and observability.",
        ),
    ]
    for project in projects:
        attach_team(agency, project["id"], team["id"], "owner")
    attach_team(agency, projects[2]["id"], platform["id"], "member")

    builders = [
        ensure_agent(agency, "builder", "Builder", "Linux build box with the full toolchain."),
        ensure_agent(agency, "design-runner", "Design Runner", "Linux runner with Playwright and ImageMagick."),
        ensure_agent(agency, "ops-runner", "Ops Runner", "Production operations and incident work."),
    ]
    for project in projects:
        for agent in builders:
            add_agent(agency, project["id"], str(agent["id"]))

    specs = [
        (
            projects[0],
            builders[0],
            "Fix the onboarding flow",
            "The onboarding flow drops invitees on a blank page after they accept an invitation. Reproduce it, find the cause and fix it. Add a regression test.",
            "settled",
            "TASK_PRIORITY_HIGH",
            "TASK_SIZE_MEDIUM",
        ),
        (
            projects[0],
            builders[1],
            "Add dark mode to settings",
            "Add a dark-mode toggle to Settings. Persist the choice per account, respect the OS preference by default, and make sure every settings panel has a readable dark palette.",
            "running",
            "TASK_PRIORITY_NORMAL",
            "TASK_SIZE_BIG",
        ),
        (
            projects[0],
            builders[0],
            "Investigate flaky login test",
            "The login integration test fails roughly once in twenty runs. Find the race, fix it, and prove the fix with thirty consecutive green runs.",
            "failed",
            "TASK_PRIORITY_HIGH",
            "TASK_SIZE_SMALL",
        ),
        (
            projects[0],
            builders[2],
            "Write webhook API docs",
            "Document the agency webhook payloads, signing algorithm and retry policy. Add runnable curl examples for task.created, task.settled and agent.connected.",
            "queued",
            "TASK_PRIORITY_NORMAL",
            "TASK_SIZE_MEDIUM",
        ),
        (
            projects[1],
            builders[1],
            "Redesign the task filters",
            "Rework the task filter bar for mobile: project, agent and status filters should fit on one row and remain usable with the keyboard.",
            "backlog",
            "TASK_PRIORITY_LOW",
            "TASK_SIZE_MEDIUM",
        ),
        (
            projects[2],
            builders[2],
            "Release v0.2.0",
            "Cut v0.2.0 from main: update the changelog, verify the public snapshot and publish the release binaries and images.",
            "completed",
            "TASK_PRIORITY_CRITICAL",
            "TASK_SIZE_BIG",
        ),
        (
            projects[2],
            builders[0],
            "Update dependency images",
            "Rebuild the CI builder image with Node 22.21.1 and Go 1.26.10. Run the full suite inside the new image before publishing it.",
            "queued",
            "TASK_PRIORITY_NORMAL",
            "TASK_SIZE_SMALL",
        ),
        (
            projects[1],
            builders[0],
            "Split the billing service",
            "Extract billing into its own service with a typed client. Keep the existing API contracts and migrate the data behind a feature flag.",
            "draft",
            "TASK_PRIORITY_HIGH",
            "TASK_SIZE_EPIC",
        ),
    ]

    created = []
    for project, agent, name, prompt, state, priority, size in specs:
        task = create_task(
            agency,
            project["id"],
            team["id"],
            str(agent["actorId"]),
            name,
            prompt,
            state,
            priority,
            size,
        )
        created.append((task, project, agent, name, state))
        print(f"created {task['id']} {name} ({state})")

    if args.db:
        for task, project, agent, name, state in created:
            seed_history(
                args.db,
                task["id"],
                name,
                int(agent["id"]),
                state,
                error=(
                    "Pi exited with status 1: the model provider returned 429 "
                    "(rate limited). Retry after the provider resets."
                    if state == "failed"
                    else ""
                ),
            )
        print(f"seeded history in {args.db}")

    print("\nScreenshot target:")
    print(f"  task    {created[0][0]['id']}")
    print(f"  project {projects[0]['slug']}")
    print(f"  agent   {args.user and builders[0]['username']}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
