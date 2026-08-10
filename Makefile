PKGSITE_BIN := $(shell go env GOPATH)/bin/pkgsite
GOLANGCI_LINT_BIN := $(shell go env GOPATH)/bin/golangci-lint

## ── Run ──────────────────────────────────────────────────────────────────────

.PHONY: run
run:                   ## Run the app natively (requires Go installed)
	go run ./cmd/cli

.PHONY: dashboard
dashboard:             ## Run the multi-pane TUI dashboard natively (requires Go installed)
	go run ./cmd/dashboard

.PHONY: docker-run
docker-run: docker-build  ## Build (if needed) and run the app in Docker
	docker compose run --rm onto

## ── Build ────────────────────────────────────────────────────────────────────

.PHONY: build
build:                 ## Build the native binary to ./onto
	go build -o onto ./cmd/cli

.PHONY: build-dashboard
build-dashboard:       ## Build the native dashboard binary to ./onto-dashboard
	go build -o onto-dashboard ./cmd/dashboard

.PHONY: docker-build
docker-build:          ## Build the Docker image
	docker compose build

.PHONY: docker-clean
docker-clean:          ## Stop and remove any leftover containers, anonymous volumes, and networks
	docker compose down --volumes --remove-orphans

## ── Test ─────────────────────────────────────────────────────────────────────

.PHONY: test
test:                  ## Run all tests
	go test ./...

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