# Spotto Marketing Website — Payload CMS Architecture & Integration Guide

## 1. Executive Summary: Can we use Payload CMS (https://payloadcms.com/)?
**YES, absolutely.** Payload CMS is an ideal match for Spotto's technology ecosystem:
- **TypeScript & Next.js Native**: Payload 3.x is built directly on Next.js (App Router), enabling seamless embedding or standalone microservice deployment.
- **PostgreSQL Synergy**: Uses `@payloadcms/db-postgres`, allowing Spotto to leverage its existing PostgreSQL database instance without introducing MongoDB or other disparate database engines.
- **R2 / S3 Asset Storage**: Compatible with Cloudflare R2 (`@payloadcms/storage-s3`), matching Spotto's existing media pipeline.
- **Decoupled Headless API**: Supplies both REST (`/api/pages`, `/api/pricing-plans`, `/api/faqs`, `/api/testimonials`, `/api/stories`) and GraphQL endpoints that feed directly into the marketing site (`website/`) with instant fallback to static pre-rendered HTML.

---

## 2. Content Architecture Matrix

| Collection | Purpose | Key Attributes |
|---|---|---|
| `Pages` | Full marketing page layout & copy | Slug (`home`, `features`, `businesses`, `pricing`, `about`, `contact`), Meta description, Hero (eyebrow, headline, highlightText, lead, CTAs), Stats rows, CTA bands |
| `PricingPlans` | Vendor subscription tiers ($3.99, $11.99, $22.99) | Name, Slug, Monthly/Yearly USD, Badge ("Most Popular"), Feature relations, Limits (activeOffers, flashSalesPerMonth, teamSeats), CTA text/link |
| `Features` | Master system feature pool | Key (`feat_map_pin`), Name, Category (discovery/marketing/loyalty/operations/insights/trust), Lucide icon |
| `FAQs` | Questions & answers | Question, Answer markdown, Category filter (`pricing`, `businesses`, `travellers`), Sort order |
| `Testimonials` | Traveller & partner social proof | Author, Quote, Highlight quote (`<span class="hl">`), Venue, Town, Avatar initials, Page tag, isPrimary |
| `Stories` | Editorial guides and articles | Title, Slug, Summary, Markdown body, Cover URL, Category, Tags, Author, Region, PublishedAt |
| `Users` | Admin & editorial authentication | Email, Password, Name, Role (`admin`, `editor`) |

---

## 3. Frontend Hydration & Fallback Resilience (`website/site.js`)

The frontend marketing site (`website/`) connects to Payload CMS through `website/site.js`:
1. **Zero Layout Shift / Instant First Paint**:
   All HTML pages have pre-rendered static HTML fallback content. When visitors load the site, content displays immediately with zero delay.
2. **Dynamic Hydration**:
   `site.js` automatically queries Payload CMS REST endpoints:
   - `/api/pages?where[slug][equals]=<slug>`: Populates Hero headline, eyebrow, lead copy, CTAs, and stats counters.
   - `/api/pricing-plans?depth=2&sort=sortOrder`: Renders the 3 vendor tiers ($3.99, $11.99, $22.99), calculates monthly/yearly rates, renders feature inclusions and exclusions, and updates the Feature Matrix table.
   - `/api/faqs?sort=sortOrder`: Dynamically populates the interactive FAQ accordions.
   - `/api/testimonials?sort=sortOrder`: Dynamically populates the primary quote and mini quote cards.
   - `/api/stories?where[status][equals]=published`: Populates the Stories hub and individual article view on `blog.html`.
3. **Resilience & Fallback**:
   If the CMS server is offline or unreachable, `site.js` gracefully retains the pre-rendered static HTML. No errors crash the UI.
4. **Endpoint Resolution**:
   - `?cms=http://localhost:3002` (URL parameter override)
   - `data-cms` attribute on `<body>`
   - `window.SPOTTO_CMS_URL`
   - Default auto-discovery for local dev (`http://localhost:3002`) and production endpoints.

---

## 4. Bi-Directional Synchronization with Spotto Command Center

Spotto Super Admins can manage vendor tiers via two synchronized surfaces:
1. **Spotto Command Center Kanban (`web/admin/packages`)**:
   - Fast, tactile HTML5 Drag-and-Drop feature allocator.
   - Saves directly to Go API `/admin/packages` and PostgreSQL `platform_settings`.
2. **Payload CMS Admin Panel (`/admin`)**:
   - Comprehensive editorial control over marketing copy, headlines, testimonials, and FAQs.
   - Run `ts-node sync-bridge.ts` to sync between Command Center and Payload CMS.

---

## 5. Setup & Running Payload CMS

```bash
# 1. Install dependencies
cd website/cms
npm install

# 2. Set environment variables (.env)
DATABASE_URL=postgres://postgres:admin123@localhost:5432/spotto?sslmode=disable
PAYLOAD_SECRET=spotto-payload-super-secret-key-32-chars-min

# 3. Seed initial marketing content, plans, FAQs, and testimonials
npx ts-node seed.ts --run

# 4. Run Payload CMS in dev mode (port 3002)
npm run dev
```
