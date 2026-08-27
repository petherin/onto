PKGSITE_BIN := $(shell go env GOPATH)/bin/pkgsite
GOLANGCI_LINT_BIN := $(shell go env GOPATH)/bin/golangci-lint
DLV_BIN := $(shell go env GOPATH)/bin/dlv
DEBUG_PORT := 2345

## ── Run ──────────────────────────────────────────────────────────────────────

.PHONY: run
run:                   ## Run the app natively (requires Go installed)
	go run ./cmd/cli

.PHONY: web
web:                   ## Run the browser-based Reality Map at http://localhost:8090 (override with ONTO_WEB_ADDR)
	go run ./cmd/web

.PHONY: debug
debug:                 ## Start a headless Delve debug server on :2345 for VS Code (or any IDE) to attach to
	@if [ ! -f "$(DLV_BIN)" ]; then \
		echo "Installing delve..."; \
		go install github.com/go-delve/delve/cmd/dlv@latest; \
	fi
	@echo "Delve listening on :$(DEBUG_PORT) — attach your IDE debugger now (VS Code: 'Attach to Onto (Delve)')"
	$(DLV_BIN) debug ./cmd/cli --headless --listen=:$(DEBUG_PORT) --api-version=2 --accept-multiclient

.PHONY: debug-kill
debug-kill:            ## Force-kill any stuck dlv/debug processes (use if a debug session hangs, e.g. paused at a breakpoint after the client disconnects)
	@pids=$$(pgrep -f 'dlv debug|dlv exec|__debug_bin|debugserver.*__debug_bin' 2>/dev/null); \
	if [ -z "$$pids" ]; then \
		echo "No stuck debug processes found."; \
	else \
		echo "Killing: $$pids"; \
		kill -9 $$pids; \
	fi

.PHONY: docker-run
docker-run: docker-build  ## Build (if needed) and run the app in Docker
	docker compose run --rm onto

## ── Build ────────────────────────────────────────────────────────────────────

.PHONY: build
build:                 ## Build the native binary to ./onto
	go build -o onto ./cmd/cli

.PHONY: build-web
build-web:             ## Build the native web binary to ./onto-web (embeds static assets)
	go build -o onto-web ./cmd/web

.PHONY: docker-build
docker-build:          ## Build the Docker image
	docker compose build

.PHONY: docker-clean
docker-clean:          ## Stop and remove any leftover containers, anonymous volumes, and networks
	docker compose down --volumes --remove-orphans

## ── Test ─────────────────────────────────────────────────────────────────────

.PHONY: test
test: test-js          ## Run all tests (Go + front-end JS)
	go test ./...

.PHONY: test-js
test-js:               ## Run the front-end JS unit tests (requires Node.js)
	cd internal/interface/web && node --test

.PHONY: validate-locations
validate-locations:    ## Validate data/locations.json graph invariants
	go run ./scripts/validate_locations.go

.PHONY: lint
lint:                  ## Run golangci-lint v2 (requires: go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest)
	$(GOLANGCI_LINT_BIN) run ./...

.PHONY: fmt
fmt:                   ## Format all Go source files with gofmt
	gofmt -w .
	$(GOLANGCI_LINT_BIN) run --fix ./...

.PHONY: mocks
mocks:                 ## Regenerate testify mocks from domain interfaces (requires: go install github.com/vektra/mockery/v2@latest)
	mockery

## ── Misc ─────────────────────────────────────────────────────────────────────

.PHONY: toc
toc:                   ## Regenerate the README table of contents
	docker run --rm -v $(PWD):/work -w /work node:18 npx -y markdown-toc -i README.md

.PHONY: help
help:                  ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*##' Makefile | awk 'BEGIN {FS = ":.*##"}; {printf "  %-16s %s\n", $$1, $$2}'

.PHONY: docs
docs:                  ## Start local documentation server (installs pkgsite if needed)
	@echo "Starting documentation server on http://localhost:8080..."
	@if [ ! -f "$(PKGSITE_BIN)" ]; then \
		echo "Installing pkgsite..."; \
		go install golang.org/x/pkgsite/cmd/pkgsite@v0.1.0; \
	fi
	@echo "Press Ctrl+C to stop"
	@$(PKGSITE_BIN) -http=:8080