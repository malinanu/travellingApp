# Spotto — Weligama travel companion

Three platforms sharing one Go API:

| Directory | What | Stack | Port |
|---|---|---|---|
| `app/` | Users (traveller) mobile app | Flutter + Riverpod | — |
| `web/vendor/` | Vendor portal | Next.js + TS + Tailwind | 3000 |
| `web/admin/` | Admin command center | Next.js + TS + Tailwind | 3001 |
| `server/` | API | Go (chi + pgx) + PostgreSQL | 8080 |
| `Prototype/` | Original React JSX design prototype (reference only) | — | — |

## Quick start

```powershell
.\run-all.ps1            # starts Postgres check, API, vendor webapp, admin webapp
.\run-all.ps1 -SkipSeed  # keep existing DB data instead of reseeding
.\run-all.ps1 -App       # also launches the Flutter traveller app
```

Each service opens in its own PowerShell window so you can see its logs; the
script waits on health checks and prints a summary with URLs and demo
accounts when everything's up. See below for what it automates, or to run
things manually one at a time.

## Run everything (4 terminals)

```powershell
# 0. Database — uses the existing shared-postgres container (localhost:5432)
docker exec shared-postgres psql -U postgres -c "CREATE DATABASE spotto" # first time only

# 1. API (runs migrations automatically on startup)
cd server
copy .env.example .env   # first time — edit if your Postgres differs
go run ./cmd/seed        # first time / demo reset — wipes & reloads demo data
go run ./cmd/api         # :8080

# 2. Vendor webapp
cd web
npm install              # first time (installs all 4 workspaces)
npm run dev:vendor       # :3000

# 3. Admin webapp
npm run dev:admin        # :3001

# 4. Traveller app (Android emulator "spotto")
cd app
flutter pub get          # first time
flutter run               # API defaults to http://10.0.2.2:8080; override with --dart-define=API_URL=...
```

## Demo accounts (created by `cmd/seed`, password `spotto123`)

- Traveller: `lena@spotto.dev`
- Vendor (Driftwood Coffee, approved): `vendor@driftwood.dev`
- Admin: `admin@spotto.dev`

## The money demo

1. In the mobile app, sign in as `lena@spotto.dev` and claim Driftwood's deal
   ("Free flat white with any day pass") → you get a fresh `SPOTTO-XXXX` code
   (the seed also pre-creates `SPOTTO-DRIF` as an already-claimed code for
   the vendor portal demo below, in case the emulator isn't running).
2. In the vendor portal (`localhost:3000`, sign in as `vendor@driftwood.dev`)
   → **Validate** → enter the code → shows valid, marked redeemed. Enter it
   again → "already redeemed".
3. Watch it appear in the dashboard's live redemption feed (3s poll).
4. In the admin center (`localhost:3001`, sign in as `admin@spotto.dev`,
   any 6 digits for 2FA) → **Approval queue** → approve a pending vendor
   (hotkey `a`, with a 6s undo toast) → that vendor's login can now reach
   the vendor portal instead of the "pending review" gate.
5. Also live in the admin center: **Content monitor** streams a new
   templated feed item every ~3s (server-side generator); **Geo-analytics**
   shows the demand/supply map with pulsing gap rings; toggle light/dark
   theme from the command bar.

## Notes

- Web auth tokens are kept in localStorage with client-side fetch — fine for
  local demos, not hardened for production (no cookie/CSRF story yet).
- Realtime is short-interval polling (3–5s); SSE is the natural upgrade path
  if this heads to production.
- Claim codes are server-generated, single-use, and vendor-scoped — re-using
  one returns `already_redeemed` with the original redemption timestamp.
- `go test ./...` includes an integration test for the validate endpoint's
  edge cases (unknown / wrong-vendor / already-redeemed codes); it skips
  itself automatically if no local Postgres is reachable.
