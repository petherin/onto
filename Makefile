## ── Run ──────────────────────────────────────────────────────────────────────

.PHONY: run
run:                   ## Run the app natively (requires Go installed)
	go run ./cmd/onto

.PHONY: docker-run
docker-run:            ## Run the app in Docker (requires Docker installed)
	docker compose run --rm onto

## ── Build ────────────────────────────────────────────────────────────────────

.PHONY: build
build:                 ## Build the native binary to ./onto
	go build -o onto ./cmd/onto

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
