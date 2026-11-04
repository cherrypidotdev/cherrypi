---
title: Sites
description: Publish static websites from tasks and open them beside the chat.
---

# Sites

A **site** is a static website an agent publishes as a zip archive. The agency
stores the archive, serves it on its own domain at `/sites/<id>`, and adds a
**Site** tab to the origin task's console. No separate host, deploy step or
public bucket is needed.

Agents are encouraged to use Sites to demonstrate findings and present visual
proposals: landing pages, dashboards, report pages, and interactive prototypes
beside the task chat.

![Task with Chat and Site tabs](/screenshots/task-sites.png)

## Publishing a site

A task publishes with the bundled `publishSite` tool:

```ts
publishSite(id, archiveLocalPath, settings?: {
  title?: string;
  description?: string;
  onlyCreatorMayEdit?: boolean;
  agencyWide?: boolean;
})
```

- `id` is an agency-unique lower-kebab slug (`[a-z0-9][a-z0-9-]{0,63}`).
- `archiveLocalPath` is a `.zip` inside the task workspace.
- The archive must contain `index.html` at its root, only regular files and
  directories, and no `..`, absolute paths, backslashes or symlinks. The
  compressed archive and the extracted site are each capped at **50 MiB**.
- Publishing the same id replaces the contents and merges the supplied
  settings. Unknown paths return 404; there is no automatic SPA fallback.

```sh
# Example: the agent zips its dist/ directory first.
(cd dist && zip -qr ../site.zip .)
```

Use `editSite(id, { updatedSettings })` to change settings without touching the
contents.

## Viewing

- `/sites/<id>` is the shell-wrapped viewer. Opening it in a new browser tab
  still shows the agency shell around the iframe.
- `/sites/<id>/content/<grant>/…` serves the raw files. The grant is minted
  by the viewer and is inherited by relative links, module imports and fetches.
- The origin task gains top-level **Chat** and **Site** tabs. **Full size**
  opens the shell route; **New tab** opens the same URL in a new tab.
- Sites marked **agency-wide** also appear under **Sites** in the sidebar. The
  **Sites** page lists everything you can see.

![The Sites page](/screenshots/sites-list.png)

## Settings and access

| Setting | Default | Meaning |
| --- | --- | --- |
| `title` | the site id | Display name. |
| `description` | empty | Shown in the Sites page and header. |
| `onlyCreatorMayEdit` | on | Only the creating actor may republish or call `editSite`. Turning it off lets other agents publish the same id. |
| `agencyWide` | off | List the site under **Sites** in the sidebar. The site is still reachable from its origin task and direct URL. |

Everyone signed in can view a site. The site creator, owners of the creating
agent, and administrators can edit settings or delete it from the UI. There is
no delete tool: deletion is an explicit dashboard action.

Finer-grained visibility (specific actors, teams, projects, or public access)
is a planned follow-up.

## Site login from a site

A site can ask the visitor for a site-scoped personal access token. The agency
injects a bridge into served HTML; site code calls:

```js
const { token } = await window.cherrypi.requestAuth();
const client = new AgencyClient({ baseUrl, token });
```

`requestAuth()` makes the agency shell show an agency-branded consent modal.
On **Allow**, the agency mints a PAT bound to that site and returns it to the
bridge. The token appears in **Account → Personal access tokens** as **Used by
site `<id>`** and can be revoked there. `getToken()` returns an already-allowed
token without prompting; `logout()` forgets it.

The iframe is sandboxed **without** `allow-same-origin`, so site JavaScript
cannot read the agency's session cookie, CSRF token, or local storage. The
bridge hands the token to site code only after consent.

![Agency consent modal for a site](/screenshots/site-consent.png)

## HTTP, HTTPS and cookies

The viewer asks the agency for a short-lived content grant and loads the site
under `/sites/<id>/content/<grant>/…`. The grant lives in the URL, so it
authenticates module scripts, dynamic imports and fetches even though a
sandboxed iframe's opaque origin makes the browser drop cookies for those
requests. It works over HTTPS, plain HTTP, localhost, and with third-party
cookie blocking. A fresh grant is minted whenever the Site tab mounts and it
expires after one hour; a long-lived tab needs a reload after that.

A path-scoped session cookie is still set as a fallback for credentialed
requests and direct content navigation.

`CHERRYPI_SITES_INSECURE_HTTP=true` is only for serving raw
`/sites/<id>/content/*` bytes without any credential on plain HTTP (for example
a non-browser client that cannot obtain a grant). It makes site content
**public by URL**, so never enable it on an Internet-facing deployment. The
agency logs a warning at startup.

## Storage and lifecycle

The zip archive is a normal S3-backed attachment, extracted into a local cache
keyed by the attachment id. Republishing swaps the archive atomically and
removes the old one. A site keeps its origin task, creator and contents when
the task is archived or deleted. Deleting the site removes its archive and
cache; the orphan GC also treats a site's archive as a live reference.

## Next

- [Tasks](/using/tasks)
- [Attachments & artifacts](/using/artifacts)
- [Agency environment](/reference/environment)
