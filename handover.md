# Handover — 2026-07-19 session

## Why this exists

Running log of what got done in this session and what's still open, so the
next session (or you) can pick up without re-deriving context. Supersedes
the previous `handover.md` (2026-07-16, latent-bug audit) — that audit's
findings are carried forward below with updated status (two of its four
items got fixed as a side effect of this session's other work).

---

## Done this session (chronological)

### 1. Google Places API migration
`web/vendor/components/location-picker-google.tsx` and `web/vendor/app/login/page.tsx`
migrated from the deprecated `google.maps.places.Autocomplete` widget (blocked
for new Google Cloud projects since March 2025 — the real cause of a
persisting "must use an API key" error in production) to the modern
`PlaceAutocompleteElement`. Verified locally with the real API key; the
remaining production-side unknown is whether "Places API (New)" is enabled
in Google Cloud Console's key restrictions.
**Committed:** `360d636`.

### 2. Hydration-mismatch fixes (unrelated, opportunistic)
Browser-extension-injected attributes (Grammarly's `data-gr-ext-installed`
on the vendor app's `<body>`, a password manager's `caret-color` style on
the admin login page's inputs) were causing harmless but noisy React
hydration-mismatch console errors. Fixed with targeted `suppressHydrationWarning`
on the specific elements in `web/vendor/app/layout.tsx` and
`web/admin/app/login/page.tsx`.
**Committed:** bundled into `360d636` (vendor) and `816136e` (admin login
page's fix — check `git show --stat` if you need the exact split).

### 3. Admin Command Center — full audit + 5 fixes
Audited `web/admin` end-to-end. Good news: all 6 core sections (Approval
queue, Content monitor, Geo-analytics, Subscriptions, Audio guides, Access
grants) were already real, DB-backed, no mock data. Gaps were all in header
chrome and admin-account infrastructure:
- **Audit log** — new `admin_audit_log` table + write on every admin
  mutation (approve/reject/suspend a vendor, moderate content, revoke a
  grant, upgrade a subscription), gated to super-admins only
  (`server/internal/http/audit.go`, migration `010_admin_audit_log.sql`).
  Also fixed a real bug: grant revocation was recording the literal string
  `"admin"` instead of the acting admin's real ID.
- **Real notification bell** — computed live from pending-vendor/flagged-
  content/bad-review-vendor counts (no drift risk vs. a stored-event table),
  replacing a completely dead `<Bell/>` button
  (`server/internal/store/postgres/store.go`'s `AdminNotifications`,
  migration `011_admin_notification_reads.sql`).
- **Real search (⌘K)** — searches vendors/feed items/subscriptions; the
  search box and the `⌘K` keybinding were both previously decorative dead
  UI (`GET /admin/search`, wired into `web/admin/components/shell.tsx`).
- **Super-admin tier via `SUPER_ADMIN_EMAILS` env var** — deliberately
  lightweight per your explicit choice over building real DB-backed
  RBAC/invites. The header's hardcoded "Super-admin" text is now real
  (`server/internal/http/superadmin.go`).
- **Polish** — forgot-password flow added to the admin login page (plus a
  real bug fix: `handleForgotPassword` was emailing admins a reset link
  into the *vendor* app, not the admin app), the previously-dead "Change
  password" button now works, error/loading states added across every
  admin route.
**Committed & pushed:** `14ec713`.

**Explicitly deferred (your call, not forgotten):** real multi-tier admin
RBAC + an admin-invite system (you chose the env-var approach instead), and
a CMS for the public Spotto marketing website (a separate future
initiative you mentioned in passing — no requirements gathered yet, saved
as a memory note).

### 4. Vendor Menu — preset menus + per-day availability + real bug fix
Root cause of your "can't add a menu" report: the save mutation auto-fired
immediately after adding a section, but read React's *pre-update* state due
to batching — it silently persisted the previous menu, so a newly-added
section appeared to save then vanished on the next refetch. Fixed by
removing the auto-save entirely; every edit now just marks the document
dirty, and the existing "Save menu" button (reads current state) is the
only thing that persists. Verified via Playwright that a new section+item
now survives a full page reload.

Also built the requested feature: vendors can create multiple named preset
menus and manually pick which one is active; any section/item can be
restricted to specific days of the week (whitelist semantics — unrestricted
by default). All resolved server-side in the existing menu-flattening logic
(`effectiveMenuSections` in `server/internal/http/vendor.go`) — **zero
Flutter changes needed**, confirmed live (an item restricted to Monday
correctly disappeared from the real `/places` response on a Sunday).
6 new Go integration tests (`vendor_menu_test.go`), all passing.
**Committed & pushed:** `816136e`.

### 5. Validate (redemption) page — real camera QR scanning
- New `web/vendor/components/qr-scanner.tsx`: real `getUserMedia` camera
  (rear camera preferred) decoding QR frames via `jsQR` on a canvas —
  replaces what was a fully static/fake "Simulate scan" placeholder.
  Self-contained requesting/active/denied/unavailable states.
- `web/vendor/app/validate/page.tsx` wired to the real scanner; manual
  entry + backend (`POST /vendor/validate`) were already fully real, no
  backend changes needed there.
- **Found a real gap while researching this:** nothing anywhere in the app
  actually generated a QR code to scan — the Flutter traveller app only
  showed the redemption code as plain text. Added a real `QrImageView` to
  the claimed-deal card (`app/lib/features/traveller/place_detail/business_drawer.dart`)
  using the `qr_flutter` package (already a dependency, already used for
  2FA) so the loop is genuinely closed end-to-end.
- **Found and fixed a second real, pre-existing bug** while testing mobile
  responsiveness: the vendor portal's shared mobile nav drawer
  (`web/vendor/components/shell.tsx`) had no CSS for its *closed* state —
  it only ever got `translateX(0)` when open, otherwise fell through to no
  transform at all, so the drawer was permanently stuck open, covering page
  content on **every** vendor page at phone-width viewports, not just
  Validate. One-line fix (`-translate-x-full lg:translate-x-0`), verified
  with real before/after mobile screenshots.
- Verified: `jsQR` correctly decodes a real generated Spotto QR end-to-end
  (via a canvas + local HTTP test harness, since no camera hardware/ffmpeg
  was available in this environment to fake a full camera→QR pipeline);
  camera-granted (Chromium fake device), camera-denied, and mobile-viewport
  Playwright passes all clean; `flutter analyze` and `next build`/`tsc`
  both clean.
- **NOT YET COMMITTED.** Touches: `app/lib/features/traveller/place_detail/business_drawer.dart`,
  `web/vendor/app/validate/page.tsx`, `web/vendor/components/shell.tsx`,
  `web/vendor/components/qr-scanner.tsx` (new), `web/vendor/package.json`
  + `web/package-lock.json` (new `jsqr` dependency).

---

## What's still open

1. ~~Push the Validate redemption work (item 5 above)~~ — **done**,
   committed (`305f418`) and pushed with your go-ahead.
2. **Production Google Places check** — after your next vendor-app
   redeploy, confirm the "must use an API key" error is actually gone; if
   not, check "Places API (New)" is allowed under the key's API
   restrictions in Google Cloud Console.
3. ~~Old Issue 1 (suspended vendors see the wrong message)~~ — **fixed**:
   `PendingGate` in `web/vendor/components/shell.tsx` now has a
   `'suspended'` branch ("Account suspended" + contact-support copy,
   mirroring the `'rejected'` branch) instead of falling through to the
   misleading "Pending review… usually within a day" text.
4. **Old minor note 3, still applicable — `web/vendor/app/account/page.tsx`'s**
   "Verified" badge check (`business?.status === 'approved'`) is
   unreachable-but-fragile: `Shell` already gates non-approved vendors away
   before this page renders, so the check is always true today. Not a bug,
   just worth remembering if that gating is ever loosened.
5. ~~Old Issue 2 (admin "Change password" does nothing)~~ — **fixed**, see
   item 3 above (Admin Command Center polish).
6. ~~Old minor note 4 (admin "Super-admin" hardcoded)~~ — **fixed**, see
   item 3 above (super-admin tier).
7. **Deferred by your choice, not forgotten:** real admin RBAC + invite
   system; a CMS for the Spotto marketing website (no requirements
   gathered yet).

## Methodology notes carried forward (still valid if repeating a similar audit)

1. List every `enum`/status/role/category type in
   `server/internal/domain/domain.go`.
2. Grep the frontend (`web/**/*.tsx`, `app/lib/**/*.dart`) for string
   literals matching those values, and check every comparison branch
   handles every value the type can actually take — not just the ones a
   demo dataset happens to exercise.
3. Cross-check what `server/internal/seed/seed.go` and
   `server/internal/gendata/generators.go` actually generate for each
   field, since those (not just `domain.go`'s type definition) are what
   real accounts will actually have.
4. Grep every `api.<method>(` call site against
   `server/internal/http/server.go`'s registered routes.
