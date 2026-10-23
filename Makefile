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

build: build_frontend copy_frontend build_backend build_agency

# Version shown in the SPA footer; passed into the Docker frontend build.
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo v0.0.0-dev)

build_frontend:
	cd agency/frontend && make

build_backend:
	cd agent && make

build_agency:
	cd agency/server && make

# The SPA is embedded only by the agency; the agent is a headless worker.
copy_frontend:
	rm -rf agency/server/cmd/agency/frontenddist/*
	touch agency/server/cmd/agency/frontenddist/.gitkeep
	cp -r agency/frontend/dist/* agency/server/cmd/agency/frontenddist/


format_root:
	npx prettier --write "*" .github public-overlay deploy

format_frontend:
	cd agency/frontend && npx prettier --write .

format_backend:
	cd agent && npx prettier --write . && go fmt ./...

format_agency:
	cd agency/server && npx prettier --write . && go fmt ./...

format_sdk:
	cd agency/sdk/js && npx prettier --write .

format_shared:
	cd shared && go fmt ./...

# golines (via golangci-lint) is enforced by `lint`; apply it during `format`
# when golangci-lint is on PATH, so the pre-commit hook stays usable without it.
format_golines:
	@if command -v golangci-lint > /dev/null; then \
	  cd $(CURDIR)/agent && golangci-lint fmt --config $(CURDIR)/.golangci.yaml; \
	  cd $(CURDIR)/agency/server && golangci-lint fmt --config $(CURDIR)/.golangci.yaml; \
	  cd $(CURDIR)/shared && golangci-lint fmt --config $(CURDIR)/.golangci.yaml; \
	else \
	  echo "golangci-lint not found; skipping golines"; \
	fi

format: format_root format_frontend format_backend format_agency format_sdk format_shared format_golines

# Non-mutating counterpart of `format`, used by CI to fail the build on
# formatting violations.
format_check: format_check_root format_check_frontend format_check_backend format_check_agency format_check_sdk format_check_shared

format_check_root:
	npx prettier --check "*" .github public-overlay deploy

format_check_frontend:
	cd agency/frontend && npx prettier --check .

format_check_backend:
	cd agent && npx prettier --check .
	@unformatted="$$(cd agent && gofmt -l .)"; \
	if [ -n "$$unformatted" ]; then \
		echo "gofmt required for:"; echo "$$unformatted"; exit 1; \
	fi

format_check_agency:
	cd agency/server && npx prettier --check .
	@unformatted="$$(cd agency/server && gofmt -l .)"; \
	if [ -n "$$unformatted" ]; then \
		echo "gofmt required for:"; echo "$$unformatted"; exit 1; \
	fi

format_check_sdk:
	cd agency/sdk/js && npx prettier --check .

format_check_shared:
	@unformatted="$$(cd shared && gofmt -l .)"; \
	if [ -n "$$unformatted" ]; then \
		echo "gofmt required for:"; echo "$$unformatted"; exit 1; \
	fi


docker:
	docker buildx build --platform linux/amd64 --build-arg VERSION=$(VERSION) -f agency/server/Dockerfile . -t ghcr.io/cherrypidotdev/cherrypi-agency
	docker buildx build --platform linux/amd64 --build-arg VERSION=$(VERSION) -f agent/Dockerfile . -t ghcr.io/cherrypidotdev/cherrypi-agent

docker_push: docker
	docker push ghcr.io/cherrypidotdev/cherrypi-agency
	docker push ghcr.io/cherrypidotdev/cherrypi-agent

compose:
	docker compose up --build

# Cross-compiled release binaries for both binaries, each with the SPA
# embedded. Produces dist/ + SHA256SUMS.
DIST ?= dist
TARGETS ?= linux/amd64 linux/arm64 darwin/amd64 darwin/arm64

binaries: build_frontend copy_frontend
	rm -rf $(DIST) && mkdir -p $(DIST)
	@for target in $(TARGETS); do \
	  os=$${target%/*}; arch=$${target#*/}; \
	  echo "Building $$os/$$arch"; \
	  CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build -C agency/server -o $(abspath $(DIST))/cherrypi-agency-$$os-$$arch ./cmd/agency || exit 1; \
	  CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build -C agent -ldflags "-X cherrypi.vladfaust.com/agent/internal/buildinfo.Version=$(VERSION)" -o $(abspath $(DIST))/cherrypi-agent-$$os-$$arch ./cmd/agent || exit 1; \
	done
	cd $(DIST) && (command -v sha256sum > /dev/null && sha256sum * > SHA256SUMS || shasum -a 256 * > SHA256SUMS)
