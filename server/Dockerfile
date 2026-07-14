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
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /out/api /app/api
COPY --from=builder /out/seed /app/seed
EXPOSE 8080
CMD ["/app/api"]
