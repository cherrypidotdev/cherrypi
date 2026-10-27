---
title: Terminals
description: Open root shells on an agent, share them with observers, and use the agency's own terminal views.
---

# Terminals

CherryPi relays real PTYs from the agent to the browser over WebSockets. A
terminal is a root shell on the agent's machine — treat it as full access to
that host.

## Opening a terminal

1. Open an agent and choose the **Terminals** tab.
2. Press **New terminal**. The first viewer becomes the **controller**.
3. Type in the terminal as usual. The PTY is resized to the browser.

Owners and administrators can open terminals. Other team members can watch
shared terminals and take control if the controller releases it or disconnects.

## Control and observation

A terminal can have several consumers:

- exactly one **controller** sends keystrokes and resize events;
- every other consumer is a read-only **observer** of the same output;
- any observer can request control.

The terminal survives a browser reload. It is closed when the shell exits or
when the agent's terminal is explicitly closed. A WebSocket close is not an
exit; the agency reports the shell's exit status separately.

Output is coalesced into frames (up to 32 KiB, with a short flush interval), and
a slow browser applies backpressure to the PTY instead of losing output.

## Where terminals run

The shell runs as the agent process's user, in the agent's environment. It does
not automatically enter a task workspace. To work on a task, change into

```sh
$CHERRYPI_DIR/sessions/<task id>
```

The shell is configured by `SHELL_BIN`, then `$SHELL`, then `/bin/bash`.

## Terminal sessions in the agency

Root terminals opened from the agency's own **Terminals** area run on the
agency host and are useful for operator work. They use the same relay and
permission model.

## Security

- A terminal bypasses project permissions: it runs as the agent user on the
  agent host.
- Restrict agent ownership. Owners can open root terminals.
- Run the agent as a dedicated, least-privileged user where possible.
- The terminal is only reachable through the authenticated tunnel, but the
  agency account that controls it is the real boundary.

## Troubleshooting

| Symptom | Check |
| --- | --- |
| Terminal connects then closes | `SHELL_BIN`/`$SHELL` is missing or not executable. |
| No output, input works | The reverse proxy is buffering; disable buffering/read timeouts. |
| Output freezes | A slow consumer; the agent applies backpressure. Close idle terminal tabs. |
| "Only owners may open terminals" | Ask the owner or an administrator for access. |

See [Troubleshooting](/advanced/troubleshooting).
