# Multi-stage build producing the API server, seed command, gendata command,
# and importspots command in one small runtime image. Migrations are
# //go:embed-ed into the binary (internal/store/postgres/migrations/*.sql),
# so nothing else needs copying.
#
# Run the API (default):    docker run spotto-server
# Reseed fixed demo data:    docker compose run --rm api /app/seed
# Generate randomized data:  docker compose run --rm api /app/gendata
#
# importspots (real-world places dataset) is NOT run on demand like the two
# above — the entrypoint below runs it automatically on every container
# start, before the API server. By design it wipes the places table (and
# reviews/claims/vibe_votes/checkins/media that reference a place) and
# reloads it from internal/importspots/spots.json on every single boot —
# see internal/importspots/importspots.go for why.

FROM golang:1.26-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/seed ./cmd/seed
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/gendata ./cmd/gendata
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/importspots ./cmd/importspots

FROM alpine:latest
RUN apk add --no-cache ca-certificates && \
    addgroup -S app && adduser -S app -G app
WORKDIR /app
COPY --from=builder --chown=app:app /out/api /app/api
COPY --from=builder --chown=app:app /out/seed /app/seed
COPY --from=builder --chown=app:app /out/gendata /app/gendata
COPY --from=builder --chown=app:app /out/importspots /app/importspots
COPY --chown=app:app docker-entrypoint.sh /app/docker-entrypoint.sh
RUN chmod +x /app/docker-entrypoint.sh
USER app
# Documentation only — the actual listen port comes from the PORT env var
# (see docker-compose.yml), which must be kept in sync with this.
EXPOSE 8881
ENTRYPOINT ["/app/docker-entrypoint.sh"]
