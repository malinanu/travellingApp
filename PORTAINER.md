# Deploying Spotto on Portainer

Three services — `api` (Go, container port 8080, published on the host as
`8881`), `vendor` (Next.js, :3000), `admin` (Next.js, :3001) — built from
`docker-compose.yml` at the repo root. This stack does **not** run its own
Postgres; it connects to one you already have reachable from the Docker
host.

(The api's host port was moved from 8080 → 18080 → 8881 as the deployment
host's port assignments evolved — the container still listens on 8080
internally, only the host-side published port changed.)

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
into Portainer's stack environment editor**, not commit it. If you're
mapping domains through CloudPanel (see step 4a below), use your real
subdomains here, over HTTPS, not the bare IP:

```
DATABASE_URL=postgres://postgres:<password>@<postgres-host>:5432/spotto?sslmode=disable
JWT_SECRET=<a long random string>
TOTP_ENCRYPTION_KEY=<output of: openssl rand -base64 32>
CORS_ORIGINS=https://spotto-vendor.yourdomain.com,https://spotto-admin.yourdomain.com
NEXT_PUBLIC_API_URL=https://spotto-api.yourdomain.com
```

`TOTP_ENCRYPTION_KEY` must be exactly 32 raw bytes, base64-encoded — the api
container refuses to start without one (used to encrypt 2FA secrets at
rest).

`NEXT_PUBLIC_API_URL` is baked into the vendor/admin webapps **at build
time** (Next.js public env vars aren't runtime-configurable) — if this
value ever changes, the stack needs a rebuild, not just a restart (see the
"Updating" section below).

If your Postgres runs as another container on the same Docker host (not via
Portainer's own network), `<postgres-host>` can usually be that container's
name if it's on a shared Docker network, or `host.docker.internal` if it's
bound to the host's own port.

**Port check**: this stack publishes `8881` (api), `3000` (vendor), `3001`
(admin) on the host. If any of these ever collide with something else
already running on the box, change the host-side number in
`docker-compose.yml`'s `ports:` mapping (left side of the `:` — leave the
right side alone, that's the container's internal port).

## 4. Deploy in Portainer

**Stacks → Add stack → Repository**:
- Repository URL: `https://github.com/malinanu/travellingApp.git`
- Compose path: `docker-compose.yml`
- Environment variables: paste the contents of your filled-in `.env`
  (one `KEY=value` per line, using Portainer's environment editor)
- **Deploy the stack**

This builds all three images on the Docker host and starts them.

## 4a. Domain mapping via CloudPanel (reverse proxy)

CloudPanel runs its own Nginx directly on the host (ports 80/443) — it
isn't part of the Docker stack, so nothing above needs to change. Point
CloudPanel at the three local ports with one **Reverse Proxy** site per
subdomain:

For each of the three (repeat with the matching port):

| Subdomain | Reverse proxy target |
|---|---|
| `spotto-api.yourdomain.com` | `http://127.0.0.1:8881` |
| `spotto-vendor.yourdomain.com` | `http://127.0.0.1:3000` |
| `spotto-admin.yourdomain.com` | `http://127.0.0.1:3001` |

In CloudPanel: **Sites → Add Site → Reverse Proxy** → enter the domain →
set the reverse-proxy target to the `127.0.0.1:<port>` above → save → then
use CloudPanel's built-in **Let's Encrypt** button on that site to issue a
free SSL cert. Repeat for all three subdomains.

Point each subdomain's DNS (A record) at this server's IP before requesting
the certificate, same as any other CloudPanel site.

No websocket/long-lived-connection config is needed — this app only uses
short HTTP polling, not websockets, so CloudPanel's default reverse-proxy
settings work as-is.

Once all three subdomains resolve over HTTPS, go back to step 3 and make
sure `NEXT_PUBLIC_API_URL`/`CORS_ORIGINS` use those `https://` domains
(rebuild if you change them after the first deploy).

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

- `https://spotto-api.yourdomain.com/health` → `{"status":"ok"}`
- `https://spotto-vendor.yourdomain.com/login` → vendor portal
- `https://spotto-admin.yourdomain.com/login` → admin command center

(Or `http://<server-ip>:8881/health` etc. directly, before/without
CloudPanel in front — useful for isolating whether an issue is in the
container or in the reverse proxy.)

## Updating after a config change

- **Code change**: re-deploy the stack from Portainer (pulls latest `main`,
  rebuilds).
- **`NEXT_PUBLIC_API_URL` or any build-time value changed**: rebuild the
  affected service images, not just restart —
  `docker compose build vendor admin && docker compose up -d`.
- **`DATABASE_URL`/`JWT_SECRET`/`CORS_ORIGINS` changed**: these are
  runtime env vars for `api` only — a restart is enough.

## Notes / limitations

- The stack itself has no TLS — that's handled by CloudPanel's reverse
  proxy + Let's Encrypt in front of it (see step 4a). If you skip CloudPanel
  and hit the ports directly, it's plain HTTP.
- Web auth tokens are kept in browser localStorage with client-side fetch —
  acceptable for a test deployment, not hardened for production (no
  cookie/CSRF story yet).
