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


FROM golang:1.26 AS backend

ARG VERSION=dev
WORKDIR /src/agent
COPY agent/go.mod agent/go.sum ./


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


COPY --from=codegen /src/agent/internal/gen ./internal/gen

RUN --mount=type=cache,target=/go/pkg/mod \
  --mount=type=cache,target=/root/.cache/go-build \
  CGO_ENABLED=0 go build \
  -ldflags "-X 'cherrypi.vladfaust.com/agent/internal/buildinfo.Version=${VERSION}'" \
  -o /out/cherrypi-agent ./cmd/agent


FROM ubuntu:24.04 AS runtime
ENV DEBIAN_FRONTEND=noninteractive

RUN --mount=type=cache,target=/var/cache/apt,sharing=locked \
  --mount=type=cache,target=/var/lib/apt,sharing=locked \
  apt-get update && apt-get install -y --no-install-recommends \
  ca-certificates curl git openssh-client gnupg \
  x11vnc x11-utils xauth

COPY --from=backend /out/cherrypi-agent /usr/local/bin/cherrypi-agent

ENTRYPOINT ["cherrypi-agent"]
