# Handover — 2026-08-02 session

## Why this exists

Running log of what got done in this session and what's still open, so the next session (or
you) can pick up without re-deriving context. Supersedes the previous `HANDOVER.md`
(2026-07-19, Admin Command Center audit + Validate QR scanning) — that doc's still-open items
are carried forward at the bottom with updated status.

---

## Done this session (chronological)

### 1. Vendor registration polish + pending-vendor dashboard access
**Commit `f9f814a`.**
- Fixed Step 4's misaligned phone-number row (the `LK +94` country-code chip had no label
  above it, so it didn't line up with the `Field` component next to it).
- Added a location pin picker to Step 4 (`LocationPickerAuto`, new component) so owners who
  skip the Step 1 business search still get to set an exact map pin before submitting.
- `Shell` (`web/vendor/components/shell.tsx`) no longer hard-blocks the *entire* dashboard for
  pending/edits-status vendors — only `rejected`/`suspended` get the full-page gate now.
  Pending/edits vendors get the real dashboard plus a persistent `StatusBanner`, so they can
  set up their profile/menu while awaiting approval instead of sitting idle. Offers/Growth/
  Validate/Reports stay hidden from nav until approved (`approvedOnly` flag on `NAV`).
- Backend: new `requireEditableVendor` (`server/internal/http/auth.go`) allows pending/edits/
  approved vendors to write to profile/identity/category/menu endpoints — `requireApprovedVendor`
  stays as-is for offers/growth/validate/reports, which depend on actually being live to
  travelers.
- Hardened the Google Maps location picker: `gm_authFailure` (Google's own hook for
  credential/referrer failures) now triggers a runtime fallback to the free Leaflet/
  OpenStreetMap picker instead of leaving Google's cryptic "Oops!" overlay on screen
  (`lib/google-maps.ts`, new `location-picker-auto.tsx`).
- **Follow-up mid-session:** the actual root cause of the Google Maps failure on staging was
  the Maps API key's HTTP-referrer restriction not covering the staging domain — fixed on your
  end in Google Cloud Console, confirmed working by a later screenshot showing a live Google
  map.

### 2. Address search + auto-fill from Google Place data
**Commit `a056fe6`.**
- "Town" became a real address-search field (`AddressSearchField`, new component) in both
  registration Step 4 and the vendor Profile page — searching resolves an approximate
  location and pans the map there, while still storing just the short town name (not the full
  street address) so admin lists/dashboard copy that expect a short label keep working.
- If the matched result is an actual Google Maps business listing, Phone/Website/Category/
  Hours get pulled in automatically — but **only into fields that are still empty**, never
  overwriting something already typed, with no confirmation UI (silent fill).
- Removed the old floating search box that used to sit on top of the Google map itself, since
  the new Address field is the single search entry point.
- Fixed a real CSS cascade bug found along the way: the Contact & links card's icon-overlapping-
  text (phone/globe/Instagram/Facebook icons sat on top of the placeholder text) — `.input`'s
  shorthand `padding` in `globals.css` was declared *after* `@tailwind utilities`, silently
  overriding the `pl-9` padding those fields need. Fixed by wrapping the custom classes in
  `@layer components {}`.
- Added a label above the Profile page's map to match Step 4's.

### 3. Current Vibe redesign + optimistic updates
**Commits `285a72c`, `eaacf82`.**
- Restyled to 3 side-by-side colored cards (your requested "old card look") with a 5-segment
  tap strip per card (each segment = 2 points on the underlying 0-10 scale) instead of 3
  stacked full-width 10-segment bars — keeps all the check-in logic (geofencing, repeat
  ratings, timestamps) exactly as it was.
- Made rating **optimistic**: `VibeCheckInsNotifier.rate()`
  (`app/lib/data/repositories/repositories.dart`) used to only update local state *after* the
  network round-trip, so a tap produced zero visible change until the request finished. Now it
  writes your tap into state immediately, reconciles with the server's response on success, and
  rolls back on failure.
- **This session's follow-up:** confirmed via your API logs the vibe-checkin POST itself is
  ~3-4ms, so the remaining "hot second" people might feel is GPS/location-fix latency
  (`devicePositionProvider`), not the tap path. Fixed the UX symptom: the card now shows
  "Finding your location…" while GPS is still resolving, instead of the misleading "Rate this
  once you're here." (which is supposed to mean "you're too far away") — see commit `cd242c2`.
  The underlying GPS wait itself is real hardware/OS latency (worse on emulators, which often
  lack a cached last-known position) and isn't something fixable in app code.

### 4. Traveller-photo UX overhaul + a real authorization fix
**Commit `285a72c`.**
- Traveller photos moved from a horizontal scroll strip to a grid (nothing hidden off-screen),
  with loading placeholders, an empty state, a relative "posted X ago" timestamp per photo, and
  a delete control on photos you posted yourself.
- Review photo thumbnails got the same sizing/loading-placeholder treatment for consistency.
- The full-screen photo viewer gained swipe-down-to-dismiss.
- **Found and fixed a real security gap** while wiring up delete: `canUploadFor`'s blanket
  `true` for `ownerType == "place"` is correct for *uploads* (any traveller can contribute a
  photo to a public place) but was also being reused to authorize *deletes* — meaning any
  signed-in traveller could delete any other traveller's place photo, not just their own.
  `handleMediaDelete` now requires you to be the uploader (or an admin) for place-owned media.

### 5. Debounced auto-save everywhere it's safe
**Commit `bf70b98`.**
New shared `useAutoSave` hook (`web/packages/shared/src/use-auto-save.ts`) and a Flutter
`DebouncedSaver` utility (`app/lib/core/utils/debounced_saver.dart`) — both follow the same
rule: **schedule the save from a `useEffect`/`Timer`, never synchronously right after a state
update.** That rule exists specifically to avoid a real historical bug: an earlier version of
the vendor Menu page called its save mutation synchronously after `setPresets(...)`, which
raced React's state batching and silently persisted the *previous* state. `flush()` forces an
immediate save on blur, and is what the existing manual "Save" buttons now call — kept as a
fallback, not removed.

Wired into:
- **Vendor portal**: Business Profile (identity/address, contact, hours, amenities), Account
  (personal details), Menu, and Offers — but Offers only auto-saves while editing an *existing*
  offer, and preserves its current live/draft status rather than reusing the Publish/Save-as-
  draft buttons' explicit status (so a typo fix never silently un-publishes a live offer).
- **Admin portal**: the vendor `EditForm` — auto-save no longer closes the form back to view
  mode; only the manual "Save changes" button does that.
- **Traveller app**: Profile screen's name field and style/interest chips (email turned out to
  already be read-only, so left alone).

**Deliberately excluded everywhere:** passwords, 2FA, destructive admin actions (takedown,
approve/reject), Cancel-able dialogs (trip create/edit sheets), and offer/guide *creation*
flows (as opposed to editing).

### 6. Upload error clarity + R2 CORS diagnosis
**Commit `466a996`.**
Vendor Photos & video uploads were failing with a bare "Failed to fetch" — traced to
`uploadMedia` (`web/packages/shared/src/upload.ts`) PUTting directly from the browser to
Cloudflare R2, which is subject to R2's own CORS policy (separate from the API server's CORS
config). Added a clearer error message for this specific failure mode. **The actual root cause
was your R2 bucket's CORS policy only allowing `GET`** — you added `PUT` + `Content-Type` to
the allowed methods/headers on the bucket directly (not a code change); should be resolved now
but worth double-checking a real upload succeeds on staging if you haven't already.
**Later:** the Command Center uploads too (audio guides, covers and photos in content posts,
the admin vendor media panel), so its origin must be in the bucket's `AllowedOrigins` as well
as the vendor site's (see `server/.env.example`); the error now names the origin to add.

### 7. Real server-side photo thumbnails
**Commit `cd242c2`.**
The Go backend now generates a small (~320px) JPEG thumbnail for every image upload:
- New `golang.org/x/image` dependency; new migration `017_media_thumbnails.sql` adds
  `thumb_key` to the `media` table; new `ThumbKey`/`ThumbURL` fields on `domain.Media`.
- `handleMediaConfirm` fetches the just-uploaded object, decodes/resizes it, and stores the
  derivative via a new `Put` method on the R2 `Uploader` interface (previously only
  `PresignPut` existed, for client-direct uploads) — failing gracefully (no thumbnail, not a
  failed upload) if the source image can't be decoded.
- Thumb URLs are signed the same way originals are, and cleaned up on delete/replace alongside
  the original object.
- **Consumed so far:** the traveller photos grid and the vendor Profile gallery load the
  thumbnail for tiles, falling back to the full `url` for photos uploaded before this existed;
  the full-screen viewer always uses the full-resolution image.
- **Not yet wired up (same backend capability, cheap to add later):** review photos
  (`Review.photos` is currently a flat `List<String>` from a different response shape than
  `Media`, needs its own small model/response change) and the admin vendor media panel.

---

## What's still open

An audit pass this session went back over the 5 items below (the previous version of this
section), verified each claim against the current code, and found several were stale or
understated a real bug. Fixes landed for everything except item 1's staging confirmation, which
only you can do.

1. **Verify the R2 CORS fix actually works** — you applied it manually in the Cloudflare
   dashboard; do one real upload on staging to confirm. Audited the upload path while at it:
   the CORS fix itself is complete (every web upload funnels through the one shared
   `uploadMedia()` helper, so nothing else needed it) — but found and fixed an adjacent bug:
   `PresignPut` URLs expire after 10 minutes, so a slow/large upload could outlive the link and
   get a bare `Upload failed (403)`; `uploadMedia()` now shows a clear "link expired, try again"
   message for that case instead.
2. **Fixed: review photos now carry thumbnails.** The old note here ("different response
   shape than `Media`, needs its own model") was wrong — review photos already lived in the
   same `media` table (`ownerType='review'`) and were already getting thumbnails generated on
   upload, just never selected/returned. `ReviewPhotos` now selects `thumb_key`,
   `domain.Review.Photos` is `[]ReviewPhoto{URL, ThumbURL}` instead of `[]string`, and the
   Flutter review-photo carousel (`business_drawer.dart`) uses the thumbnail for the strip and
   the full image only in the full-screen viewer.
3. **Fixed: admin vendor media panel now uses thumbnails.** Was a literal one-line gap —
   `MediaThumb` always rendered `m.url` instead of `m.thumbUrl || m.url`, unlike the vendor
   portal's own photo grid, which already did this correctly. Every vendor detail page load in
   admin was pulling full-resolution originals (up to 8MB each) to paint 64×64px tiles — fixed.
4. **Fixed: two real bugs found behind the "GPS is just slow" framing for Current Vibe.** The
   "Finding your location…" timing itself was always fine (bounded, never hangs) — but (a)
   permission-denied and location-services-disabled were silently mapped to a fake fallback
   position (the app's fallback point) and shown the same "too far away" message forever, with no way to
   fix it — now shows an actionable "enable location" prompt (tap → Settings or retry),
   matching the pattern `discover_screen.dart`/`spots_screen.dart` already used elsewhere in the
   app; and (b) the resolved position was cached for the whole app session with no re-check —
   now re-checks once each time a place's drawer opens, so walking into range unlocks the cards
   without an app restart. Also closed a related latent bug this surfaced: the geofence/check-in
   lat-lng could previously be computed against the fallback point for a denied/disabled
   user, not just a real position.
5. Carried forward from the 2026-07-19 handover:
   - **Corrected, not a bug:** `web/vendor/app/account/page.tsx`'s "Verified" badge check
     (`business?.status === 'approved'`) was previously described as unreachable-but-fragile.
     That was already stale before this session even started — this session's `Shell` change
     (commit `f9f814a`) lets pending/edits vendors reach the real dashboard including
     `/account`, so the check is genuinely live code today, and it behaves correctly (no
     badge for non-approved, badge for approved). Nothing to fix.
   - Real admin RBAC + an admin-invite system, and a CMS for the public Spotto marketing
     website — both still deliberately deferred by your choice, not forgotten.
