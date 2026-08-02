# Spotto Marketing Website — Build Prompt for Claude Code

Build a static marketing website (vanilla HTML/CSS/JS, no framework, no build step) for **Spotto** — a travel-discovery app for Sri Lanka. Travellers use the free app to find curated local spots, claim deals, and get AI-guided tips; local businesses pay for subscription tiers to reach them.

## Pages (7, all linked via shared nav + footer)
1. `index.html` — Home
2. `features.html` — Traveller app features
3. `businesses.html` — For businesses/vendors
4. `pricing.html` — Vendor subscription tiers
5. `about.html` — Company story
6. `blog.html` — Stories/blog index
7. `contact.html` — Contact form

Shared assets: `site.css` (one stylesheet), `site.js` (scroll reveal + mobile nav).

## Brand & visual language
Playful, rounded, tropical — matches the Spotto app.

**Colors (CSS custom properties):**
- `--accent: #0d9488` (teal, primary), `--accent-2: #14b8a6`, `--tint: #d6efec`
- `--sun: #f6b81e` (gold, secondary CTA), `--sun-soft: #fff3d1`
- `--coral: #f0682f` accent; supporting hues: sky `#0369a1`, pink `#db2777`
- Ink `#103d3a`, soft ink `#3a5a57`, muted `#6c8582`
- Page background: warm sand `#fbf6ed`; footer/deep panels: `#081c20`

**Type:** Fredoka (400–600) for display/headings, Nunito (600–900) for body/UI. Google Fonts.

**Shape language:** pill buttons (border-radius 999px), cards radius 22px, soft long shadows (`0 14px 34px -20px rgba(8,28,32,.4)`), 1px faint borders `rgba(16,61,58,.1)`. Icon chips: 52px rounded-16px squares in pastel tints (teal/gold/coral/blue/pink) with 24px stroke icons.

**Components:** pill chips for labels; uppercase letter-spaced teal eyebrows above section titles; white cards on sand background; alternating text/visual rows; full-width gradient CTA bands (radius 30px); FAQ as native `<details>` with +/– indicator; dark footer with a wave-top SVG divider.

**Imagery:** NO photos. Gradient placeholder panels (135° two-to-three-stop gradients in brand hues) with a large 15%-opacity white line-icon and a small white pill caption tag bottom-left.

**Copy tone:** fun, warm, concise, traveller-to-traveller ("Travel like you know a local", "Made with sun & salt water"). Sparse emoji OK (🌊 👋).

## Navigation
Sticky floating pill nav (white, blur backdrop, shadow): logo (teal map-pin with wave inside + "Spotto" in Fredoka), links (Features, For businesses, Pricing, About, Stories, Contact), teal pill CTA. Active link gets teal tint pill. Mobile: hamburger opens dropdown card; CTA hidden.

## Home page — hero with "8 Wonders of Sri Lanka" animated scene loop
Two-column hero on sand: left = gold chip "Made in Sri Lanka", H1 "Travel like you know a local" with hand-drawn gold squiggle SVG underline under "know a local", subcopy, teal + ghost CTAs, three stat chips. Right = a rounded gradient panel with floating notification cards (deal claimed, quest complete, surf report) bobbing on offset `floaty` keyframes.

**The signature feature:** the bottom strip of the hero (~175px tall, full-width) is an animated scene stage that cross-fades (1.3s opacity) through 8 stylized flat-silhouette scenes, one every 6s, looping, paused on hover. The hero panel's gradient, place-name tag, and small progress dots (top-right of panel) sync with the active scene:

1. **Arugam Bay** (teals `#5eead4/#14b8a6/#0d9488`) — 3 parallax wave layers (each a 200%-wide flex pair of tileable wave SVGs scrolling via `translateX(-50%)` loops at 13s/9s/7s, varying opacity) + a surfer silhouette on a gold board crossing left→right over 16s with a 1.6s bob/tilt.
2. **Mirissa** (sunset `#fbbf24/#f97316/#ea580c`) — glowing sun ball (radial gradient + box-shadow glow) low over orange waves, whale silhouette with raised tail swimming across.
3. **Sigiriya** (amber `#fcd34d/#d97706/#b45309`) — tapered Lion Rock silhouette (narrow flat summit, sheer sides) rising from dunes, gold fresco squiggle strokes on the face, small bird flock (three 2-arc birds) drifting across with slight vertical sine.
4. **Ella** (greens `#86efac/#22c55e/#16a34a`) — rolling tea hills, Nine Arches Bridge as an SVG `<pattern>`: solid deck + piers with semicircular arch openings cut out at the bottom (`M0 0 H150 V113 H132 V73 A57 57 0 0 0 18 73 V113 H0 Z`, tile 150×113), and a little dark-green train with lit windows crossing the deck (9s) trailing two fading smoke puffs.
5. **Yala** (savanna golds `#eab308/#ca8a04/#a16207`) — grass tuft strokes, pale sun disc, leopard silhouette prowling across (12s) with a quick 0.7s gait bob.
6. **Galle Fort** (sea blues `#7dd3fc/#0ea5e9/#0369a1`) — crenellated rampart wall (SVG pattern), white lighthouse with red roof and blue stripes standing on the wall (outlined so it reads against the sand bg), warm light beam sweeping ±15° from the lantern, waves in front.
7. **Kandy** (temple reds `#fda4af/#ef4444/#dc2626`) — Dalada Maligawa silhouette: layered dark-red pavilions with a central golden-roofed shrine (stacked gold roof tiers), flickering oil-lamp dots (staggered 2.2s opacity pulses) along the terrace.
8. **Adam's Peak** (dawn `#f9a8d4/#818cf8/#6366f1`) — two indigo mountain ridge layers with the sacred peak center, a dawn sun slowly rising behind the ridge (12s alternate), and a trail of flickering pilgrim lights zig-zagging up the slope.

Implementation: each scene is an absolutely-positioned layer (`.scene`, opacity 0 → `.on` opacity 1). A small JS array holds per-place: name, tagline, 3 gradient colors, panel icon path. On tick: toggle scene classes, cross-fade the panel gradient via two stacked bg layers, fade-swap the icon + tag text (450ms), update dots.

Below hero: stats row (120+ spots, deals claimed, partners, 4.9★) → 6 feature cards (map with taste / real deals / AI guide / surf alerts / quests / trip planner) → 3-step "How it works" on white → For-businesses split row with dashboard placeholder → 3 testimonial cards (stars, quote, avatar circle) → teal gradient CTA band.

Scroll-reveal: `.rv` elements fade/translate up on IntersectionObserver, with `.d1/.d2/.d3` stagger delays.

## Features page
Page hero (chip, H1, subcopy centered) + 5 alternating rows, each: eyebrow, H3, paragraph, 3 checkmark bullets, gradient placeholder panel opposite (Discover/map, Ask/AI, Save/deals+quests, Surf/alerts, Plan/trip planner). End: dark-teal CTA band "Free for travellers. Always."

## For Businesses page
Hero with coral chip + dual CTAs → stats row (6,800+ travellers, 248 redemptions/wk, <60s flash sale, $0 to start) → 6 tool cards (flash sales, digital stamp cards, QR validation, live analytics, boost campaigns, audience insight) → 3-step onboarding row with dashboard placeholder → 3 partner testimonials → coral CTA band.

## Pricing page (vendor tiers)
Monthly/Yearly pill toggle ("Yearly · 2 months free"; Premium shows $49 → $41/mo).
- **Starter — $0 forever:** full listing, photos/hours/tags, 1 live offer, QR scanning; excluded (dimmed ✕): flash sales, stamp cards/analytics. Ghost CTA.
- **Premium — $49/mo (highlighted):** dark teal gradient card, scale 1.04, gold "Most popular" badge; everything unlimited + flash sales 3km + stamp cards + analytics + priority placement. Gold CTA "Start 14-day trial".
- **Collective — Custom:** multi-venue dashboard, co-branded quests, partner manager, quarterly reports. Ghost CTA "Talk to us".

Below: 3 boost add-on cards ($12 local map top spot / $24 category feature — gold border / $39 push+map+category) → 5-question FAQ (`<details>`) → teal CTA band.

## About page
Story hero → "why we exist" split row → 3 value cards (Local first / Free for travellers / The island wins together) → 3 team cards (avatar initials K/M/D) → dark CTA band "Coming to a bay near you".

## Blog page
Featured post card (split: gradient panel + body) + 6 post cards in 3-col grid (gradient thumb, category chip, title, excerpt, author/date meta). Gold newsletter CTA band with inline email form (JS swaps to a thank-you on submit).

## Contact page
Two columns: card form (name, email, "I am a…" select, message; submit swaps to success state) + 3 info cards (email, location, business fast-lane). Focus states: teal border + tint ring.

## Footer (all pages)
Dark `#081c20` with wave-top SVG: brand column (logo + tagline "Your pocket local for Sri Lanka"), 3 link columns (Explore/Business/Company), bottom bar "© 2026 Spotto · Sri Lanka" / "Made with sun & salt water".

## Responsive
Breakpoints 960px (3→2 cols, hero stacks) and 680px (all 1-col, hamburger nav, alternating rows put visual first, hero scene strip stays full-width). Body text ≥16px; buttons ≥44px tall.

## Quality bar
- All animation CSS-driven (keyframes/transitions); JS only orchestrates class toggles and timers.
- Scenes are flat silhouette SVG art in scene-matched palettes — cohesive, playful, never clip-arty.
- Semantic HTML, one shared stylesheet, no external JS libraries.
