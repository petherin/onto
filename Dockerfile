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

COPY --from=builder /build/onto .

# Ensure the data directory exists even without a host volume mount; the app
# creates locations.json itself on first save if it's missing.
COPY data/ ./data/

ENTRYPOINT ["./onto"]
