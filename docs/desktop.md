# Desktop

The agent page's **RDP** button opens the agent machine's desktop in the
browser: an RFB (VNC) stream rendered by noVNC.

CherryPi **does not boot a desktop session**. The operator provides one — the
host user's Xfce/X11 session, or (in Docker) the host's X socket mounted into
the container — and the agent attaches `x11vnc` to it. Booting, geometry
control, and GPU acceleration are out of scope; the display is used exactly as
it is.

## How it works

```mermaid
sequenceDiagram
  participant B as Browser (noVNC)
  participant A as Agency
  participant G as Agent
  participant X as X server

  B->>A: DesktopService.GetDesktop(agent_id)
  A->>G: GetDesktop (over tunnel)
  G->>X: xdpyinfo $DISPLAY
  X-->>G: geometry
  G-->>A: display, geometry, ws_path
  A-->>B: available + ws_path
  B->>A: WS /api/agents/<id>/desktop/ws
  A->>G: relay with X-CherryPi-View-Only: 0|1
  G->>G: start x11vnc -localhost -rfbport <free> (once per mode)
  G->>X: attach to the display
  B<->>X: RFB bytes (read-write for owners, view-only for everyone else)
```

- **`agent.v1.DesktopService.GetDesktop`** reports whether an X display is
  reachable (and its geometry); it starts nothing.
- The RFB data plane is a WebSocket at `/api/desktop/ws` on the agent, relayed
  by the agency at `/api/agents/<agentId>/desktop/ws` — the same tunnel relay
  terminals use.
- The agency resolves the caller's agent ownership and selects the bridge: an
  owner (or administrator) gets the read-write `x11vnc`, everyone else gets a
  second `x11vnc -viewonly` instance. Input is refused by the VNC server, not
  merely hidden by noVNC.
- The agent starts each managed `x11vnc` on the first attach for its mode (and
  reuses it for later viewers), listening on `127.0.0.1` with no VNC password.
  The agency authenticates the browser, so the bridges are only reachable
  through it.
- One desktop is shared by every user credential of an agent (through the
  read-write or view-only bridge), unlike [terminals](../VOCABULARY.md#terminal),
  which are per-user.

## Keyboard

noVNC only receives key events while its canvas has focus. The RDP tab focuses
the canvas as soon as noVNC connects, so typing works without a click first;
clicking the screen again restores focus if a header control took it.

Browsers keep some combinations for themselves (`Ctrl+W`, `Ctrl+T`, `Ctrl+N`,
`Ctrl+P`, …) and never hand them to a page. The **Fullscreen** button in the
desktop header fills the view with the desktop and, on Chromium-based browsers,
holds a [Keyboard Lock](https://developer.mozilla.org/docs/Web/API/Keyboard/lock)
so those combinations reach the remote session too. While the lock is held a
**Keyboard captured · Esc to exit** badge sits at the bottom of the screen.
Press <kbd>Esc</kbd> or use **Exit fullscreen** to leave fullscreen; either
releases the lock. The lock is
only requested for owners — a view-only viewer cannot send input anyway.

Firefox and Safari do not implement Keyboard Lock, so fullscreen still works
there but reserved shortcuts stay with the browser. The API also requires a
secure context (HTTPS or `localhost`).

## Configuration

| Env | Meaning | Default |
| --- | --- | --- |
| `DISPLAY` | X display to attach to | — |
| `XAUTHORITY` | X authority file | — |

Both are standard X variables, and the agent binary also accepts `--display`
and `--xauthority`, which override the environment (see
[`agent/README.md`](../agent/README.md)). CherryPi adds no `CHERRYPI_DESKTOP_*`
variables.

Only Linux/X11 is supported. macOS (XQuartz) and Wayland are follow-ups. If no
display is configured or `x11vnc` is missing, the agent page explains why
instead of showing the canvas.

## Running the agent on the host

Run `cherrypi-agent` as the user whose session you want to expose. It inherits
`DISPLAY` (`:0`) and `XAUTHORITY`; nothing else is needed.

## Running the agent in Docker

The agent image already installs the bridge (`x11vnc`, `x11-utils` for
`xdpyinfo`, and `xauth`). Give the container the host's X socket and authority
file; the [`compose.desktop.yaml`](../compose.desktop.yaml) override does that:

```sh
CHERRYPI_HOST_XAUTHORITY="$HOME/.Xauthority" \
  docker compose -f compose.yaml -f compose.desktop.yaml up -d --build
```

No separate image is needed — the base `cherrypi-agent` image carries the
bridge.

On a headless host, start an X server and a session first (for example Xvfb +
Xfce, or `x11vnc`'s own `-create`), then point `DISPLAY` at it.
CherryPi deliberately does not do this for you.

## Scope and security

- The desktop is **read-write for agent owners** and **view-only for everyone
  else**. The agency relay sets a per-connection header and the agent starts a
  separate `x11vnc -viewonly` bridge, so a non-owner cannot inject pointer or
  keyboard input even with a modified client. The VNC bridges have no password
  because they never leave loopback.
- **Root terminals are owner-only.** Their WebSocket relay and every terminal
  RPC reject non-owners (administrators pass). The agent page disables the
  terminal tab/button for them.
- GPU acceleration (VirtualGL), session booting, geometry control, and
  screenshots are out of scope.
- If the CDN/tunnel path is unavailable, the view degrades to "Desktop
  unavailable" with the reason from the agent.
