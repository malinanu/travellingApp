# Spotto — Build Prompt for Claude Code (Vanilla HTML / CSS / JS)

Paste everything below into Claude Code. It is a complete functional spec for a mobile-first
travel-companion prototype called **Spotto**, set in Weligama, Sri Lanka. Build it with **plain
HTML, CSS, and vanilla JavaScript only** — no React, no build step, no frameworks. The whole app
must run by opening `index.html` in a browser.

---

## 0. What you are building

Spotto is a single-page mobile app prototype that helps a traveller discover places around
Weligama Bay. It renders inside a **phone frame centered on the screen** (letterboxed on a dark
background, like a device mockup). It has an onboarding flow, a sign-in screen, a bottom tab bar
with 3 tabs, a floating AI-guide button, and several bottom sheets / modals.

The app is a **prototype**: all data is hard-coded in JS, there is no backend. Interactions update
in-memory state and re-render. Persist a few things to `localStorage` so a refresh keeps state.

---

## 1. Tech constraints

- **Vanilla only**: HTML + CSS + JS. No React/Vue/Tailwind/bundlers.
- Structure: one `index.html`, one `styles.css`, and JS split into small files
  (`data.js`, `app.js`, plus one file per screen if helpful) loaded with `<script>` tags.
- Rendering: keep a single global `state` object and a `render()` function that rebuilds the
  current screen's DOM, OR use small per-component render functions. Re-render on every state change.
- Icons: use **inline SVG** or an icon font of your choice (Lucide names are referenced below —
  substitute equivalents). Don't block on exact icons.
- Fonts (Google Fonts): **Fredoka** for headings, **Nunito** for body. Load via `<link>`.
- No images required — places use **colored gradient placeholders** with a large faded icon.
- Mobile canvas: design for a **~390 × 844** phone viewport inside the frame. Scale the frame to
  fit the window with a CSS `transform: scale()` so it never overflows.

---

## 2. Design system (use these exact tokens)

Define as CSS custom properties on `:root`:

```css
--accent:        #0d9488;   /* teal — primary */
--accent-2:      #14b8a6;   /* lighter teal for gradients */
--accent-tint:   #d6efec;   /* pale teal chip/background */
--accent-shadow: rgba(13,148,136,0.55);
--sun:           #f6b81e;   /* gold — deals, ratings, quests */
--ink:           #103d3a;   /* near-black text */
--ink-soft:      #3a5a57;   /* secondary text */
--muted:         #6c8582;   /* tertiary text / labels */
--sand:          #fbf6ed;   /* app background (warm off-white) */
--font-display:  'Fredoka', system-ui;
--font-body:     'Nunito', system-ui;
```

Visual language:
- Rounded, friendly, soft. Cards: white, `border-radius: 16–24px`, soft shadows
  (`0 8px 22px -16px rgba(8,28,32,0.35)`).
- Bottom sheets slide up from the bottom with rounded top corners (`28–30px`) and a grab handle.
- Buttons: bold weight, `active:scale(0.98)` press feel, accent fill for primary.
- Chips/pills: pale `--accent-tint` background with `--accent` text, or white with subtle border.
- Headings use `--font-display`; everything else `--font-body`. Body weight is bold-ish (600–800).
- Deals & ratings use `--sun` (gold). Star ratings filled gold.
- **No emoji-heavy UI**, keep it clean. Gradients only on hero/accent surfaces, not everywhere.

There are **4 selectable accent palettes** (teal default). Allow switching the accent at runtime:

| key (accent)| accent2   | tint      | shadow                  |
|-------------|-----------|-----------|-------------------------|
| `#0d9488`   | `#14b8a6` | `#d6efec` | `rgba(13,148,136,0.55)` |
| `#f0682f`   | `#fb923c` | `#ffe5d6` | `rgba(240,104,47,0.5)`  |
| `#0369a1`   | `#0ea5e9` | `#d6ecfa` | `rgba(3,105,161,0.5)`   |
| `#db2777`   | `#fb7185` | `#ffdcec` | `rgba(219,39,119,0.5)`  |

---

## 3. Data model (hard-code in `data.js`)

### Travel styles (onboarding)
`{ id, label, icon, blurb }` — 4 of them:
- `nomad` "Digital Nomad" / Laptop / "Fast Wi-Fi, good coffee, places to focus."
- `surf` "Surf Traveler" / Waves / "Breaks, board rentals, dawn patrol."
- `luxury` "Luxury" / Gem / "Refined stays, fine dining, sunset views."
- `backpacker` "Backpacker" / Backpack / "Cheap eats, hostels, good company."

### Interests (onboarding)
`{ id, label, icon }` — Coffee, Coworking, Nightlife, Surf, Yoga, Local Food, Beaches, Live Music.

### Places (the core list) — `{ id, name, type, icon, x, y, rating, reviews, distance, priceLevel, wifi:{score,speed,label}, hours, openNow, bestTimes:[], deal:{active,label,tag}, indoor?, artisan?, seaView?, tags:[], blurb, photo }`
`x`/`y` are **percentages (0–100)** positioning the pin on the map canvas. `photo` is a gradient key.
Five places:

1. **Driftwood Coffee & Co-Work** — Coworking Café, icon Coffee, x33 y38, ★4.8, 212 rev, 400 m, $$,
   wifi 9.2 / 88 Mbps / "Blazing", 7AM–9PM, open, best morning+afternoon,
   deal active "Free flat white with any day pass" tag "Perk", seaView, photo `cowork`,
   tags [Coffee, Coworking, Fast Wi-Fi, Power outlets].
2. **Weligama Bay Surf School** — Surf School, icon Waves, x62 y64, ★4.9, 540 rev, 650 m, $$,
   wifi 6.1 / 24 Mbps / "Okay", 6AM–6PM, open, best morning+afternoon,
   deal active "20% off beginner group lessons — today only" tag "20% OFF", seaView, photo `surf`.
3. **Hangtime Rooftop & Sunset Bar** — Sunset Bar, icon Martini, x48 y22, ★4.7, 388 rev, 900 m, $$$,
   wifi 7.0 / 31 Mbps / "Good", 4PM–1AM, **closed now**, best evening,
   deal active "Happy hour 5–7 PM · 2-for-1 arrack cocktails" tag "Happy Hr", seaView, photo `sunset`.
4. **Clay & Co Pottery Studio** — Pottery Workshop, icon Palette, x19 y34, ★4.9, 96 rev, 300 m, $$,
   wifi 5.0 / 18 Mbps, 9AM–6PM, open, best morning+afternoon, **indoor, artisan**,
   deal active "Drop-in wheel-throwing taster — Rs 1,500" tag "Workshop", photo `clay`.
5. **Sea Salt Batik Atelier** — Batik Atelier, icon Brush, x73 y30, ★4.8, 73 rev, 750 m, $$,
   wifi 5.5 / 20 Mbps, 9:30AM–5:30PM, open, best morning+afternoon, **indoor, artisan**,
   no deal, photo `batik`.

Gradient placeholders by `photo` key:
`cowork` teal `#0e8c8a→#0a5f6e` · `surf` orange `#f08a3c→#d9532b` · `sunset` `#ff7e5f→#9b3b6a` ·
`clay` `#c2683f→#7a3b22` · `batik` purple `#8a6bff→#4f3aa6`. Draw the place icon large and faded
(`rgba(255,255,255,0.2)`) in the corner of each gradient card.

### Times of day
`morning` (Sunrise, 6–11 AM), `afternoon` (Sun, 11–4 PM), `evening` (Sunset, 4–10 PM).

### Weather (3 states)
- `sunny` — Sun, 31°, "Perfect beach & surf weather"
- `cloudy` — CloudSun, 29°, "Warm and a little hazy"
- `rainy` — CloudRain, 26°, "Monsoon shower — swap to indoor picks"

### Travellers (Group mode) — `{ id, name, initial, color, x, y, at, sub, in }`
Mia, Theo, Inès (seeded **in** the group); Kai, Sofia, Noah (suggested to add). Each has an avatar
color and a map position. The current user **ME** = `{ initial:'LM', name:'You', color:'#0d9488' }`.
Group seed: name "Weligama Crew", code "WLG-4K2", members [mia, theo, ines].

### Vibes (crowd-sourced mood per place)
`productive` (teal), `loud` (orange), `empty` (grey). Seed vote counts per place (e.g. Driftwood:
productive 28 / loud 3 / empty 4). Tapping a vibe increments its count.

---

## 4. App shell & global state

Global `state`:
```
authed: false            // has the user signed in
onboarded: null          // { style, styles[], interests[] } or null
tab: 'map'               // 'map' | 'spots' | 'trip'
selectedPlaceId: null    // open business drawer for this place
showGuide / showGroup / showProfile: false
time: 'afternoon'        // active time-of-day filter
weather: 'sunny'
groupMode: false
group: { name, code, memberIds[] }
quest: { done: 0, goal: 2 }
checkedIn: []            // artisan place ids visited
vibes: { placeId: { productive, loud, empty } }
accent: '#0d9488'        // active palette key
```

**Persistence (localStorage):** `spotto_auth`, `spotto_prefs` (onboarding result), `spotto_group`,
`spotto_trip`. Read on load, write on change. A logout/restart clears the relevant keys.

**Boot flow:** if not `authed` → show **Auth**. Else if `onboarded` is null → show **Onboarding**.
Else show the main app (tabs + current screen).

**Main app chrome:**
- Bottom **tab bar** (white, fixed at bottom): `Discover` (Map icon), `Spots` (LayoutGrid), `Trip`
  (Route/Calendar). Active tab = accent color + tint pill behind icon.
- **Floating AI button** (bottom-right, above tab bar): gradient circle with Sparkles icon → opens
  the AI Guide full screen.
- Apply the accent palette as CSS variables on the app root so the whole UI recolors live.

---

## 5. Screens & functions (build each)

### 5.1 Auth (sign-in)
Simple branded sign-in screen (Spotto logo = a map-pin with a gold sun dot). A "Continue" /
social-style button that just sets `authed = true` and persists `spotto_auth`. Keep it light — it's
a gate, not real auth. (Original supports 3 layouts: `hero` / `immersive` / `minimal` — one is fine.)

### 5.2 Onboarding (3 steps, progress bar at top)
- **Step 0 — Welcome:** hero image (use the map canvas or a gradient), headline
  "Travel like you've got a friend here.", subcopy, "Get started" button.
- **Step 1 — Travel style:** 2-col grid of style cards (multi-select, at least 1 required). Selected
  card flips to accent fill with a check badge.
- **Step 2 — Interests:** wrap of interest chips (multi-select, at least 1 required). Selected =
  accent fill.
- Footer button: "Continue" → "Explore Weligama" on last step. Disabled until the step is valid.
  Back chevron returns a step. On finish: save `{ style: styles[0], styles, interests }` to
  `onboarded` + localStorage, go to the Map tab.

### 5.3 Discover (the Map tab) — **3 selectable home layouts**
A runtime setting `homeLayout` switches between `map` (default), `feed`, `today`. Build all three;
they share the same data and the time/weather/quest/group state.

**(a) Map layout** — the signature view:
- Full-bleed **stylized satellite map** of the bay (draw with CSS gradients / SVG — abstract
  land/water/roads, not a real map). Place a **pin for each place** at its `x%/y%`. Pins show the
  place icon; tapping a pin opens that place's Business Drawer.
- **Weather bar** (top): shows current weather icon + temp + a one-line tip. Tapping cycles
  sunny → cloudy → rainy. When **rainy**, overlay an animated rain effect and bias recommendations
  to **indoor** places.
- **Time filter** row (Morning / Afternoon / Evening pills). Changing it re-ranks/highlights places
  whose `bestTimes` include the selected time.
- **Group mode**: a draggable "you" avatar on the map + the group members' avatars at their
  positions, with a live distance readout. A toggle enables/disables it; an avatar cluster opens the
  **Group Sheet**.
- **Daily quest** banner (see 5.8) and a **profile** avatar button (opens Profile).

**(b) Feed layout** — editorial scroll:
- Header: greeting ("Good morning/afternoon/evening") + "Weligama Bay" + weather/style line +
  profile avatar. A search field (non-functional placeholder is fine). Time-of-day pills.
- **Hero "Top pick now"** card: the highest-rated place matching the current time (or highest-rated
  **indoor** place when rainy). Big gradient image, rating/distance/wifi, deal tag. Taps → drawer.
- **"Happening now" rail**: horizontal scroll of places with active deals. Taps → drawer.
- **Quest banner**, then a **"More in the bay"** vertical list of the remaining places. Taps → drawer.

**(c) Today layout** — personal briefing:
- Accent **gradient hero**: greeting "…, Lena" + "Here's your Weligama", a big weather block
  (icon + temp + tip), and a 3-stat row (spots near / open now / live deals — computed from data).
- A **group strip** card (shows member avatars if group mode on, else "Explore together" CTA →
  Group Sheet).
- **Agenda**: "Picked for your {time}" (or "Dry-day picks" when rainy) — top 3 places ranked by
  rating, numbered, taps → drawer. Includes the time pills.
- **Quest banner** at the bottom.

> Implement `topPick(time, rainy)` and `agenda(time, rainy)` helpers: when rainy, filter to
> `indoor` places; otherwise filter to places whose `bestTimes` includes `time`; sort by `rating`.

### 5.4 Spots (the Spots tab)
A simple **scrollable list of all places as cards** — gradient thumbnail, name, type chip, rating,
distance, deal tag. Each card opens the Business Drawer. (This is the plain "browse everything"
view, no map.)

### 5.5 Business Drawer (bottom sheet for a selected place)
Slides up over any screen. Contains:
- **Photo header** (gradient + faded icon), close button, deal tag if active, grab handle.
- Title, type · price level, gold rating pill, reviews count, distance.
- **Quick stats**: a **Wi-Fi meter** (4 bars colored by score) with score + speed + label, and an
  **Hours** stat ("Open now" green / "Opens later" red) + the hours string.
- **Deal block** (if `deal.active`): gold card with the offer; a **"Claim offer"** button that, when
  tapped, reveals a fake voucher code (`SPOTTO-XXXX`) + "Show at counter".
- **Vibe Check**: the 3 vibe options as tappable bars showing % of votes; tapping casts a vote
  (increments that vibe's count for this place) and highlights your choice. Shows the leading vibe
  + total votes.
- **Artisan check-in** (only for `artisan` places): a "Check in for Daily Quest" button → on tap,
  marks the place checked-in and **advances the daily quest** (done +1, capped at goal). Disabled
  once checked in.
- Blurb paragraph, tag chips, and a bottom action row: "Directions" (primary) + a bookmark button.
- Closing: tap the backdrop or the X. Animate in/out (slide + fade).

### 5.6 AI Guide (full-screen chat, opened by the floating button)
A chat UI tuned to the user's travel style + interests + current weather:
- Header: "Local Guide", a status dot, "Tuned for {style}", close button.
- Opening assistant message adapts to weather (rainy → promises indoor picks).
- **Suggestion chips** (e.g. "Where can I work today with a sea view?", "Best surf spot for a
  beginner?", "Where's the move for sunset tonight?", "Any deals on right now?").
- A text composer (textarea + send). On send, append the user bubble and a typing indicator, then
  produce an assistant reply.
  - **For the vanilla build:** implement a local rule-based responder that matches keywords
    (work/wifi → Driftwood; surf/beginner → Bay Surf School; sunset/drinks → Hangtime; rain/indoor
    → Clay & Co or Batik; deals → list active deals) and replies in 2–4 friendly sentences. *(The
    original calls a real LLM; a keyword responder is a fine stand-in — keep the function isolated so
    it can be swapped for a real API later.)*
- If a reply names a place, show a small **"View on map →" card** under it that opens that place's
  Business Drawer.

### 5.7 Trip (the Trip tab) — itinerary planner
- **Month calendar view**: highlights the days that belong to the current trip; tapping a planned
  day opens its **single-day timeline**. Shows trip length + date range. "Add another day" (up to 4).
- **Day timeline**: ordered **stop cards** (icon, name, type · cost, an editable **time chip**),
  with **travel legs** between stops (walk / tuk-tuk, minutes + distance + a "Directions" link that
  deep-links to Google Maps). Reorder stops up/down (swap their time slots so the timeline stays
  chronological), remove a stop (with an owner-confirm dialog), and a per-day summary
  (stops / total travel minutes / total cost in Rs) + "Open day route in Google Maps".
- **Add-stop sheet**: search any place (filter a catalog) or pick a style-based suggestion; can also
  add a **custom stop** typed by the user. Prevent adding the same place twice.
- **New-trip sheet**: name, **date picker** (inline month calendar), and length (1 / 2 / 3 days).
  Creates a fresh trip.
- **Share / collaborators sheet**: invite link (`spotto.app/t/CODE`) with copy button, add/remove
  collaborators from the traveller roster, invite phone contacts (with a shareable **invitation
  card**). Owner-only deletions.
- Persist the whole trip object to `spotto_trip`. Seed an example 2-day trip so the tab isn't empty.

> Trip helpers to implement: format time (mins → "9:30 AM"), format Rs cost, compute a travel leg
> between two stops (fake but plausible minutes/distance + a Google Maps directions URL from
> lat/lng), and a whole-day Google Maps route URL.

### 5.8 Daily Quest
A small banner (shown on the home layouts) tracking "Visit 2 local artisans → free Ceylon Tea".
Progress `done/goal`. Tapping it jumps to the map and opens the next un-visited artisan place.
Checking in at an artisan (5.5) advances it; at `done == goal` the banner flips to a gold
"🎉 Free Ceylon Tea unlocked!" state.

### 5.9 Group Sheet (bottom sheet)
Manage the travel group: header with group name + member avatar cluster; an **invite card**
(link `spotto.app/g/CODE`, copy button, a decorative **QR code** drawn in SVG, share button);
a **members list** (you = Host, others removable); a **"Add travellers nearby"** list (the
suggestions) with Add buttons; and a "Leave group" button. Adding/removing updates `group.memberIds`
(and the avatars shown on the map in group mode). Persist to `spotto_group`.

### 5.10 Profile
A sheet/screen showing the user's chosen travel style + interests, with options to edit prefs,
restart onboarding, and log out (clears auth + prefs and returns to Auth).

---

## 6. Settings / tweak controls (optional but nice)

Expose a small settings affordance to flip prototype states live (these mirror the original's
"Tweaks"):
- **Accent palette** (the 4 colors above) — recolors the whole app.
- **Heading font** (Fredoka / Baloo 2 / Quicksand / Nunito).
- **Home layout** (map / feed / today).
- **Weather** (sunny / cloudy / rainy).
- **Group mode** (on / off).

---

## 7. Suggested file structure

```
index.html          // frame, fonts, root, <script> includes
styles.css          // tokens + components
data.js             // all hard-coded data (section 3)
state.js            // global state + localStorage load/save + render()
auth.js             // sign-in
onboarding.js       // 3-step flow
map.js              // satellite map + pins + weather + time + group overlay
home.js             // feed + today layouts + quest banner
spots.js            // list view
drawer.js           // business drawer (wifi meter, deal claim, vibe vote, check-in)
guide.js            // AI chat (keyword responder)
trip.js             // calendar + day timeline + add/new/share sheets
group.js            // group sheet + QR
profile.js          // profile / settings
```

---

## 8. Build order

1. Frame + tokens + fonts + tab bar shell with empty screens.
2. `data.js` + global `state` + `render()` + localStorage.
3. Auth → Onboarding → land on Map.
4. Map layout (pins, weather cycle, time filter).
5. Business Drawer (open from pins) with wifi meter, deal claim, vibe vote.
6. Spots list.
7. Feed + Today home layouts + Daily Quest.
8. Group mode + Group Sheet.
9. AI Guide chat.
10. Trip planner (calendar → day timeline → add/new/share sheets).
11. Profile + settings/tweaks.

Keep each interaction working and re-rendering before moving on. Match the look in section 2:
warm sand background, teal accent, gold for deals/ratings, rounded friendly cards, slide-up sheets.

---
---

# PART B — Spotto Admin "Command Center" (desktop web app)

This is a **second, separate app** from the traveller mobile app above. It is the internal back-office
console operators use to run the platform. Build it as its own page — **`admin.html`** — with its own
JS files. It is a **desktop dashboard** (full-window, NOT inside a phone frame): a fixed dark left
navigation rail, a top command bar, and a routed main content area. Same vanilla HTML/CSS/JS rules.

## B0. Look & feel (distinct from the mobile app)

This is a **data-dense, professional admin UI** — think Linear / Stripe dashboard, not the warm
playful traveller app. Font: **IBM Plex Sans** (with `system-ui` fallback); use a monospace for IDs,
codes, and numbers. Supports **Light and Dark themes** (toggle).

Theme tokens (CSS variables, swap the whole set on theme change):

```css
/* Light */
--bg:#f4f6f9; --surface:#ffffff; --line:#e4e8ee;
--ink:#0f1b2d; --ink-soft:#33425a; --muted:#6b7a90;
/* Dark */
--bg:#0a0f1a; --surface:#111a2b; --line:rgba(255,255,255,0.09);
--ink:#f1f5f9; --ink-soft:#cbd5e1; --muted:#7689a3;
```

The **left rail is always dark** (`#0d1525`) in both themes. Accent options (default teal):
`#0d9488` · `#2563eb` · `#7c3aed` · `#0891b2`. Status colors used throughout:
green `#16a34a` (ok/approved/active), amber `#d97706` (needs review / renews soon),
red `#e11d48` (flagged / overdue / reject), slate `#64748b` (free / neutral).

Components: rounded `8–12px` cards with `1px var(--line)` borders (not the soft mobile shadows),
compact rows, small uppercase labels (`11px`, letter-spacing), pill badges, real `<table>`s.

## B1. Demo data (hard-code in `admin-data.js`)

- **VENDOR_QUEUE** — 6 pending vendor applications: `{ id 'V-20xx', name, cat, town, owner, submitted,
  priority: high|normal|low, docs: bool, photos: int, wifi: number|null, flags: string[], note,
  region }`. (e.g. Ahangama Yoga Studio / high / 6 photos; Ella Rock Trail Guides / missing license /
  docs false; Fort Print & Coffee / strong nomad fit, wifi 9.0; etc.)
- **FEED_SEED** + **FEED_TEMPLATES** — live content items: `{ id 'C9xx', vendor, town, type: Offer|Event,
  text, status: ok|review|flag, t (relative time), reason? }`. Templates are randomly streamed in.
- **REGIONS** — 12 Sri Lankan regions: `{ id, name, x, y, travelers (0–100), vendors (0–100) }`
  (Colombo, Negombo, Kandy, Sigiriya, Ella, Nuwara Eliya, Galle, Weligama, Mirissa, Arugam Bay,
  Trincomalee, Jaffna). `x/y` position them on a stylised island map. **gap = travelers − vendors.**
- **SUBS** — 10 vendor subscriptions: `{ id, vendor, town, tier: Premium|Free, mrr (49 or 0),
  renews, state: active|soon|overdue|trial|free, since }`.
- **AUDIO_GUIDES** — 4 geotagged guides: `{ id, site, region, lat, lng, depths[], lang, published,
  plays, duration{} }`. **DEPTH_INFO**: `Quick` (3–5 min), `Standard` (10–15 min), `In-depth`
  (25–35 min), each with an icon + description.
- **ADMIN_KPIS** — header stats: Pending approvals (= queue length, amber), Flagged content (7, red),
  Live offers (412, teal), Platform MRR ($6.1k, green).

## B2. Shell & routing

- **Auth gate** (`spotto_admin_auth` in localStorage). Boot shows **Admin sign-in**; after sign-in,
  show the console. A logout clears the key and returns to sign-in.
- **Left rail** (width ~232, dark): Spotto logo + "Command Center" label; an "Operations" nav group
  with 5 items — **Approval queue** (badge "6"), **Content monitor** (live green dot), **Geo-analytics**,
  **Subscriptions**, **Audio guides**. Active item: subtle highlight + a 2px accent left-border + accent
  icon. Footer: an "All systems operational · API 42ms · 99.98% uptime" health chip.
- **Command bar** (top, height ~58): a search box with a `⌘K` kbd hint, a horizontal **mini-KPI strip**
  (the 4 ADMIN_KPIS), a notification bell with a red dot, and an account button (avatar "AD" / "Admin ·
  Super-admin") that routes to the **profile** page.
- **Main area** renders the active route. `route` state: `queue` (default) | `monitor` | `geo` | `subs`
  | `audio` | `profile`.
- **Keyboard shortcuts** (only on the queue route, and not while typing in an input): `A` approve,
  `R` reject, `E` request edits — trigger the matching action button.

## B3. Sign-in (Admin)

Dark, branded, security-themed, **two-step**: (1) credentials — work email + password (with
show/hide), "Remember device" checkbox, "Forgot?" link; (2) **2FA** — a 6-digit code input (6 separate
boxes, auto-advance / backspace-to-previous), "Trust this device 30 days", "Verify & enter" → signs in.
Also a "Forgot password" sub-screen that fake-sends a reset link. Two layouts: **split** (left = brand
panel with marketing copy + the 4 KPI tiles; right = form) and **centered** (form in a card over a
grid-glow background). It's a prototype — any credentials pass; the Continue/Verify buttons just advance.

## B4. Route screens (build all five)

### B4.1 Approval queue (`queue`) — master/detail
Two-column: a **list** of incoming vendor applications (id, priority pill, flag count, name,
category · town · submitted-ago; decided rows dim and show their outcome) and a **detail panel** for
the selected vendor — hero (icon, id, priority, name, category · town · owner), an amber **flags
callout** if any, a 3-tile meta grid (**Documents** complete/missing, **Photos** count, **Wi-Fi**
score/10 — red when missing/low), the owner's note, and a **submitted-photos** strip (gradient
placeholders). **Action bar**: **Approve** (green), **Request edits** (neutral), **Reject** (red).
Deciding records the outcome, shows a confirmation with an **Undo**, and auto-advances to the next
undecided vendor. Wire the `A`/`R`/`E` shortcuts to these (give the buttons `data-qa` hooks).

### B4.2 Content monitor (`monitor`) — live moderation feed
A **streaming feed** of offers/events. Every ~3.2s a new item is prepended (pull from FEED_TEMPLATES,
bump the relative times of older items, cap ~40); newly arrived rows briefly flash a tint. Each item:
status dot, type badge, vendor, town, timestamp, the quoted text, and a reason line for non-ok items.
**flag**/**review** items show **Approve / Remove / Warn vendor** buttons → on click the row resolves
with a "✓ Content removed / Vendor warned / Approved" line. A **Pause/Resume stream** button stops the
ticker. **Filter tabs**: All activity / Flagged / Needs review (each with a live count) + a Live/Paused
indicator.

### B4.3 Geo-analytics (`geo`) — demand vs supply
Left: a **stylised silhouette map of Sri Lanka** (draw with an SVG path; this is abstract, not
geographically exact) with a **bubble per region** — an **outer ring sized by traveler demand** and an
**inner dot sized by vendor supply**; regions with a big gap get a **pulsing accent ring**. Tap a region
to select it. Right: a **detail card** for the selected region (demand vs supply metrics as bars, a
**supply-gap** number, and an auto-written recommendation that varies by gap size), plus a **"Top
acquisition opportunities" leaderboard** ranking regions by gap (gap ≥ 12). Color regions by gap tone:
≥28 critical (red), ≥12 under-served (amber), ≤−10 saturated (teal), else balanced (slate).

### B4.4 Subscriptions (`subs`) — tiers & MRR
Top: 4 **summary cards** — Premium count (+ $ /mo), Free-tier count, Total MRR (+9% MoM), Renews ≤7d.
**Filter tabs**: All / Premium / Free, plus an "Export CSV" button (decorative). A **table**: Vendor
(+ mono id), Location, Tier (Premium shows a crown), MRR, Renewal (colored pill — "renews soon" amber,
"overdue" red, etc.), Member since, and a row action (Free rows → **Upsell** button; Premium → a hover
"⋯" menu). Sticky header row.

### B4.5 Audio guides (`audio`) — geotagged CMS
Two views toggled by **Library / New guide**:
- **Library**: a card grid of existing guides — gradient header with site name + lat/lng + a
  Published/Draft badge, the offered **depth chips** (Quick/Standard/In-depth with durations), and a
  footer with play count + language + an Edit link.
- **New guide**: a form — site name, region, latitude, longitude; a **depth picker** (tap the 3
  DEPTH_INFO cards to include/exclude); **per-depth fake file upload** rows (Upload → shows a
  `site_depth.mp3` filename); and a sticky **preview/publish** panel showing what the traveller will
  see + a **Publish** button (disabled until a site name and ≥1 depth exist; on publish shows a
  "Published!" success state).

### B4.6 Profile (`profile`) — operator account & security
A settings page: a Save-changes button (shows "Saved" briefly); an **identity hero** (avatar "AD", name,
"Super-admin" badge, email, Verified chip); then a responsive grid of cards — **Profile details**
(name, email, role read-only, region), **Security** (change password, a **two-factor toggle**, recovery
codes), **Active sessions** (list of devices with a "This device" tag / Revoke), and **Notifications**
(toggles for approval queue / flagged content / weekly report / sign-in alerts). A **Sign out** card at
the bottom calls logout.

## B5. Admin settings / tweaks (optional)
Expose: **Theme** (Light / Dark), **Accent** (the 4 admin colors), **Sign-in layout** (split / centered),
and a **Log out** button.

## B6. Admin file structure & build order
```
admin.html          // dark rail shell, IBM Plex font, theme vars, route includes
admin.css           // light/dark tokens + dashboard components
admin-data.js       // section B1 data
admin-app.js        // rail + command bar + routing + auth gate + shortcuts
admin-auth.js       // 2-step sign-in
queue.js  monitor.js  geo.js  subs.js  audio.js  admin-profile.js
sri-lanka-map.js     // the SVG silhouette + region bubbles
```
Build order: shell + theme toggle + rail/command-bar with empty routes → data → auth gate →
Approval queue (the flagship) → Content monitor (streaming) → Geo-analytics (map) → Subscriptions
(table) → Audio CMS (library + form) → Profile.

> The traveller app (Part A) and the admin console (Part B) are **two independent pages** that share
> only the Spotto brand mark and the Sri Lanka / Weligama context — keep their code, styling, and
> typography separate.
```
