# syntax=docker/dockerfile:1

FROM golang:1.24-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .

# Build bucketbird binary (includes built-in migration support)
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/bucketbird ./cmd/bucketbird

# Build migrate tool
RUN CGO_ENABLED=0 GOOS=linux go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest && \
    cp /go/bin/migrate /out/migrate

FROM alpine:3.20
WORKDIR /app

# Copy application binary and migrate tool
COPY --from=builder /out/bucketbird /app/bucketbird
COPY --from=builder /out/migrate /usr/local/bin/migrate

# Copy migrations
COPY migrations /app/migrations

# Copy entrypoint script
COPY docker-entrypoint.sh /app/docker-entrypoint.sh
RUN chmod +x /app/docker-entrypoint.sh

EXPOSE 8080
ENTRYPOINT ["/app/docker-entrypoint.sh"]
