# Build stage
FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
# Allow GOTOOLCHAIN to auto-download if needed
ENV GOTOOLCHAIN=auto
RUN go mod download

# Copy source
COPY . .

# Build binaries
RUN CGO_ENABLED=0 GOOS=linux go build -o /quantumlife ./cmd/quantumlife
RUN CGO_ENABLED=0 GOOS=linux go build -o /ql ./cmd/ql

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy binaries
COPY --from=builder /quantumlife /app/quantumlife
COPY --from=builder /ql /app/ql

# Create data directory
RUN mkdir -p /data

# Environment
ENV QUANTUMLIFE_DATA_DIR=/data
ENV QUANTUMLIFE_PORT=8080

EXPOSE 8080

VOLUME ["/data"]

ENTRYPOINT ["/app/quantumlife"]
CMD ["--data-dir", "/data", "--port", "8080"]
