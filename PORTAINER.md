# Deploying Spotto on Portainer

Three services — `api` (Go, :8080), `vendor` (Next.js, :3000), `admin`
(Next.js, :3001) — built from `docker-compose.yml` at the repo root. This
stack does **not** run its own Postgres; it connects to one you already have
reachable from the Docker host.

## 1. One-time database setup

On whatever Postgres you're pointing at:

```sql
CREATE DATABASE spotto;
```

## 2. Push this repo to GitHub (if not already)

```powershell
git remote add origin https://github.com/malinanu/travellingApp.git
git branch -M main
git push -u origin main
```

## 3. Fill in your environment

Copy `.env.example` to `.env` and fill in real values — **you'll paste this
into Portainer's stack environment editor**, not commit it:

```
DATABASE_URL=postgres://postgres:<password>@<postgres-host>:5432/spotto?sslmode=disable
JWT_SECRET=<a long random string>
CORS_ORIGINS=http://<server-ip-or-domain>:3000,http://<server-ip-or-domain>:3001
NEXT_PUBLIC_API_URL=http://<server-ip-or-domain>:8080
```

`NEXT_PUBLIC_API_URL` is baked into the vendor/admin webapps **at build
time** (Next.js public env vars aren't runtime-configurable) — if your
server's address ever changes, the stack needs a rebuild, not just a
restart (see step 5).

If your Postgres runs as another container on the same Docker host (not via
Portainer's own network), `<postgres-host>` can usually be that container's
name if it's on a shared Docker network, or `host.docker.internal` if it's
bound to the host's own port.

## 4. Deploy in Portainer

**Stacks → Add stack → Repository**:
- Repository URL: `https://github.com/malinanu/travellingApp.git`
- Compose path: `docker-compose.yml`
- Environment variables: paste the contents of your filled-in `.env`
  (one `KEY=value` per line, using Portainer's environment editor)
- **Deploy the stack**

This builds all three images on the Docker host and starts them.

## 5. Load demo data (first run only)

From Portainer's console for the `api` container, or via the CLI on the
host:

```
docker compose run --rm api /app/seed
```

This wipes and reloads the demo dataset — accounts `lena@spotto.dev`,
`vendor@driftwood.dev`, `admin@spotto.dev`, password `spotto123` for all.
Safe to re-run any time you want a clean demo state.

## 6. Verify

- `http://<server>:8080/health` → `{"status":"ok"}`
- `http://<server>:3000/login` → vendor portal
- `http://<server>:3001/login` → admin command center

## Updating after a config change

- **Code change**: re-deploy the stack from Portainer (pulls latest `main`,
  rebuilds).
- **`NEXT_PUBLIC_API_URL` or any build-time value changed**: rebuild the
  affected service images, not just restart —
  `docker compose build vendor admin && docker compose up -d`.
- **`DATABASE_URL`/`JWT_SECRET`/`CORS_ORIGINS` changed**: these are
  runtime env vars for `api` only — a restart is enough.

## Notes / limitations

- This exposes plain HTTP directly on three ports — there's no TLS or
  reverse proxy in this stack. Fine for an internal/test deployment; put a
  reverse proxy (Traefik, Nginx, Caddy) in front before exposing this
  publicly.
- Web auth tokens are kept in browser localStorage with client-side fetch —
  acceptable for a test deployment, not hardened for production (no
  cookie/CSRF story yet).
