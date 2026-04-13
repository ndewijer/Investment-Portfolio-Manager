# Multi-stage build for optimal image size
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Download dependencies first (cached layer)
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
ARG VERSION=dev
RUN go build \
    -ldflags "-X github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/version.Version=${VERSION}" \
    -o server \
    ./cmd/server/main.go

# Final stage — minimal image
FROM alpine:3.23

# ca-certificates for outbound HTTPS (Yahoo Finance, IBKR)
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/server ./server

# Copy entrypoint
COPY docker-entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENV DB_DIR=/data/db \
    LOG_DIR=/data/logs \
    DOMAIN=localhost

RUN mkdir -p /data/db /data/logs

EXPOSE 5000

ENTRYPOINT ["/entrypoint.sh"]
CMD ["/app/server"]
