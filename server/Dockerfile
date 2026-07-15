# Multi-stage build producing both the API server and the seed command in
# one small runtime image. Migrations are //go:embed-ed into the binary
# (internal/store/postgres/migrations/*.sql), so nothing else needs copying.
#
# Run the API (default):    docker run spotto-server
# Reseed demo data:         docker compose run --rm api /app/seed

FROM golang:1.26-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/seed ./cmd/seed

FROM alpine:latest
RUN apk add --no-cache ca-certificates && \
    addgroup -S app && adduser -S app -G app
WORKDIR /app
COPY --from=builder --chown=app:app /out/api /app/api
COPY --from=builder --chown=app:app /out/seed /app/seed
USER app
# Documentation only — the actual listen port comes from the PORT env var
# (see docker-compose.yml), which must be kept in sync with this.
EXPOSE 8881
CMD ["/app/api"]
