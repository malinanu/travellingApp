# Skipped — Apple Sign-In and real Stripe billing (2026-07-18)

## Why this exists

During the vendor-portal design-parity pass (Shell, Dashboard, Account,
Login/Auth rebuilt to match `Prototype/design/`), the design mockups call
for two features this app doesn't have real backend support for: Apple
Sign-In and payment-processor-backed billing management. Everything else
gated on missing backend infra was built for real in this pass — a real
SMTP mailer (`server/internal/mail`), email-delivered team invites with a
new "manager" role, a real forgot/reset-password flow, and real Google
OAuth sign-in (`server/internal/http/oauth.go`). These two were not, for
the reasons below, and are documented here instead per the repo's
existing `handover.md` precedent — copy either section into a task when
you're ready to build it, or use it as a spec.

---

## 1. Apple Sign-In

**Design reference:** `Prototype/design/VendorAuth.jsx`'s `VSocial`
component renders a Google button and an Apple button side-by-side on the
login screen. This pass built and wired the Google button for real
(`GET /auth/google/login` / `/auth/google/callback` in
`server/internal/http/oauth.go`); the Apple button is not rendered at all
in the rebuilt `web/vendor/app/login/page.tsx` — a visibly present but
non-functional button would be worse than an absent one.

**Why it wasn't built:** Unlike Google OAuth (a free Google Cloud client
ID/secret is all that's needed), "Sign in with Apple" requires:
- An active **Apple Developer Program membership** (currently $99/year,
  paid, tied to a real Apple ID/organization) — there's no free tier.
- A registered **Services ID** and **App ID** in the Apple Developer
  portal, with "Sign in with Apple" capability enabled and the exact
  redirect URL registered.
- A **private key** (downloaded once, .p8 file) plus its **Key ID** and
  your **Team ID**, used to mint a short-lived signed JWT as the OAuth
  "client secret" on every token exchange (Apple's client-secret format is
  itself a JWT you generate — there's no static secret string like
  Google's).
- The client-secret JWT expires (max 6 months) and must be regenerated
  and redeployed periodically, which is an ongoing operational task, not
  a one-time setup.

None of this can be provisioned by an agent — it requires the account
owner to enroll in and pay for the Apple Developer Program and generate
credentials through Apple's own portal.

**To build it later:** once you have the Team ID/Key ID/private key and
Services ID, the shape mirrors `oauth.go`'s Google implementation almost
exactly (`golang.org/x/oauth2` supports a generic `oauth2.Config`, though
you'll need a small helper to mint the JWT client secret from the .p8 key
using `golang-jwt/jwt` — already a dependency here). Add
`APPLE_TEAM_ID`/`APPLE_KEY_ID`/`APPLE_PRIVATE_KEY`/`APPLE_CLIENT_ID` env
vars alongside the existing `GOOGLE_*` ones in `server/internal/config`,
add `handleAppleLogin`/`handleAppleCallback` following `oauth.go`'s
pattern (including routing through the same `authCodeStore` one-time
exchange), and add the button back to the login page's social row
(`GoogleG`/social button markup is already there to copy).

---

## 2. Real Stripe / payment-processor billing management

**Design reference:** `Prototype/design/AccountProfile.jsx`'s "Plan &
team" card shows an active `Manage` button next to the plan tier. The
rebuilt `web/vendor/app/account/page.tsx` keeps this button disabled with
a "Billing management is coming soon" tooltip, matching the pre-existing
behavior (this was already a stub before this pass, per the file's own
header comment) — it wasn't turned into a working button, and no payment
processor code was added.

**Why it wasn't built:** today, "Premium" (`domain.Vendor.Tier` in
`server/internal/domain/domain.go`) is a plain string an admin sets
directly via `POST /admin/subscriptions/{id}/upgrade`
(`server/internal/http/admin.go`'s `handleUpgradeSubscription`) — no money
ever actually moves. Turning `Manage` into a real button requires:
- A **live Stripe account** (or another processor) — real business/bank
  details, tax info, and payout setup, which nobody but the business owner
  can create.
- A **product/price catalog** defined in that Stripe account (what
  "Premium" costs, billing interval, currency) — a business decision, not
  a technical one.
- New endpoints: creating a Stripe Customer per vendor, a Checkout Session
  (upgrade) or Billing Portal Session (manage/cancel) redirect, and a
  **webhook endpoint** to receive `customer.subscription.*` events and
  keep `Vendor.Tier`/the `Subscription` row in sync — this is the part
  that actually matters for correctness (a UI button alone doesn't keep
  billing state truthful; the webhook does).
- A `STRIPE_SECRET_KEY` and `STRIPE_WEBHOOK_SECRET`, plus a publicly
  reachable HTTPS endpoint for Stripe to call (won't work against
  `localhost` without a tunnel like ngrok during development).

This is meaningfully higher-risk than the OAuth/email work in this pass —
it touches real money and requires getting webhook-driven state
reconciliation right, not just wiring up a form. Attempting a partial
version (e.g. a fake "Manage" button that doesn't talk to a real
processor) would be worse than leaving today's honest "coming soon" stub,
which is what this pass kept.

**To build it later:** once a Stripe (or equivalent) account and price
catalog exist, add a `server/internal/billing` package following the same
shape as `server/internal/mail` (config struct, optional-at-startup),
`POST /vendor/business/billing/portal` returning a redirect URL for the
`Manage` button, and a `POST /webhooks/stripe` handler (unauthenticated,
verified via Stripe's signature header instead of the JWT middleware)
that updates `Vendor.Tier` and the `Subscription` row on
subscription-changed events.
