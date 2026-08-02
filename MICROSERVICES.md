# Microservice extraction candidates

This is a map, not a to-do list. It surveys the current Go backend (`server/`) for genuine
service-boundary candidates, ranks them by how safe/cheap each extraction would be, and — just as
importantly — identifies what should *stay* a monolith and why. Every claim below is backed by a
specific file, query, or table from this codebase, not generic microservices advice.

## Executive summary

**Current state**: one Go container (`api`), two Next.js containers (`vendor`, `admin`), one
externally-hosted Postgres — see `docker-compose.yml`. No CI/CD pipeline for the server (no
`.github/workflows` entry builds/deploys it), no Kubernetes, no message broker, no Redis. Deploys
are manual: redeploy the stack from Portainer, which pulls `main` and rebuilds (`PORTAINER.md`).
Solo-maintained.

**Why this matters for "should we split this up"**: the 2026 architecture consensus is consistent
— teams below ~10 engineers should stay monolithic unless they're facing a *specific, concrete*
pain (a measured scaling bottleneck, a real deploy-coordination problem across teams), not extract
services speculatively. Premature decomposition without that pain tends to produce a "distributed
monolith" — services that still share a database and still have to deploy together, which is worse
than what you started with. Nothing in this codebase today shows that pain. So: this document is a
map for *when* a concrete trigger shows up, ranked so you know which move is cheap/safe and which
is a real project.

**Ranked recommendation**:

| Tier | What | Cost/risk | Do it when |
|---|---|---|---|
| 0 | Split background schedulers into a second container (same DB, no new API) | Near zero | Thumbnail/CPU work visibly contends with API latency, or you want independent worker redeploys |
| 1a | Local Guide / AI chat service | Low | You want independent scaling/deploy for the chat feature specifically |
| 1b | External enrichment service (weather/holidays/place-details) | Low | Rarely urgent — already well-cached; do alongside 1a if ever |
| 1c | Media/thumbnail service (full version) | Medium | Only if Tier 0 alone isn't enough |
| 2 | BR-cert verification service | Medium-high (needs a schema prerequisite) | Not soon — low traffic feature today |
| — | Core domain (auth, vendors, places, reviews, claims, offers, admin) | **Don't** | Never, absent a team-scaling reason — see §5 |

---

## Tier 0 — process split, not a real microservice yet

**This isn't a network-boundary microservice at all**, and that's the point — it's the cheapest
possible move and the right first step if you ever feel pain here.

`cmd/api/main.go:30-33` starts four background schedulers as goroutines inside the API process:

```go
go srv.StartHolidayScheduler(ctx)   // internal/http/holiday_scheduler.go, every 12h
go srv.StartWeatherScheduler(ctx)   // internal/http/weather_scheduler.go, every 45m
go srv.StartBRCertScheduler(ctx)    // internal/http/br_cert_scheduler.go, every 2m
go srv.StartThumbnailWorker(ctx)    // internal/http/thumbnail_worker.go, every 10s
```

Every one of these already coordinates purely through Postgres columns (`thumb_status`,
`br_cert_ocr_status`, etc.) — not shared memory, not the in-process `internal/jobs` queue. That
means splitting them into a second binary costs **nothing in redesign**: add a `cmd/worker/main.go`
that starts these four goroutines instead of `cmd/api` doing it, point it at the same
`DATABASE_URL`, and run it as a second container from the same image (the Dockerfile already
builds 4 binaries from one module — `api`, `seed`, `gendata`, `importsrilanka` — a `worker` binary
is the same pattern).

- **What it buys you**: the CPU-bound thumbnail work (`internal/media`'s decode/resize) and
  request-serving stop contending for the same container's CPU; either can restart/redeploy
  independently of the other.
- **What it doesn't require**: no new API, no data-ownership change, no distributed transaction, no
  new credential.
- **Do it when**: you *observe* (not predict) request latency degrading during a thumbnail burst,
  or want to ship worker-only code changes without restarting the request path. Not urgent today —
  the sweep is already capped at 20 images/10s.

---

## Tier 1 — genuine microservice candidates (clean boundary, low risk)

### 1a. Local Guide / AI chat service
**Packages**: `internal/openrouter`, `internal/recommend`, `internal/http/guide.go`.

This is the cleanest candidate in the codebase. Of 27 tables, `guide_messages → guide_conversations`
is the *only* pair with a real, self-contained foreign key — every other table either has no FK at
all or references `vendors`/`users`. It owns exactly two tables, calls exactly one third-party API
(OpenRouter) that nothing else touches, and never writes into any other domain.

- **Data ownership**: `guide_conversations`, `guide_messages`.
- **API surface**: `POST /chat`, `GET /conversations/latest` (same shapes as today's
  `/guide/chat`, `/guide/conversations/latest`).
- **Cross-service dependency**: `recommend.Rank` needs read access to `places`. Simplest: call the
  main API's existing `GET /places` over HTTP — one extra network hop per chat turn, acceptable
  given chat already tolerates LLM latency. A read replica is overkill at this scale.
- **Credential**: `OPENROUTER_API_KEY` moves with it, fully decoupled from every other secret.
- **Trade-off**: that one extra network hop per turn, in exchange for independently scaling/
  deploying a feature with a fundamentally different traffic shape (bursty, LLM-latency-bound)
  than the rest of the API.

### 1b. External enrichment service
**Packages**: `internal/weather`, `internal/holidays`, `internal/placedetails`.

All three are read-through caches over third-party APIs (Tomorrow.io, Google Calendar, Google
Places), each backed by its own cache table (`weather_cache`, `place_details_cache`), with zero
writes into vendor/traveller/places tables.

- **Real wrinkle, worth naming rather than hiding**: `GoogleCalendarAPIKey` and
  `GooglePlacesAPIKey` are *the same key value* today
  (`server/internal/http/server.go:102` — `GooglePlacesAPIKey: cfg.GoogleCalendarAPIKey`), one
  Google Cloud project with Calendar + Places both enabled. Extracting this service means either
  keep sharing that one key (fine — same trust boundary either way) or provision a second
  restricted key. Recommend: keep sharing, don't add key-management overhead for no real gain.
- **API surface**: `GET /weather?lat=&lng=`, `GET /holidays?country=`, internal
  `GET /place-details/{googlePlaceId}`.
- **Data ownership**: `weather_cache`, `place_details_cache`.
- **Trade-off to resolve**: the holiday scheduler currently writes `vendor_notifications` directly
  into the `vendors` domain. Keep that write in the main API (this service just returns holiday
  data; the main API decides what to notify) rather than giving the enrichment service cross-domain
  write access — keeps it a clean, stateless-from-the-domain's-perspective read service.

### 1c. Media/thumbnail service (the full version)
Only worth doing if Tier 0's process split genuinely isn't enough — e.g. you want the media
pipeline on different hardware, or want upload handling itself (not just the async thumbnail
sweep) out of the main API.

- **Data ownership**: `media` table, R2 credentials (`R2_ACCOUNT_ID`/`R2_ACCESS_KEY_ID`/
  `R2_SECRET_ACCESS_KEY`/`R2_BUCKET`).
- **API surface**: `POST /media/presign`, `POST /media/confirm`, `DELETE /media/{id}`,
  `GET /media/raw/*` (the signed-URL read proxy) all move; the main API's vendor/place/review
  handlers call this service instead of touching `s.Store`/`s.Media` directly.
- **Trade-off**: `media.owner_type`/`owner_id` is a polymorphic reference into vendor/place/review/
  user rows. Today's delete-authorization check (`canUploadFor` in `media.go`) reaches directly
  into vendor/review ownership data to decide who can delete what. That logic has to move with the
  service or become a cross-service call — real work, not a config change. This is why it's 1c,
  not 1a.

---

## Tier 2 — possible, but needs a prerequisite refactor first

### BR-cert verification service
**Packages**: `internal/gemini`, `internal/eroc`, `internal/http/br_cert_scheduler.go`.

- **Why it's not Tier 1**: its entire state machine lives as seven columns bolted directly onto the
  `vendors` table (`br_cert_number`, `br_cert_ocr_status`, `br_cert_verify_status`, etc. — added in
  migration `018_br_cert.sql`), not a separate table. There's no clean data boundary to cut today.
- **Prerequisite** (worth doing on its own merits regardless of extraction): migrate those 7
  columns into a new `vendor_br_certs` table (`vendor_id` FK, one row per vendor) first — a small,
  low-risk, independently valuable schema cleanup.
- **Once that exists**: same shape as 1a/1b — own table, own external APIs (Gemini + eROC), own
  scheduler, API surface `POST /verify`, `GET /status/{vendorId}`, called by the main API's vendor
  handlers instead of writing those columns directly.
- **Priority note**: low-traffic feature today (one-time per vendor registration) — not worth rushing.

---

## Do not extract — the core domain

Auth/users, vendor management, places/traveller, reviews, claims/offers, admin. This isn't a
default "keep everything together" hedge — it's backed by direct evidence from your own code:

- **`syncVendorPlace`** (`server/internal/http/vendor.go`, called after every vendor profile/menu/
  hours/identity mutation) is a synchronous read-then-write spanning vendor → places/traveller data
  in the same request. Across a service boundary this becomes either a blocking cross-service call
  on every vendor edit, or accepted eventual consistency on what travellers see on the map — a real
  regression for no real gain.
- **`VendorStats`/`VendorSummary`** (`server/internal/store/postgres/store.go:856+`) is a single SQL
  statement with roughly 14 correlated subqueries across `claims`, `checkins ⋈ places`, `reviews`,
  `offers`, `users`, `vendor_invites`, plus a `LATERAL` join to `subscriptions` — this is the admin
  dashboard's core query. Splitting the underlying tables across services turns this into a fan-out
  of a dozen-plus network calls per dashboard load, or forces a shared reporting store — exactly
  the "distributed monolith" failure mode the research above warns is the #1 premature-extraction
  mistake.
- **`AdminSearch`** similarly spans vendors/feed_items/subscriptions in one function.
- **Schema note that actually helps here**: only 5 of 27 tables have a real SQL `REFERENCES`
  constraint; most cross-domain relationships (`places.vendor_id`, `reviews.{place_id,vendor_id}`,
  `claims.{place_id,vendor_id,traveller_id}`, `media.owner_id`, etc.) are bare TEXT columns tied
  together by application code. That's good news for the candidates above (no FK cascade chains to
  unwind) — but it also means Postgres itself isn't enforcing the referential integrity these
  cross-domain admin queries depend on today; splitting them apart would make that worse, not
  better.

**Net**: this is one team, one deploy cadence, with reads that are tightly coupled *because the
product needs them answered together* — an admin looking at a vendor needs claims + reviews +
offers + team + subscription state in one glance. Keep it one deployable. This is what "modular
monolith" means in practice: the existing `internal/` package boundaries (auth, vendor-handling
code, traveller-handling code) already give you the organizational clarity without paying a
network/consistency cost for it.
