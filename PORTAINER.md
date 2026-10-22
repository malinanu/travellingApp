# Deploying Spotto on Portainer

Three services — `api` (Go, container port `8881`, published on the host as
`8881` too), `vendor` (Next.js, :3000), `admin` (Next.js, :3001) — built from
`docker-compose.yml` at the repo root. This stack does **not** run its own
Postgres; it connects to one you already have reachable from the Docker
host.

(The api's port was moved from 8080 → 18080 → 8881 as the deployment host's
port assignments evolved — host port 8080 is taken by another site on this
box. Host and container ports are now the same number, 8881, to keep things
simple; the `PORT` env var in `docker-compose.yml` and the `EXPOSE` line in
`server/Dockerfile` must be kept in sync if this ever changes again.)

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
ADMIN_EMAIL=<your admin login email>
ADMIN_PASSWORD=<a long random password, 12+ characters>
APP_ENV=development   # set to production on the live deployment

# Optional — see .env.example for the full list (SMTP, Google OAuth, and
# the two vars below). Blank is fine; each feature just no-ops until filled in.
GOOGLE_CALENDAR_API_KEY=<your Google Cloud API key, Calendar API enabled>
NEXT_PUBLIC_GOOGLE_MAPS_API_KEY=<same key works, with Maps JS API + Places API also enabled>
GOOGLE_PLACES_API_KEY=<Google key with Places API (New) enabled, for the Inspiration Board; falls back to GOOGLE_CALENDAR_API_KEY>
```

`ADMIN_EMAIL`/`ADMIN_PASSWORD` define the admin login. The api creates
that account, or resets its role and password to match, on every start,
so change them here and redeploy to rotate. `APP_ENV=production` locks
every demo/import login (`*@spotto.dev`, except `ADMIN_EMAIL`) at boot
and makes `/app/seed` and `/app/gendata` refuse to run. Only set it on
the live deployment, because staging still relies on the demo accounts.

`TOTP_ENCRYPTION_KEY` must be exactly 32 raw bytes, base64-encoded — the api
container refuses to start without one (used to encrypt 2FA secrets at
rest).

Two different kinds of "optional" here, by how they take effect:
- **Build-time** (`NEXT_PUBLIC_API_URL`, `NEXT_PUBLIC_GOOGLE_MAPS_API_KEY`):
  baked into the vendor/admin webapps' JS bundle when the image is built
  (Next.js public env vars aren't runtime-configurable) — if either value
  is added or changes, the stack needs a **rebuild** of that image, not just
  a restart (see the "Updating" section below). Forgetting this is exactly
  why adding a Maps key to Portainer's environment editor alone doesn't
  make the map picker work — the vendor image needs an actual rebuild to
  pick it up.
- **Runtime** (everything else — `GOOGLE_CALENDAR_API_KEY`, the SMTP vars,
  the Google OAuth vars, `VENDOR_APP_URL`, `PUBLIC_API_URL`): read by the `api` container each
  time it starts — a plain restart after changing these is enough.

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

## 5. Load demo or test data (first run only)

From Portainer's console for the `api` container, or via the CLI on the
host — two options, either wipes and reloads all tables:

```
docker compose run --rm api /app/seed
```
A small, fixed demo dataset — accounts `lena@spotto.dev`,
`vendor@driftwood.dev`, `admin@spotto.dev`, password `spotto123` for all.
Safe to re-run any time you want a clean, predictable demo state.

```
docker compose run --rm api /app/gendata
```
A larger, randomized dataset for manual testing (~40 travellers, ~18
vendors/places, ~250 reviews, offers, claims, groups, feed items, etc.) —
every run generates different names, businesses, and review text. Prints
a handful of ready-to-use sample logins (all sharing one password) to the
command output at the end, since the emails are random each run.

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
- **`NEXT_PUBLIC_API_URL` or `NEXT_PUBLIC_GOOGLE_MAPS_API_KEY` changed**:
  rebuild the affected service images, not just restart —
  `docker compose build vendor admin && docker compose up -d`.
- **`DATABASE_URL`/`JWT_SECRET`/`CORS_ORIGINS`/`GOOGLE_CALENDAR_API_KEY`/
  SMTP/Google-OAuth vars changed**: these are runtime env vars for `api`
  only — a restart is enough, e.g. `docker compose up -d` (no `build`).

## Notes / limitations

- The stack itself has no TLS — that's handled by CloudPanel's reverse
  proxy + Let's Encrypt in front of it (see step 4a). If you skip CloudPanel
  and hit the ports directly, it's plain HTTP.
- Web auth tokens are kept in browser localStorage with client-side fetch —
  acceptable for a test deployment, not hardened for production (no
  cookie/CSRF story yet).
