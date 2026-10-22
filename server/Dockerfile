# Multi-stage build producing the API server, seed command, gendata command,
# and importsrilanka command in one small runtime image. Migrations are
# //go:embed-ed into the binary (internal/store/postgres/migrations/*.sql),
# so nothing else needs copying.
#
# Run the API (default):    docker run spotto-server
# Reseed fixed demo data:    docker compose run --rm api /app/seed
# Generate randomized data:  docker compose run --rm api /app/gendata
#
# importsrilanka (real-world places dataset, from sri_lanka_spots.json) is
# NOT run on demand like the two above — the entrypoint below runs it
# automatically on every container start, before the API server. It wipes
# the places table (and reviews/claims/vibe_checkins/checkins/media that
# reference a place) and reloads it, with a vendor + owner account per
# place, on every single boot — see cmd/importsrilanka/main.go for why, and
# note the entrypoint tolerates it failing so a bad run can't take the API
# down.

FROM golang:1.26-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/seed ./cmd/seed
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/gendata ./cmd/gendata
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/importsrilanka ./cmd/importsrilanka
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/seedcategories ./cmd/seedcategories

# debian:bookworm-slim, not alpine — yt-dlp's Linux release binary is a
# PyInstaller build with an embedded CPython linked against real glibc.
# Alpine's musl libc can't run it even with the gcompat shim installed (it
# gets far enough to extract the bundle, then fails relocating glibc-only
# symbols like dladdr1) — this is a well-known yt-dlp-on-Alpine limitation,
# not something a compat package fixes. yt-dlp fetches a public Instagram
# reel/post's caption via `yt-dlp --dump-json --skip-download <url>` for the
# Inspiration Board's "extract places" feature — see
# internal/http/inspiration_extract.go. No official Instagram API exists for
# this, so this is the best-effort fetch step ahead of Gemini place-matching.
# Checksum-verified so a compromised/altered release asset can't silently
# end up in the image. Fetched via the "latest" alias rather than a
# version-tagged URL: GitHub's tag-specific asset URLs redirect to a
# short-lived signed release-assets.githubusercontent.com link that this
# environment's Docker build network couldn't reliably reach (consistent
# 404s), while the /latest/download/ alias worked reliably — the checksum
# pin still catches any drift (a newer "latest" release fails the build
# loudly instead of silently shipping unverified bytes) so update
# YTDLP_SHA256 (from that release's SHA2-256SUMS asset on GitHub) whenever
# that happens.
FROM debian:bookworm-slim
ARG YTDLP_SHA256=d8bd33fca9d5b23bd5b9952b3fca36db5c46331933fda9f598a3da14ce2c243f
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl && \
    curl -fL --retry 3 --retry-delay 2 -o /usr/local/bin/yt-dlp "https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp_linux" && \
    echo "${YTDLP_SHA256}  /usr/local/bin/yt-dlp" | sha256sum -c - && \
    chmod +x /usr/local/bin/yt-dlp && \
    rm -rf /var/lib/apt/lists/* && \
    groupadd -r app && useradd -r -g app app
WORKDIR /app
COPY --from=builder --chown=app:app /out/api /app/api
COPY --from=builder --chown=app:app /out/seed /app/seed
COPY --from=builder --chown=app:app /out/gendata /app/gendata
COPY --from=builder --chown=app:app /out/importsrilanka /app/importsrilanka
COPY --from=builder --chown=app:app /out/seedcategories /app/seedcategories
COPY --chown=app:app sri_lanka_spots_r2_ready.json /app/sri_lanka_spots_r2_ready.json
COPY --chown=app:app docker-entrypoint.sh /app/docker-entrypoint.sh
RUN chmod +x /app/docker-entrypoint.sh
USER app
# Documentation only — the actual listen port comes from the PORT env var
# (see docker-compose.yml), which must be kept in sync with this.
EXPOSE 8881
ENTRYPOINT ["/app/docker-entrypoint.sh"]
