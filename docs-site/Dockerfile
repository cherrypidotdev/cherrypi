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
# Standalone image for the public VitePress documentation.
FROM node:22 AS build


ARG DOCS_BASE=/
ENV DOCS_BASE=$DOCS_BASE

WORKDIR /src/docs-site
COPY docs-site/package.json docs-site/package-lock.json ./
RUN --mount=type=cache,target=/root/.npm \
  npm ci
COPY docs-site ./

RUN npm run build


FROM nginx:1.29.8-alpine AS runtime

ARG VERSION=dev
LABEL org.opencontainers.image.title="CherryPi documentation" \
  org.opencontainers.image.description="Public CherryPi documentation (VitePress, English + Russian)" \
  org.opencontainers.image.url="https://docs.cherrypi.dev" \
  org.opencontainers.image.source="https://github.com/cherrypidotdev/cherrypi" \
  org.opencontainers.image.version="$VERSION" \
  org.opencontainers.image.licenses="AGPL-3.0-only"

COPY docs-site/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /src/docs-site/.vitepress/dist /usr/share/nginx/html

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -q --spider http://127.0.0.1:8080/ || exit 1
