PKGSITE_BIN := $(shell go env GOPATH)/bin/pkgsite
GOLANGCI_LINT_BIN := $(shell go env GOPATH)/bin/golangci-lint
DLV_BIN := $(shell go env GOPATH)/bin/dlv
DEBUG_PORT := 2345

# Standalone CLI game in a container (deploy/cli). Distinct from the MiniStack
# AWS simulation (MINISTACK_COMPOSE, below). --project-directory . in the docker-*
# targets resolves its build context, ./data mount, and .env against the repo root.
CLI_COMPOSE := deploy/cli/docker-compose.yml

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
docker-run: docker-build  ## Build (if needed) and run the CLI game in Docker
	docker compose -f $(CLI_COMPOSE) --project-directory . run --rm cli

## ── Build ────────────────────────────────────────────────────────────────────

.PHONY: build
build:                 ## Build the native binary to ./onto
	go build -o onto ./cmd/cli

.PHONY: build-web
build-web:             ## Build the native web binary to ./onto-web (embeds static assets)
	go build -o onto-web ./cmd/web

.PHONY: docker-build
docker-build:          ## Build the CLI game Docker image
	docker compose -f $(CLI_COMPOSE) --project-directory . build

.PHONY: docker-clean
docker-clean:          ## Stop and remove any leftover CLI containers, anonymous volumes, and networks
	docker compose -f $(CLI_COMPOSE) --project-directory . down --volumes --remove-orphans

## ── MiniStack (local AWS: S3 + ECS/ALB + Route53 via Terraform) ───────────────

MINISTACK_COMPOSE := deploy/ministack/docker-compose.yml
MINISTACK_HOSTS   := onto.world api.onto.world
API_IMAGE         := onto-api:local

TF_LOCAL := deploy/terraform/envs/local
TF_AWS   := deploy/terraform/envs/aws
# MiniStack creds + checksum settings (its S3 rejects the SDK's default CRC64NVME).
TF_LOCAL_ENV := AWS_ACCESS_KEY_ID=test AWS_SECRET_ACCESS_KEY=test AWS_DEFAULT_REGION=eu-west-1 \
	AWS_REQUEST_CHECKSUM_CALCULATION=WHEN_REQUIRED AWS_RESPONSE_CHECKSUM_VALIDATION=WHEN_REQUIRED

.PHONY: ministack
ministack: ministack-hosts ministack-up ministack-provision  ## One command: mock the domain in /etc/hosts, start MiniStack/StackPort/edge, build the image, terraform apply the whole stack
	@echo ""
	@echo "Onto is up:"
	@echo "  SPA        → http://onto.world/"
	@echo "  API        → http://api.onto.world/api/state"
	@echo "  StackPort  → http://localhost:8080/   (browse MiniStack resources)"

.PHONY: ministack-up
ministack-up:          ## Start MiniStack, the StackPort UI, and the local edge proxy
	docker compose -f $(MINISTACK_COMPOSE) up -d

.PHONY: ministack-image
ministack-image:       ## Build the API image MiniStack's ECS RunTask starts
	docker build -f deploy/ministack/Dockerfile.api -t $(API_IMAGE) .

.PHONY: ministack-provision
ministack-provision: ministack-image  ## Build the API image and terraform apply (S3 + ECS/ALB + Route53) on MiniStack
	cd $(TF_LOCAL) && $(TF_LOCAL_ENV) terraform init -input=false
	cd $(TF_LOCAL) && $(TF_LOCAL_ENV) terraform apply -auto-approve -input=false

.PHONY: ministack-down
ministack-down:        ## Tear down: terraform destroy, remove ECS containers, stop MiniStack + edge
	-cd $(TF_LOCAL) && $(TF_LOCAL_ENV) terraform destroy -auto-approve -input=false
	-docker ps -aq --filter 'name=ministack-ecs-' | xargs -r docker rm -f
	docker compose -f $(MINISTACK_COMPOSE) down --remove-orphans

## ── AWS (real deployment via Terraform) ───────────────────────────────────────

.PHONY: tf-aws-init
tf-aws-init:           ## terraform init for the real AWS environment (deploy/terraform/envs/aws)
	cd $(TF_AWS) && terraform init -input=false

.PHONY: tf-aws-plan
tf-aws-plan:           ## terraform plan for real AWS (supply vars via terraform.tfvars or -var)
	cd $(TF_AWS) && terraform plan

.PHONY: tf-aws-apply
tf-aws-apply:          ## terraform apply for real AWS
	cd $(TF_AWS) && terraform apply

.PHONY: ministack-hosts
ministack-hosts:       ## Add onto.world / api.onto.world → 127.0.0.1 to /etc/hosts (needs sudo)
	@for h in $(MINISTACK_HOSTS); do \
		if grep -qE "^127\.0\.0\.1[[:space:]].*$$h( |$$)" /etc/hosts; then \
			echo "$$h already in /etc/hosts"; \
		else \
			echo "Adding $$h to /etc/hosts (sudo)"; \
			echo "127.0.0.1 $$h" | sudo tee -a /etc/hosts >/dev/null; \
		fi; \
	done

.PHONY: ministack-hosts-remove
ministack-hosts-remove: ## Remove the onto.world / api.onto.world entries from /etc/hosts (needs sudo)
	@for h in $(MINISTACK_HOSTS); do \
		echo "Removing $$h from /etc/hosts (sudo)"; \
		sudo sed -i.bak "/^127\.0\.0\.1[[:space:]][[:space:]]*$$h$$/d" /etc/hosts; \
	done

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
	@echo "Starting documentation server on http://localhost:6060..."
	@if [ ! -f "$(PKGSITE_BIN)" ]; then \
		echo "Installing pkgsite..."; \
		go install golang.org/x/pkgsite/cmd/pkgsite@v0.1.0; \
	fi
	@echo "Press Ctrl+C to stop"
	@$(PKGSITE_BIN) -http=:6060