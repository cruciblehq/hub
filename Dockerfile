# Build stage
FROM golang:1.25.1-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary (pure Go, no CGO)
RUN CGO_ENABLED=0 go build -o hub ./cmd/hub

# Runtime stage
FROM alpine:latest

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/hub .

# Expose default port
EXPOSE 8080

# Set default environment variables
ENV PORT=8080
ENV DB_PATH=/data/hub.db
ENV ARCHIVE_ROOT=/data/archives

# Create data directory
RUN mkdir -p /data/archives

CMD ["./hub"]
