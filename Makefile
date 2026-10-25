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


VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo v0.0.0-dev)

build_frontend:
	cd agency/frontend && make

build_backend:
	cd agent && make

build_agency:
	cd agency/server && make




codegen:
	@set -e; \
	  (cd shared && buf generate) & buf_pid=$$!; \
	  (cd agent && sqlc generate) & agent_pid=$$!; \
	  (cd agency/server && sqlc generate) & server_pid=$$!; \
	  wait $$buf_pid; wait $$agent_pid; wait $$server_pid; \
	  (cd agency/server && go run ./internal/store/gen)


copy_frontend:
	rm -rf agency/server/cmd/agency/frontenddist/*
	touch agency/server/cmd/agency/frontenddist/.gitkeep
	cp -r agency/frontend/dist/* agency/server/cmd/agency/frontenddist/

test: test_backend test_agency test_frontend test_sdk

test_backend:
	cd agent && make test

test_agency:
	cd agency/server && make test

test_frontend:
	cd agency/frontend && make test

test_sdk:
	cd agency/sdk/js && make test

format_root:
	npx prettier --write "*"

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



format_golines:
	@if command -v golangci-lint > /dev/null; then \
	  cd $(CURDIR)/agent && golangci-lint fmt --config $(CURDIR)/.golangci.yaml; \
	  cd $(CURDIR)/agency/server && golangci-lint fmt --config $(CURDIR)/.golangci.yaml; \
	  cd $(CURDIR)/shared && golangci-lint fmt --config $(CURDIR)/.golangci.yaml; \
	else \
	  echo "golangci-lint not found; skipping golines (see TESTING.md#linting)"; \
	fi

format: format_root format_frontend format_backend format_agency format_sdk format_shared format_golines



format_check: format_check_root format_check_frontend format_check_backend format_check_agency format_check_sdk format_check_shared

format_check_root:
	npx prettier --check "*"

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



lint: lint_actions lint_shell lint_go

# actionlint also runs shellcheck on inline `run:` blocks when it is on PATH.
lint_actions:
	@command -v actionlint > /dev/null || { echo "actionlint not found; see TESTING.md#linting"; exit 1; }
	actionlint .gitea/workflows/*.yaml public-overlay/.github/workflows/*.yml

lint_shell:
	@command -v shellcheck > /dev/null || { echo "shellcheck not found; see TESTING.md#linting"; exit 1; }
	shellcheck .gitea/scripts/*.sh









lint_go:
	@command -v golangci-lint > /dev/null || { echo "golangci-lint not found; see TESTING.md#linting"; exit 1; }
	cd agent && golangci-lint run --config $(CURDIR)/.golangci.yaml ./...
	cd agency/server && golangci-lint run --build-tags dev --config $(CURDIR)/.golangci.yaml ./...
	cd shared && golangci-lint run --config $(CURDIR)/.golangci.yaml ./...

docker:
	docker buildx build --platform linux/amd64,linux/arm64 --build-arg VERSION=$(VERSION) -f agency/server/Dockerfile . -t gitea.vladfaust.com/cherrypi/agency
	docker buildx build --platform linux/amd64,linux/arm64 --build-arg VERSION=$(VERSION) -f agent/Dockerfile . -t gitea.vladfaust.com/cherrypi/agent

docker_push: docker
	docker push gitea.vladfaust.com/cherrypi/agency
	docker push gitea.vladfaust.com/cherrypi/agent

compose:
	docker compose up --build



DIST ?= dist
TARGETS ?= linux/amd64 linux/arm64 darwin/amd64 darwin/arm64

binaries: build_frontend copy_frontend binaries_build



binaries_build:
	rm -rf $(DIST) && mkdir -p $(DIST)
	@for target in $(TARGETS); do \
	  os=$${target%/*}; arch=$${target#*/}; \
	  echo "Building $$os/$$arch"; \
	  CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build -C agency/server -o $(abspath $(DIST))/cherrypi-agency-$$os-$$arch ./cmd/agency || exit 1; \
	  CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build -C agent -ldflags "-X cherrypi.vladfaust.com/agent/internal/buildinfo.Version=$(VERSION)" -o $(abspath $(DIST))/cherrypi-agent-$$os-$$arch ./cmd/agent || exit 1; \
	done
	cd $(DIST) && (command -v sha256sum > /dev/null && sha256sum * > SHA256SUMS || shasum -a 256 * > SHA256SUMS)
