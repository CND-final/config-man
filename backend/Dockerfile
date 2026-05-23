# ── Stage 1: Build ───────────────────────────────────────────────────────────
FROM golang:1.26-alpine AS builder

WORKDIR /build

# Copy dependency manifests first for layer caching.
# go mod download only re-runs when go.mod/go.sum change.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO_ENABLED=0 produces a fully static binary (no glibc dependency).
# pgx/v5/stdlib is pure Go, so CGO is not required.
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/config-man ./cmd

# ── Stage 2: Runtime ─────────────────────────────────────────────────────────
FROM alpine:3

# ca-certificates: required if any upstream connection uses TLS.
RUN apk add --no-cache ca-certificates

WORKDIR /app
COPY --from=builder /app/config-man .

EXPOSE 3000

CMD ["/app/config-man"]
