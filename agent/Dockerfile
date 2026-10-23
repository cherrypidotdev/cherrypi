# syntax=docker/dockerfile:1

# CherryPi: A multi-agent-multi-human self-hosted agency
# Copyright (C) 2026 CherryPi authors <cherrypi.dev>
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU Affero General Public License as published
# by the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.
#
# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU Affero General Public License for more details.
#
# You should have received a copy of the GNU Affero General Public License
# along with this program.  If not, see <https://www.gnu.org/licenses/>.


# The agent image is a headless worker: it runs Pi and hosts sessions and
# terminals, and dials the agency over the tunnel. It ships no web UI. It also
# carries the desktop bridge (`x11vnc` + X utilities) so it can attach to an
# already-running X session; see docs/desktop.md.

# ---- Code generation --------------------------------------------------------
# Generated trees are gitignored: the shared Go protobuf/connect stubs
# (shared/gen) and the agent's sqlc code (agent/internal/gen/sqlc).
FROM golang:1.26 AS codegen
RUN --mount=type=cache,target=/go/pkg/mod \
  --mount=type=cache,target=/root/.cache/go-build \
  go install github.com/bufbuild/buf/cmd/buf@v1.70.0 \
  && go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1

WORKDIR /src/shared
COPY shared/go.mod shared/go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
  --mount=type=cache,target=/root/.cache/go-build \
  go mod download
COPY shared/buf.yaml shared/buf.gen.yaml ./
COPY shared/proto ./proto
RUN --mount=type=cache,target=/go/pkg/mod \
  --mount=type=cache,target=/root/.cache/go-build \
  buf generate

WORKDIR /src/agent
COPY agent/go.mod agent/go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
  --mount=type=cache,target=/root/.cache/go-build \
  go mod download
COPY agent/sqlc.yaml ./
COPY agent/sql ./sql
RUN sqlc generate

# ---- Agent build ------------------------------------------------------------
FROM golang:1.26 AS backend
# The agent build version, stamped into the binary so it can report it in the
# tunnel hello. `make docker`/build-push.sh pass the git-describe VERSION.
ARG VERSION=dev
WORKDIR /src/agent
COPY agent/go.mod agent/go.sum ./
# agent/go.mod replaces cherrypi.vladfaust.com/shared with ../shared, so the
# replaced module's go.mod must exist for `go mod download` to resolve it.
COPY shared/go.mod /src/shared/go.mod
COPY shared/go.sum /src/shared/go.sum
RUN --mount=type=cache,target=/go/pkg/mod \
  --mount=type=cache,target=/root/.cache/go-build \
  go mod download

COPY shared /src/shared
COPY --from=codegen /src/shared/gen /src/shared/gen

COPY agent/cmd ./cmd
# The hand-written sql package embeds sql/migrations via //go:embed.
COPY agent/sql ./sql
COPY agent/internal ./internal
# Generated sqlc + Go protobuf/connect code from the codegen stage (overwrites
# the excluded host gen tree).
COPY --from=codegen /src/agent/internal/gen ./internal/gen

RUN --mount=type=cache,target=/go/pkg/mod \
  --mount=type=cache,target=/root/.cache/go-build \
  CGO_ENABLED=0 go build \
  -ldflags "-X cherrypi.vladfaust.com/agent/internal/buildinfo.Version=${VERSION}" \
  -o /out/cherrypi-agent ./cmd/agent

# ---- Runtime ----------------------------------------------------------------
FROM ubuntu:24.04 AS runtime
ENV DEBIAN_FRONTEND=noninteractive

ENV NVM_DIR=/opt/nvm
ENV NODE_VERSION=22.21.1

# Install system dependencies and tools for the runtime environment. The apt
# cache mounts keep the package lists and downloaded .debs across rebuilds;
# cache mounts are not committed to the image, so no cleanup is needed.
RUN --mount=type=cache,target=/var/cache/apt,sharing=locked \
  --mount=type=cache,target=/var/lib/apt,sharing=locked \
  apt-get update && apt-get install -y --no-install-recommends \
  wget ca-certificates curl git openssh-client gnupg \
  x11vnc x11-utils xauth

# Install Node.js via NVM
RUN mkdir -p "$NVM_DIR" \
  && curl --silent -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.8/install.sh | bash \
  && . "$NVM_DIR/nvm.sh" \
  && nvm install "$NODE_VERSION" \
  && nvm alias default "$NODE_VERSION" \
  && nvm use default

ENV PATH=$NVM_DIR/versions/node/v$NODE_VERSION/bin:$PATH

# Install the Pi Coding Agent globally via npm
RUN npm install -g --ignore-scripts @earendil-works/pi-coding-agent@0.99.1 \
  && pi --version

COPY --from=backend /out/cherrypi-agent /usr/local/bin/cherrypi-agent

# No port is exposed: the agent dials the agency and serves its API over the
# tunnel.
ENTRYPOINT ["cherrypi-agent"]
