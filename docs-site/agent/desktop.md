---
title: Desktop access
description: Expose an existing X display from an agent to the browser with x11vnc and noVNC.
---

# Desktop access

CherryPi can stream a running X display to the browser. This is useful for GUI
tools that a task needs: a browser, an IDE, a design tool, or a debugging
session on the worker host.

The agent does **not** start a desktop. It attaches `x11vnc` to a display that
is already running, on the first browser attach. If there is no display, the
feature is simply unavailable — the agent page explains why.

## Requirements

- An X display on the agent host (`:0`, `:1`, …).
- `x11vnc`, `xauth` and X utilities. The published agent image includes them.
- Access to the X socket and the authority file.
- A browser on the agency side. NoVNC renders the RFB stream.

## Host X session

On a normal Linux desktop, the X session is already running. Run the agent as a
user that can access it:

```sh
cherrypi-agent \
  --agency-url https://cherrypi.example.com \
  --token pat_… \
  --display :0 \
  --xauthority "$HOME/.Xauthority"
```

The equivalent environment variables are `DISPLAY` and `XAUTHORITY`.

If you use a headless machine, start a virtual display first, for example with
`Xvfb :99 -screen 0 1920x1080x24`, then set `DISPLAY=:99`. CherryPi does not
manage Xvfb for you.

## Docker

Mount the X socket and the authority file into the container:

```sh
docker run -d --name cherrypi-agent \
  --restart unless-stopped \
  -v /srv/cherrypi/agent:/root/.cherrypi \
  -v /srv/cherrypi/pi:/root/.pi/agent \
  -v /tmp/.X11-unix:/tmp/.X11-unix \
  -v "$HOME/.Xauthority:/root/.Xauthority:ro" \
  -e DISPLAY=:0 \
  -e XAUTHORITY=/root/.Xauthority \
  -e CHERRYPI_AGENCY_URL=https://cherrypi.example.com \
  -e CHERRYPI_AGENT_TOKEN=pat_… \
  ghcr.io/cherrypidotdev/cherrypi-agent:latest
```

On macOS, X11 is not available by default; run the agent on Linux or a Linux
VM.

## Using it

Open the agent page and choose **Desktop**. The first attach starts `x11vnc`;
later viewers join the same stream. The stream is view-only for people who are
not owners of the agent; owners get keyboard and pointer control.

Close the tab to detach. The agent does not keep a VNC server alive after the
last viewer leaves.

## Security

- The desktop is reachable only through the authenticated agency tunnel.
- Treat the display as the agent user's session: every window and every
  credential visible there is exposed to viewers.
- Prefer a dedicated user and display for agent work.
- Do not mount your personal `$HOME` into the agent container just to share an
  X session.

## Troubleshooting

| Symptom | Check |
| --- | --- |
| Desktop button explains it is disabled | `DISPLAY`/`--display` is unset on the agent. |
| `cannot open display` in agent logs | The X socket or authority file is not mounted/readable. |
| Black screen | The display exists but no window manager/compositor is running. Start the session you want to show. |
| Frequent disconnects | The reverse proxy is timing out the WebSocket; disable its read timeout. |

See [Troubleshooting](/advanced/troubleshooting).
