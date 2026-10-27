---
title: Attachments & artifacts
description: Upload files to a task and consume the artifacts an agent publishes.
---

# Attachments & artifacts

CherryPi distinguishes **attachments** (files you give the task) from
**artifacts** (files the task produces). Both are stored in the agency's S3
bucket and served through the agency, so the bucket never needs to be public.

## Attachments

Add files by dragging them into the console, pasting a screenshot, or using the
paperclip in the composer. Limits are controlled by the agency:

| Setting | Default |
| --- | --- |
| Max size per file | 25 MiB (`CHERRYPI_ATTACHMENT_MAX_BYTES`) |
| Files per message | 10 (`CHERRYPI_ATTACHMENT_MAX_PER_MESSAGE`) |

When a task starts, the agent pulls the task's attachments into its workspace
under `attachments/`. Images are also passed to Pi as image content, so the
model can see them directly. Text files and documents are passed by path.

Attachments stay available in the **Attachments** tab of the task's details
drawer.

## Artifacts

An agent publishes an artifact with the bundled `setTaskArtifacts` tool. An
artifact has:

- a name and optional description;
- one or more parts, each a file, text or structured data;
- an optional media type.

Artifacts appear in the task's **Artifacts** tab. File parts can be previewed or
downloaded. They are the durable output of a task: markdown reports, screenshots,
generated code, test logs.

Artifacts are attached to the task, not to a message, so they stay visible after
the conversation scrolls.

## Links

The `setTaskUrls` tool publishes links separately from artifacts. They appear in
the **Links** tab and on the task list, for example a pull request or a deployed
preview.

## Storage operations

- Uploads are proxied by the agency with the user's session or token.
- Agents download and upload with their own personal access token.
- Deleting a task deletes its attachments and artifacts (subject to bucket
  lifecycle rules).
- Back up the bucket together with the database. The database stores metadata
  and object references, not file bodies.

## Retention

Attachment and artifact retention follows the bucket's lifecycle policy and the
agency's task retention. Configure versioning and an expiry policy on the bucket
if you handle large outputs.

## Next

- [Tasks](/using/tasks)
- [Install the agency](/agency/install)
- [Configure the agency](/agency/configuration)
