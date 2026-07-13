# Spotto — Weligama travel companion

Three platforms sharing one Go API:

| Directory | What | Stack | Port |
|---|---|---|---|
| `app/` | Users (traveller) mobile app | Flutter + Riverpod | — |
| `web/vendor/` | Vendor portal | Next.js + TS + Tailwind | 3000 |
| `web/admin/` | Admin command center | Next.js + TS + Tailwind | 3001 |
| `server/` | API | Go (chi + pgx) + PostgreSQL | 8080 |
| `Prototype/` | Original React JSX design prototype (reference only) | — | — |

## Run everything (4 terminals)

```powershell
# 0. Database — uses the existing shared-postgres container (localhost:5432)
docker exec shared-postgres psql -U postgres -c "CREATE DATABASE spotto" # first time only

# 1. API
cd server
go run ./cmd/seed   # first time / demo reset
go run ./cmd/api    # :8080

# 2. Vendor webapp
cd web
npm install         # first time
npm run dev -w vendor   # :3000

# 3. Admin webapp
npm run dev -w admin    # :3001

# 4. Traveller app (Android emulator "spotto")
cd app
flutter run  # API defaults to http://10.0.2.2:8080; override with --dart-define=API_URL=...
```

## Demo accounts (created by `cmd/seed`, password `spotto123`)

- Traveller: `lena@spotto.dev`
- Vendor (Driftwood Coffee, approved): `vendor@driftwood.dev`
- Admin: `admin@spotto.dev`

## The money demo

1. In the mobile app, sign in and claim Driftwood's deal → you get a `SPOTTO-XXXX` code.
2. In the vendor portal (`localhost:3000`) → Validate → enter the code → valid, marked redeemed.
3. Watch it appear in the dashboard's live redemption feed.
4. In the admin center (`localhost:3001`) → approve a pending vendor from the queue → that vendor's login can now access the portal.

## Notes

- Web auth tokens are kept in localStorage with client-side fetch — fine for local demos, not hardened for production (no cookie/CSRF story yet).
- Realtime is short-interval polling; SSE is the upgrade path if this heads to production.
