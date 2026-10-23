# Agent compatibility

The agent and the agency speak a versioned protocol. An agent reports two
versions in the opening **hello** frame of its tunnel
(`tunnel.v1.TunnelService.Connect`):

- **build version** — the git-describe stamp baked into the image, injected at
  link time into `agent/internal/buildinfo.Version` from the `VERSION` build
  arg (see the `Makefile` and `agent/Dockerfile`). It lets an operator identify
  the exact binary.
- **protocol version** — a semver defined once in `shared/protocol.Version`
  and compiled into both binaries. Compatibility is judged on this, so a
  cosmetic agency release does not look like a required agent upgrade.

The agency compares the reported protocol version with its own and derives one
of three verdicts on `Agent.version_status`:

| Verdict | Rule | Behaviour |
| --- | --- | --- |
| `CURRENT` | same compatibility line, agent ≥ agency | accepted |
| `OUTDATED` | same compatibility line, agent < agency | accepted; yellow **Binary should be updated** |
| `INCOMPATIBLE` | different compatibility line, or no version reported | the tunnel is **refused**; red **Binary incompatible** |

An agent that predates the hello handshake reports nothing and is refused. The
refusal is still recorded (the registry keeps the reported versions while the
agent reports offline), so the Agents page can explain why the agent is not
connecting.

## Beta versioning (pre-1.0)

The protocol is in **beta** and versioned below 1.0. Semver's compatibility
unit is the **minor** while the major is 0, so the agency keys compatibility on
the minor (the *compatibility line*) until the protocol graduates to 1.0,
after which it keys on the major. With `shared/protocol.Version = v0.1.0`:

- `v0.1.0` and `v0.1.3` are the same line — a patch behind is `OUTDATED`.
- `v0.1.0` and `v0.2.0` are different lines — `INCOMPATIBLE`.

## Bumping the protocol version

`shared/protocol.Version` is the single source of truth; there is no runtime
knob, by design. Bump it in the PR that changes the surface, add the
`agent-bump` label, and add a row to the log below.

While the protocol is pre-1.0 (beta):

- **minor** — a breaking change (the tunnel handshake/framing or the agent-tool
  REST contract). Older agents land on a different compatibility line and are
  refused, so they must be rebuilt.
- **patch** — an additive, backward-compatible change. Older agents connect but
  are flagged `OUTDATED`.

After the protocol reaches 1.0 this flips back to the usual semver rule: major
for a break, minor for an addition, patch for fixes.

### Rollout order

Roll the **agents out before the agency**. The hello is an extra empty-`data`
frame on the stream, and an older agency's yamux plumbing skips empty frames,
so a new agent keeps working against an older agency. The reverse is not safe:
a new agency expects a hello and refuses an agent that does not send one.

## Version log

| Protocol | Agency change | Notes |
| --- | --- | --- |
| `v0.1.0` | Tunnel hello: agents report their build and protocol versions | Beta start; pre-hello binaries are refused. |
