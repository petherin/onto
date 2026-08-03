# ── Build stage ──────────────────────────────────────────────────────────────
FROM golang:1.26-alpine AS builder

WORKDIR /build

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o onto ./cmd/onto

# ── Runtime stage ─────────────────────────────────────────────────────────────
FROM alpine:3.20

WORKDIR /app

COPY --from=builder /build/onto .

# Copy the default data so the container works even without a host volume.
COPY data/ ./data/

ENTRYPOINT ["./onto"]
