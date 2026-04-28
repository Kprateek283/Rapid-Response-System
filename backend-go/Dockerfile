# ── Stage 1: Build ──────────────────────────────────────────
FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Copy go.mod and go.sum first for layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s" -o /rapid-api ./cmd/api/

# ── Stage 2: Runtime ────────────────────────────────────────
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /rapid-api .

# Cloud Run uses PORT env var
ENV SERVER_PORT=8080
EXPOSE 8080

# Run as non-root
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser

CMD ["./rapid-api"]
