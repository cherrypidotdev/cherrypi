# Security

## Reporting a vulnerability

Please report suspected vulnerabilities privately instead of opening a public
issue. Use either channel:

- [GitHub private vulnerability reporting](https://github.com/cherrypidotdev/cherrypi/security/advisories/new)
- Email [security@cherrypi.dev](mailto:security@cherrypi.dev)

Include the affected component, a reproduction, and the impact you expect. We
will acknowledge the report and coordinate a fix and disclosure.

## Trust model

CherryPi runs code from several sources with different levels of trust:

| Component | Trust |
| --- | --- |
| The agency and agent binaries | Trusted. They hold agency credentials and the tunnel. |
| Installed Lua plugins | Trusted once installed. An administrator reviews the bundle before installing it. |
| Plugin webhook callers | Untrusted. A plugin validates its own secret or signature. |
| Task prompts and model output | Untrusted. Tools and sandbox limits apply. |

### Plugin HTTP invocations

`/plugins/<id>/*` is served outside the agency's `/api` auth middleware so a
provider webhook can reach a plugin anonymously. The same path is also
reachable with an agency session cookie or personal access token, which the
handler resolves into `cherrypi.actor`.

`handle_http_request` receives **every** request header, including `cookie`,
`authorization`, `x-csrf-token`, and `proxy-authorization`. This is deliberate:
providers such as Gitea and GitHub authenticate a webhook with a static
`Authorization` header or an `X-Hub-Signature-256`-style HMAC, and filtering
those headers made header-based webhook authentication impossible.

The consequence is that a plugin's Lua code can read the agency credentials of
a signed-in caller and, if the manifest allowlist permits it, forward them to
an outbound host with `cherrypi.http.request`. Treat every installed plugin as
code that can act with the credentials of any user who visits its routes.

Mitigations in place:

- Installation is an administrator action; review the bundle's Lua before
  enabling it.
- Outbound calls are HTTPS-only, host-allowlisted, and refused to
  private/loopback addresses in production.
- Every invocation and host call is logged in the plugin panel.
- Disabling or uninstalling a plugin stops its handlers immediately.

Hardening of the header surface (re-filtering agency credentials while keeping
provider webhook authentication) is planned. Until then, plugin authors must
not log the `headers` table or forward it to an outbound host.

### Plugin bundles

`cherrypi-plugin pack`, `lint`, `test`, and `e2e` skip version-control metadata
(`.git`, `.hg`, `.svn`, and other VCS directories) and paths matched by the
tree's `.gitignore` files. This keeps a repository-root bundle from shipping
git history or ignored build output. A plain `zip -r` does not apply those
rules; use `cherrypi-plugin pack` when the bundle directory is a repository
root.

## Supported versions

Security fixes land on `main` and are released from there. Run a current
release; older tags are not patched.
