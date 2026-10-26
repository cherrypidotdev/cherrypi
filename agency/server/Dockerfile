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

WORKDIR /src/agency/server
COPY agency/server/sqlc.yaml ./
COPY agency/server/sql ./sql
RUN sqlc generate


FROM node:22 AS frontend

ARG VERSION=
ENV VITE_APP_VERSION=$VERSION

WORKDIR /src/agency/sdk/js
COPY agency/sdk/js/package.json agency/sdk/js/package-lock.json ./
RUN --mount=type=cache,target=/root/.npm \
  npm ci

COPY agency/sdk/js/tsconfig.json agency/sdk/js/tsconfig.build.json ./
COPY agency/sdk/js/src ./src
COPY --from=codegen /src/agency/sdk/js/src/gen ./src/gen
RUN npm run build

WORKDIR /src/agency/frontend
COPY agency/frontend/package.json agency/frontend/package-lock.json ./
RUN --mount=type=cache,target=/root/.npm \
  npm ci

COPY agency/frontend/index.html agency/frontend/vite.config.ts agency/frontend/tsconfig.json agency/frontend/tsconfig.app.json agency/frontend/tsconfig.node.json agency/frontend/postcss.config.cjs ./
COPY agency/frontend/components.json ./
COPY agency/frontend/src ./src
COPY agency/frontend/public ./public


COPY agency/frontend/scripts ./scripts
RUN npm run build


FROM golang:1.26 AS backend


ARG VERSION=dev
WORKDIR /src/agency/server
COPY agency/server/go.mod agency/server/go.sum ./


COPY shared/go.mod /src/shared/go.mod
COPY shared/go.sum /src/shared/go.sum
RUN --mount=type=cache,target=/go/pkg/mod \
  --mount=type=cache,target=/root/.cache/go-build \
  go mod download

COPY shared /src/shared
COPY --from=codegen /src/shared/gen /src/shared/gen

# The hand-written sql package embeds the sqlite/pg migration trees via //go:embed.
COPY agency/server/sql ./sql
COPY agency/server/internal ./internal

COPY --from=codegen /src/agency/server/internal/gen ./internal/gen


RUN go run ./internal/store/gen
COPY agency/server/cmd ./cmd


COPY --from=frontend /src/agency/frontend/dist ./cmd/agency/frontenddist/

RUN --mount=type=cache,target=/go/pkg/mod \
  --mount=type=cache,target=/root/.cache/go-build \
  CGO_ENABLED=0 go build \
  -ldflags "-X 'cherrypi.vladfaust.com/agency/cmd/agency/buildinfo.Version=${VERSION}'" \
  -o /out/cherrypi-agency ./cmd/agency


FROM ubuntu:24.04 AS runtime
ENV DEBIAN_FRONTEND=noninteractive

RUN --mount=type=cache,target=/var/cache/apt,sharing=locked \
  --mount=type=cache,target=/var/lib/apt,sharing=locked \
  apt-get update && apt-get install -y --no-install-recommends \
  ca-certificates curl

COPY --from=backend /out/cherrypi-agency /usr/local/bin/cherrypi-agency

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD curl -fsS http://localhost:8080/api/rest/health || exit 1

ENTRYPOINT ["cherrypi-agency"]
