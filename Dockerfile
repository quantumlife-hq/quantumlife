# Build stage
FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git ca-certificates gcc musl-dev

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
# Allow GOTOOLCHAIN to auto-download if needed
ENV GOTOOLCHAIN=auto
RUN go mod download

# Copy source
COPY . .

# Build binaries (CGO enabled for SQLite)
RUN CGO_ENABLED=1 GOOS=linux go build -o /quantumlife ./cmd/quantumlife
RUN CGO_ENABLED=1 GOOS=linux go build -o /ql ./cmd/ql

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata sqlite-libs

WORKDIR /app

# Copy binaries
COPY --from=builder /quantumlife /app/quantumlife
COPY --from=builder /ql /app/ql

# Copy static files, landing page, and migrations
COPY internal/api/static /app/static
COPY web/landing /app/web/landing
COPY migrations /app/migrations

# Create data directory
RUN mkdir -p /data

# Environment
ENV DATABASE_PATH=/data/quantumlife.db
ENV QUANTUMLIFE_DATA_DIR=/data
ENV QUANTUMLIFE_PORT=8080

EXPOSE 8080 8090

VOLUME ["/data"]

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/v1/health || exit 1

ENTRYPOINT ["/app/quantumlife"]
CMD ["--data-dir", "/data", "--port", "8080"]
