# CLI image: builds and runs the interactive terminal game (cmd/cli). This is the
# standalone containerised CLI — unrelated to the MiniStack AWS simulation under
# deploy/ministack (which runs the web/API on S3 + ECS/ALB).
#
# Build context is the repository root:
#   docker build -f deploy/cli/Dockerfile -t onto:local .

# ── Build stage ──────────────────────────────────────────────────────────────
FROM golang:1.26-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o onto ./cmd/cli

# ── Runtime stage ─────────────────────────────────────────────────────────────
FROM alpine:3.20

WORKDIR /app

# Run as an unprivileged user rather than root: the CLI only needs to read its
# binary and read/write /app/data (no root capabilities required).
RUN addgroup -S onto && adduser -S -G onto onto

COPY --from=builder --chown=onto:onto /build/onto .

# Ensure the data directory exists even without a host volume mount; the app
# creates locations.json itself on first save if it's missing.
COPY --chown=onto:onto data/ ./data/

USER onto

ENTRYPOINT ["./onto"]
