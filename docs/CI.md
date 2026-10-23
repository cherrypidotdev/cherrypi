# Releases

This repository is public and generated: every commit on `main` is a release
snapshot, and a tag `vX.Y.Z` marks it. A tag push runs one workflow that builds
the release images and binaries and publishes the GitHub Release. There is no
test suite here; the development repository runs the tests, and only a green
tree becomes a release.

```mermaid
flowchart LR
  tag[tag vX.Y.Z pushed] --> images[build + push images<br/>ghcr.io/cherrypidotdev/cherrypi-{agency,agent}]
  tag --> binaries[build 8 binaries<br/>+ SHA256SUMS]
  images --> release[GitHub Release]
  binaries --> release
```

| Trigger | Workflow | Result |
| --- | --- | --- |
| tag `v*` | `.github/workflows/release.yml` `images` | `cherrypi-agency` and `cherrypi-agent` at `vX.Y.Z` and `latest` (linux/amd64) |
| tag `v*` | `binaries` | agency + agent for `linux/{amd64,arm64}` and `darwin/{amd64,arm64}` + `SHA256SUMS` |
| tag `v*` | `release` | GitHub Release with the tag annotation as its notes and the binaries attached |

## Images

```sh
docker pull ghcr.io/cherrypidotdev/cherrypi-agency:v0.1.0
docker pull ghcr.io/cherrypidotdev/cherrypi-agent:v0.1.0
```

`latest` tracks the most recent release. Packages are public; no login is
needed to pull.

## Binaries

Each release attaches:

```
cherrypi-agency-linux-amd64
cherrypi-agency-linux-arm64
cherrypi-agency-darwin-amd64
cherrypi-agency-darwin-arm64
cherrypi-agent-linux-amd64
cherrypi-agent-linux-arm64
cherrypi-agent-darwin-amd64
cherrypi-agent-darwin-arm64
SHA256SUMS
```

## Reproducing a build

`make binaries` reproduces the attached binaries; `make docker` reproduces the
images. Generated code (proto stubs, sqlc) is not checked in, so install
`buf`/`sqlc` and run the generation steps in
[`README.md`](../README.md#develop) first.

## Release notes

The GitHub Release body is the tag annotation: curated highlights followed by
the grouped changelog. The cumulative log is [`CHANGELOG.md`](../CHANGELOG.md).
