// Hand-maintained TS mirror of the Go API's JSON shapes (see server/internal/domain).

export type Role = 'traveller' | 'vendor' | 'admin';
export type VendorStatus = 'pending' | 'approved' | 'rejected' | 'edits' | 'suspended';
export type OfferStatus = 'draft' | 'live' | 'ending' | 'ended';
export type ClaimStatus = 'claimed' | 'redeemed';

export type VendorRole = 'owner' | 'staff' | 'manager';

export interface User {
  id: string;
  email: string;
  role: Role;
  name: string;
  country: string;
  travelStyle: string;
  interests: string[] | null;
  vendorId?: string;
  vendorRole?: VendorRole;
  createdAt: string;
  vendorStatus?: VendorStatus; // present on GET /me for vendor users
  /** True only for admin accounts listed in the server's SUPER_ADMIN_EMAILS. */
  isSuperAdmin?: boolean;
  /** Admin may hand vendor logins over to their owners (super-admins always). */
  canManageVendorLogins?: boolean;
  /** May write travel content in the Command Center (admins always can). */
  publisher?: boolean;
  totpEnabled: boolean;
  phone: string;
  /** Confirmed with a one-time code (WhatsApp or SMS); a new number resets it. */
  phoneVerified: boolean;
  shareLocation: boolean;
  notifyRedemptions: boolean;
  notifyReviews: boolean;
  notifyPayouts: boolean;
  notifyGrowthTips: boolean;
  /** Public R2 URL of the profile picture, or "" to fall back to initials. */
  avatarUrl: string;
}

/** One uploaded object in Cloudflare R2 — GET /vendor/media,
 * /admin/vendors/{id}/media. role is cover|gallery|promo|doc|avatar|... */
export interface Media {
  id: string;
  ownerType: string;
  ownerId: string;
  kind: 'image' | 'video' | 'doc' | 'audio';
  role: string;
  objectKey: string;
  url: string;
  /** A small (~320px) JPEG derivative for grid/list tiles — absent for
   * media uploaded before this existed, non-image kinds, or if generation
   * failed; fall back to `url` whenever this is missing. */
  thumbUrl?: string;
  contentType: string;
  size: number;
  width: number;
  height: number;
  uploadedBy: string;
  status: string;
  sort: number;
  createdAt: string;
}

export interface TeamMember {
  id: string;
  name: string;
  email: string;
  vendorRole: VendorRole;
  createdAt: string;
}

export interface TeamInvite {
  id: string;
  code: string;
  email?: string;
  role: 'staff' | 'manager';
  createdAt: string;
  expiresAt: string;
}

export interface TeamResponse {
  members: TeamMember[];
  invites: TeamInvite[];
}

/** A time-limited PIN grant giving a manager temporary Business Profile
 * edit access — see server/internal/http/vendor_grants.go. */
export interface VendorProfileGrant {
  id: string;
  vendorId: string;
  granteeId: string;
  createdBy: string;
  durationMins: number;
  createdAt: string;
  pinExpiresAt: string;
  redeemedAt?: string;
  accessExpiresAt?: string;
  revokedAt?: string;
  revokedBy?: string;
}

/** Joined, display-ready row for the platform admin's cross-vendor grant
 * list — GET /admin/vendor-grants. */
export interface AdminVendorProfileGrant extends VendorProfileGrant {
  vendorName: string;
  granteeName: string;
  ownerName: string;
  status: 'active' | 'pending' | 'expired' | 'revoked';
}

export interface AuthResponse {
  token: string;
  /** The session's refresh token (absent from servers before sessions). */
  refreshToken?: string;
  user: User;
}

/** Returned by POST /auth/login instead of AuthResponse when 2FA is enabled. */
export interface TwoFactorRequiredResponse {
  twoFactorRequired: true;
  twoFactorToken: string;
}

export function isTwoFactorRequired(
  res: AuthResponse | TwoFactorRequiredResponse,
): res is TwoFactorRequiredResponse {
  return (res as TwoFactorRequiredResponse).twoFactorRequired === true;
}

export interface TwoFactorSetupResponse {
  secret: string;
  otpauthUrl: string;
}

export interface Vendor {
  id: string;
  name: string;
  category: string;
  town: string;
  /** ISO 3166-1 alpha-2 code (e.g. "LK", "US") — picks the right public-
   * holiday calendar for the closure-prompt feature. Defaults to "LK". */
  country: string;
  regionId?: string;
  owner: string;
  status: VendorStatus;
  priority: string;
  docs: boolean;
  photos: number;
  wifi?: number;
  flags: string[] | null;
  note: string;
  tier: string;
  profile: VendorProfile;
  menu: StoredMenuDoc | null;
  submittedAt: string;
  lat: number;
  lng: number;
  /** Business Registration certificate number (e.g. "PV00334782") — Gemini
   * pre-fills this from an uploaded "br_cert" doc, always editable by the
   * vendor, then verified in the background against Sri Lanka's DRC eROC
   * registry. See brCertOcrStatus/brCertVerifyStatus for pipeline state. */
  brCertNumber: string;
  brCertOcrStatus: '' | 'pending' | 'done' | 'error';
  brCertVerifyStatus: '' | 'pending' | 'verified' | 'invalid' | 'manual_review';
  brCertCompanyName?: string;
  brCertMessage?: string;
  brCertAttempts?: number;
  brCertVerifiedAt?: string;
  badReviewCount?: number;
  flagged?: boolean;
  /** Crowdsourced Current Vibe reading for this vendor's synced place —
   * only present from GET /vendor/business (vendor's own view) and the
   * admin per-vendor vibe endpoint, undefined everywhere else. Distinct
   * from the vendor-entered Wi-Fi/speed "Connectivity & vibe" card, which
   * is self-reported, not traveller check-in data. */
  vibe?: VibeAggregate;
}

/** Current Vibe reading for a place — 1-5 per dimension (0 = no data yet),
 * averaged from real traveller check-ins. See sampleSize for how many
 * distinct travellers contributed within the rolling window. */
export interface VibeAggregate {
  productive: number;
  loud: number;
  crowded: number;
  sampleSize: number;
}

/** A traveller's rating/text for a Place; 1-2 star reviews count against the
 * owning vendor for the admin map's red-flag threshold. */
export interface Review {
  id: string;
  placeId: string;
  vendorId?: string;
  travellerId: string;
  rating: number;
  text: string;
  createdAt: string;
  travellerName?: string;
}

export type Weekday = 'monday' | 'tuesday' | 'wednesday' | 'thursday' | 'friday' | 'saturday' | 'sunday';
export const WEEKDAYS: Weekday[] = ['monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday', 'sunday'];

export interface DayHours {
  open: string;
  close: string;
  closed?: boolean;
}

/** An owner/manager's explicit open/closed decision for one date, recorded
 * via POST /vendor/holidays/decision — a Closed:true entry for today always
 * overrides the regular weekly `hours` schedule. */
export interface HolidayOverride {
  date: string; // "YYYY-MM-DD"
  name: string;
  closed: boolean;
}

/** An upcoming public holiday the vendor hasn't yet recorded a decision
 * for — GET /vendor/holidays/upcoming. */
export interface UpcomingHoliday {
  date: string; // "YYYY-MM-DD"
  name: string;
}

export interface VendorProfile {
  wifi?: { enabled: boolean; score: number; speed: number };
  bestTimes?: string[];
  seaView?: boolean;
  hours?: Partial<Record<Weekday, DayHours>>;
  holidayOverrides?: HolidayOverride[];
  amenities?: string[];
  blurb?: string;
  contact?: { phone?: string; website?: string; instagram?: string; facebook?: string };
}

export interface MenuItem {
  id: string;
  name: string;
  price: number;
  desc: string;
  img?: string | null;
  icon: string;
  /** Whitelist, not a blacklist: absent = available every day (the default,
   * and the state of every item that predates this field). Once present,
   * only days explicitly mapped to `true` are shown to travellers — an
   * unlisted or `false` day is excluded. Only ever create this map when the
   * vendor deliberately restricts to specific days. */
  availableDays?: Partial<Record<Weekday, boolean>>;
}

export interface MenuSection {
  id: string;
  name: string;
  items: MenuItem[];
  /** Same whitelist semantics as MenuItem.availableDays. */
  availableDays?: Partial<Record<Weekday, boolean>>;
}

/** A whole named menu the vendor can create and manually toggle live — see
 * StoredMenuDoc.activePresetId. Only one preset is served to travellers at
 * a time, chosen by the owner, independent of which one they're editing. */
export interface MenuPreset {
  id: string;
  name: string;
  sections: MenuSection[];
}

/** Wire/storage shape for Vendor.menu and GET/PUT /vendor/menu. May still be
 * the pre-preset legacy shape ({} or {sections:[...]}) for any vendor who
 * hasn't resaved since presets shipped — the vendor portal normalizes this
 * into an implicit single preset on load (see menu/page.tsx), never assume
 * `presets` is populated. */
export interface StoredMenuDoc {
  sections?: MenuSection[]; // legacy-only, ignored once `presets` is non-empty
  presets?: MenuPreset[];
  activePresetId?: string;
}

export interface Offer {
  id: string;
  vendorId: string;
  title: string;
  type: string;
  value: string;
  status: OfferStatus;
  start: string;
  end: string;
  limit: number;
  /** Live-computed from real claims, not a stored counter. */
  claimed: number;
  /** Live-computed from real claims, not a stored counter. */
  redeemed: number;
  /** Real traveller impressions, via POST /offers/:id/view. */
  views: number;
  updatedAt?: string;
}

export interface Claim {
  id: string;
  code: string;
  placeId: string;
  /** Links back to the specific vendor Offer, when resolvable. */
  offerId?: string;
  offerLabel: string;
  offerValue: string;
  vendorId?: string;
  travellerId: string;
  status: ClaimStatus;
  claimedAt: string;
  redeemedAt?: string;
  travellerName?: string;
  travellerCountry?: string;
  travellerStyle?: string;
}

export type ValidateResult = 'valid' | 'already_redeemed' | 'wrong_vendor' | 'unknown';

export interface ValidateResponse {
  result: ValidateResult;
  claim?: Claim;
  /** Set instead of `claim` when the scanned/entered code is a BOOK- code
   * (booking check-in) rather than a SPOTTO- code (offer redemption) — see
   * server's handleValidate prefix dispatch. */
  booking?: Booking;
  redeemedAt?: string;
}

export type BookingServiceType = 'session' | 'table' | 'stay';
export type BookingStatus = 'confirmed' | 'cancelled' | 'completed' | 'no_show';

export interface ServiceAvailabilityWindow {
  start: string; // "HH:MM"
  end: string;
}
export type ServiceAvailabilityDoc = Record<string, ServiceAvailabilityWindow[]>;

/** One bookable thing a vendor offers (e.g. "Beginner surf lesson", "Deluxe
 * Room"). No payment is collected anywhere in this platform — price is
 * display-only, vendor collects payment in person. */
export interface BookableService {
  id: string;
  vendorId: string;
  type: BookingServiceType;
  name: string;
  description: string;
  /** session/table only — how long one booking occupies capacity. */
  durationMins?: number;
  /** How many bookings may hold this service concurrently (concurrent
   * session/table bookings, or total rooms of this type for stay). */
  capacity: number;
  partySizeMin?: number; // session/table only
  partySizeMax?: number; // session/table only
  guestsPerRoom?: number; // stay only
  priceLabel: string;
  /** Weekly availability rule for session/table — undefined for stay
   * (stay availability is capacity-only) or when unset (falls back to the
   * vendor's own Hours on the server). */
  availability?: ServiceAvailabilityDoc;
  active: boolean;
  createdAt: string;
  updatedAt: string;
}

/** A traveller's real, auto-confirmed commitment against a BookableService.
 * No payment collected — vendor checks the traveller in via the same
 * BOOK-XXXX QR/code flow offers use for redemption. */
export interface Booking {
  id: string;
  code: string;
  vendorId: string;
  serviceId: string;
  type: BookingServiceType;
  placeId: string;
  travellerId: string;
  startAt: string;
  endAt: string;
  partySize: number;
  status: BookingStatus;
  note?: string;
  createdAt: string;
  cancelledAt?: string;
  cancelledBy?: 'traveller' | 'vendor';
  checkedInAt?: string;
  serviceName?: string;
  travellerName?: string;
  travellerCountry?: string;
  vendorName?: string;
}

export interface VendorNotification {
  id: string;
  icon: string;
  tone: string;
  title: string;
  sub: string;
  time: string;
  unread: boolean;
}

export interface DashboardBlob {
  kpis?: { id: string; label: string; value: number; icon: string; delta: number | null; tone: string; hint: string }[];
  week?: { day: string; views: number; redemptions: number }[];
  popular?: { h: number; v: number }[];
  trend?: { wk: string; you: number; cat: number }[];
  questClaimsToday?: number;
  questCompletionsToday?: number;
}

/** A traveller finishing one day's quest (GET /admin/quests/completions,
 * GET /vendor/quest/completions — the vendor list has no travellerEmail). */
export interface QuestCompletion {
  travellerId: string;
  travellerName: string;
  travellerEmail?: string;
  date: string; // UTC quest day, YYYY-MM-DD
  vendorId?: string;
  sponsorName: string;
  goal: number;
  rewardDescription: string;
  completedAt: string;
  redeemedAt?: string;
}

/** One row of GET /admin/users?q=. */
export interface AdminUserSummary {
  id: string;
  name: string;
  email: string;
  role: Role;
  vendorName?: string;
  publisher: boolean;
  createdAt: string;
  activeSessions: number;
  lastActiveAt?: string;
}

/** A signed-in device (server session); the refresh token is never sent. */
export interface AdminUserSession {
  id: string;
  userId: string;
  client: 'app' | 'admin' | 'vendor';
  deviceLabel: string;
  createdAt: string;
  lastUsedAt: string;
  expiresAt: string;
}

/** GET /admin/users/{id}. */
export interface AdminUserDetail {
  user: User;
  sessions: AdminUserSession[];
  questCompletions: QuestCompletion[];
  audit: AdminAuditLogEntry[];
  /** Extra powers granted to an admin by a super-admin, e.g. 'vendor_logins'. */
  adminPermissions: string[];
  isSuperAdmin: boolean;
}

/** POST /admin/users/{id}/role — super-admins move an account between
 * traveller and vendor (server/internal/http/admin_superadmin.go). */
export interface AdminChangeRoleRequest {
  role: 'traveller' | 'vendor';
  reason: string;
  /** Becoming a vendor: join an existing business with a team role… */
  vendorId?: string;
  vendorRole?: VendorRole;
  /** …or own a new pending business. */
  newBusiness?: { name: string; category: string; town: string };
  /** Required when they own a business that still has other members. */
  newOwnerId?: string;
}

export interface AdminChangeRoleResponse {
  user: User;
  /** Google/Apple-only sign-up: they need "Forgot password" before they
   * can sign in to the vendor site. */
  noPassword: boolean;
}

/** One scheduled day (GET /admin/quests/schedules). */
export interface QuestSchedule {
  date: string;
  vendorId: string;
  goal: number;
  sponsorName?: string;
  rewardDescription?: string;
  completions: number;
  redemptions: number;
}

export interface GrowthBlob {
  flash?: { radiusKm: number; inRange: number; nomads: number; surfers: number; lastSentMins: number; discountPct?: number };
  stampCard?: {
    reward: string;
    stampsNeeded: number;
    buckets: { away: number; count: number; label: string }[];
    totalHolders: number;
    redeemedThisMonth: number;
  };
  btsClips?: { id: string; title: string; secs: number; views: number; grad: string; icon: string }[];
  boost?: {
    event: { name: string; when: string; dj: string };
    tiers: { id: string; label: string; reach: string; days: number; price: number; icon: string; best?: boolean }[];
  };
}

export interface Region {
  id: string;
  name: string;
  x: number;
  y: number;
  lat: number;
  lng: number;
  /** On GET /admin/regions: traveller check-ins, reviews and saves on
   * places in this town in the last 30 days. */
  travelers: number;
  /** On GET /admin/regions: live (approved) businesses in this town. */
  vendors: number;
}

export interface Subscription {
  id: string;
  vendorId?: string;
  vendor: string;
  town: string;
  tier: string;
  mrr: number;
  renewsAt?: string;
  state: string;
  since: string;
}

/** Travel content written in the Command Center (/studio/content) and read
 * by travellers (/content). */
export type ContentType = 'audio' | 'blog' | 'guide' | 'itinerary' | 'tip';
export type ContentStatus = 'draft' | 'in_review' | 'published' | 'rejected' | 'archived';

export interface ContentChapter {
  title: string;
  startSec: number;
}

/** A Spotto place (placeId) or a named point, optionally at a time of day. */
export interface ItineraryStop {
  placeId?: string;
  name: string;
  lat?: number;
  lng?: number;
  /** "09:30" */
  time?: string;
  note?: string;
}

export interface ItineraryDay {
  title: string;
  stops: ItineraryStop[];
}

/** One line of an audio guide's transcript, timed in seconds. */
export interface TranscriptSegment {
  start: number;
  end: number;
  text: string;
}

export type TranscriptStatus = 'queued' | 'working' | 'done' | 'failed';

/** Written by the transcriber service after an audio upload. Travellers only
 * get a finished one, without status, edited or error. */
export interface ContentTranscript {
  status?: TranscriptStatus;
  /** Detected language code, e.g. "en". */
  language: string;
  durationSec: number;
  segments: TranscriptSegment[];
  /** Someone corrected the lines. */
  edited?: boolean;
  /** Why it failed (status failed). */
  error?: string;
  updatedAt: string;
}

export interface ContentPlace {
  id: string;
  name: string;
  category: string;
  lat: number;
  lng: number;
  cover?: string;
}

export interface ContentItem {
  id: string;
  type: ContentType;
  slug: string;
  title: string;
  summary: string;
  /** Markdown; left out of list responses. */
  body?: string;
  /** Signed /media/raw paths (resolveMediaUrl), or "". */
  coverUrl: string;
  audioUrl: string;
  audioSeconds: number;
  chapters: ContentChapter[];
  placeIds: string[];
  days: ItineraryDay[];
  tags: string[];
  regionId: string | null;
  regionName: string;
  regionLat?: number;
  regionLng?: number;
  language: string;
  authorId: string | null;
  authorName: string;
  status: ContentStatus;
  /** Why an admin sent it back (status rejected). */
  reviewNote: string;
  reviewedBy: string | null;
  submittedAt: string | null;
  publishedAt: string | null;
  createdAt: string;
  updatedAt: string;
  views: number;
  plays: number;
  /** A live post's linked places (GET /content/{key} only). */
  places?: ContentPlace[];
  /** Audio posts: the transcript, if one was made (the Command Center also
   * gets it while it's being made, or when it failed). */
  transcript?: ContentTranscript;
}

/** GET /studio/content */
export interface StudioContentList {
  items: ContentItem[];
  /** Posts per status (the caller's own for publishers). */
  counts: Partial<Record<ContentStatus, number>>;
}

/** GET /studio/regions — the towns a post can be about. */
export interface StudioTown {
  id: string;
  name: string;
  lat: number;
  lng: number;
}

/** GET /admin/search?q=... — grouped results across the entities the
 * command bar's search box can match by name (vendors, subscriptions). */
export interface AdminSearchResults {
  vendors: Vendor[];
  subscriptions: Subscription[];
}

/** GET /admin/notifications — derived live from current pending-vendor and
 * bad-review data, same shape as VendorNotification (see server's
 * domain.AdminNotification doc comment). */
export interface AdminNotification {
  id: string;
  icon: string;
  tone: string;
  title: string;
  sub: string;
  time: string;
  unread: boolean;
}

/** A single entry from GET /admin/audit-log — one destructive action a
 * platform admin took, with a denormalized actor identity so entries stay
 * readable even if that admin account is later deleted. */
export interface AdminAuditLogEntry {
  id: number;
  actorId: string;
  actorEmail: string;
  actorName: string;
  action: string;
  targetType: string;
  targetId: string;
  metadata?: unknown;
  createdAt: string;
}

/** GET/PUT /admin/settings/phone-codes — the daily limits on phone
 * verification codes (each a paid SMS/WhatsApp message), kept apart for
 * Sri Lankan and foreign numbers, and how many went out in the last 24
 * hours. Only super-admins may change the limits. */
export interface AdminPhoneCodeSettings {
  limitSriLanka: number;
  limitOther: number;
  sentSriLanka: number;
  sentOther: number;
}

export interface AdminKPIs {
  pendingVendors: number;
  mrr: number;
  activeSubs: number;
}

/** One vendor's live activity roll-up, for the admin Vendors monitor pane.
 * Derived on read from claims/checkins/reviews/offers — not a
 * stored snapshot, so it never goes stale. */
export interface AdminVendorStats {
  claimed: number;
  redeemed: number;
  redeemed30d: number;
  checkins: number;
  checkins30d: number;
  reviews: number;
  avgRating: number;
  badReviews: number;
  flagged: boolean;
  offers: number;
  offersLive: number;
  offersDraft: number;
  teamTotal: number;
  teamManagers: number;
  teamStaff: number;
  pendingInvites: number;
  /** Absent for vendors with no subscriptions row (typical for Free tier). */
  subTier?: string;
  subMrr?: number;
  subState?: string;
  subRenewsAt?: string;
}

/** One heat-map layer on the admin geo page — GET /admin/geo/heat. `previous*`
 * is the period before the chosen one (absent meaning for `trips`, which
 * looks ahead). */
export interface AdminGeoHeat {
  layer: 'travellers' | 'activity' | 'opportunity' | 'businesses' | 'wanted' | 'trips';
  days: number;
  points: { lat: number; lng: number; w: number }[];
  total: number;
  previousTotal: number;
  regions: AdminGeoTown[];
  /** Activity layers only: actions per type in the period. */
  breakdown?: Record<string, number>;
  /** Wanted layer only: most-saved places that aren't on Spotto. */
  wanted?: AdminWantedPlace[];
}

export interface AdminGeoTown {
  id: string;
  name: string;
  lat: number;
  lng: number;
  value: number;
  previous: number;
  /** Live (approved) businesses near the town, in the chosen category. */
  businesses: number;
  /** A traveller count too small to show (under 3). */
  hidden?: boolean;
}

export interface AdminWantedPlace {
  id: string;
  name: string;
  city: string;
  kind: string;
  lat: number;
  lng: number;
  mapsUrl: string;
  savers: number;
  previous: number;
  lastSaved: string;
}

export interface AdminGeoFilters {
  countries: { value: string; count: number }[];
  styles: { value: string; count: number }[];
  activityTypes: string[];
}

/** Everything within radiusKm of a clicked point — GET /admin/geo/area. */
export interface AdminGeoArea {
  lat: number;
  lng: number;
  radiusKm: number;
  days: number;
  travellers: number;
  travellersHidden?: boolean;
  activity: Record<string, number>;
  liveBusinesses: number;
  newBusinesses: number;
  wantedPlaces: number;
  upcomingStops: number;
}

/** Network-wide vendor roll-up behind the Vendors page's aggregate band —
 * the same activity metrics as AdminVendorStats summed across every vendor,
 * plus a status breakdown. `flagged` counts vendors, not reviews. */
export interface AdminVendorSummary {
  total: number;
  pending: number;
  approved: number;
  edits: number;
  suspended: number;
  rejected: number;
  flagged: number;
  redeemed: number;
  redeemed30d: number;
  checkins: number;
  checkins30d: number;
  reviews: number;
  avgRating: number;
  badReviews: number;
  offers: number;
  offersLive: number;
  offersDraft: number;
  activeSubs: number;
  mrr: number;
}

/** A still-pending team invite an owner generated — GET
 * /admin/vendors/{id}/team (and the vendor-side team endpoint). */
export interface VendorInvite {
  id: string;
  vendorId: string;
  code: string;
  email?: string;
  role: string;
  createdAt: string;
  expiresAt: string;
}

/** Team tab payload for the admin vendor detail: staff roster, pending
 * invites, and any profile-access PIN grants for this vendor. */
/** POST /admin/vendors/{id}/members/{userId}/login result. */
export interface VendorLoginHandoverResult {
  mode: 'password' | 'link' | 'none';
  email: string;
  emailSent: boolean;
  /** Returned when the set-password email couldn't be sent, to pass on by hand. */
  setupLink?: string;
  emailError?: string;
}

export interface AdminVendorTeam {
  members: User[];
  invites: VendorInvite[];
  grants: AdminVendorProfileGrant[];
}

/** Local Guide dashboard's aggregate summary — GET /admin/local-guide/stats.
 * The 7d-suffixed fields are rolling-window counts, not calendar weeks. */
export interface AdminGuideStats {
  totalConversations: number;
  totalMessages: number;
  messagesToday: number;
  messages7d: number;
  requests7d: number;
  okCount7d: number;
  rateLimitedCount7d: number;
  errorCount7d: number;
  avgLatencyMs7d: number;
  semanticQueryCount7d: number;
  placesEmbedded: number;
  placesTotal: number;
}

export interface AdminGuideToolUsage {
  tool: string;
  count: number;
}

/** GET /admin/local-guide/stats' full response. */
export interface AdminGuideStatsResponse {
  stats: AdminGuideStats;
  toolUsage: AdminGuideToolUsage[];
  rateLimitRPM: number;
  /** false when GEMINI_API_KEY isn't set on the server. */
  configured: boolean;
  models: { chat: string; embed: string };
  aiUsage: AdminAIUsageRow[];
  /** 14 Pacific-time days, oldest first, zero-filled. */
  daily: AdminAIDailyPoint[];
  quota: AdminAIQuota;
  quotaUsed: AdminAIQuotaUsage;
  embeddings: AdminEmbeddingStats;
  /** In-memory snapshot of the embedding worker's last sweep (resets on deploy). */
  embedWorker: { lastRunAt?: string; lastError: string; lastEmbedded: number; pausedUntil?: string };
}

/** One feature+model's Gemini calls over today (Pacific) / 7d / 30d. */
export interface AdminAIUsageRow {
  feature: string;
  kind: 'chat' | 'embed';
  model: string;
  requestsToday: number;
  requests7d: number;
  requests30d: number;
  errors7d: number;
  rateLimited7d: number;
  inputTokensToday: number;
  outputTokensToday: number;
  inputTokens7d: number;
  outputTokens7d: number;
  inputTokens30d: number;
  outputTokens30d: number;
  /** Embedding token counts are a characters/4 estimate — the API reports none. */
  tokensEstimated: boolean;
}

export interface AdminAIDailyPoint {
  day: string;
  requests: number;
  inputTokens: number;
  outputTokens: number;
}

/** Admin-entered daily limits — Gemini has no remaining-quota API. */
export interface AdminAIQuota {
  chatRequestsPerDay: number;
  chatTokensPerDay: number;
  embedRequestsPerDay: number;
}

export interface AdminAIQuotaUsage {
  chatRequestsToday: number;
  chatTokensToday: number;
  embedRequestsToday: number;
}

export interface AdminEmbeddingStats {
  model: string;
  total: number;
  /** Embedded with the current model. */
  embedded: number;
  /** Never embedded, or embedded with a previous model. */
  pending: number;
}

/** POST /admin/local-guide/health — one live call per model. */
export interface AdminAIHealthCheck {
  model: string;
  ok: boolean;
  latencyMs: number;
  error?: string;
}

export interface AdminAIHealthResponse {
  chat: AdminAIHealthCheck;
  embed: AdminAIHealthCheck;
}

/** One row in the Local Guide dashboard's browsable conversation list —
 * GET /admin/local-guide/conversations. */
export interface AdminGuideUserSummary {
  travellerId: string;
  travellerName: string;
  totalMessages: number;
  errorCount: number;
  inputTokens: number;
  outputTokens: number;
  tripsPlanned: number;
  lastActiveAt: string;
}

/** One turn in a Local Guide conversation — GET
 * /admin/local-guide/conversations/{id} */
export interface AdminGuideMessage {
  conversationId: string;
  role: 'user' | 'model';
  content: string;
  placeIds?: string[];
  /** Card shown with a trip-planner reply (server guide_planner.go). */
  payload?: AdminGuidePayload;
  createdAt: string;
}

export interface AdminGuidePayload {
  type: 'questions' | 'plan' | 'calendar';
  questions?: { questions: { id: string; text: string }[] };
  plan?: { title: string; days: { stops: unknown[] }[] };
  calendar?: { mode: string; stops: unknown[] };
}

/** One failed Gemini call from any feature — GET /admin/local-guide/errors.
 * The traveller only ever sees a generic "unexpected error" message, so this
 * is the real upstream failure reason (auth, quota, retired model, etc.). */
export interface AdminGuideErrorEntry {
  feature: string;
  model: string;
  status: 'error' | 'rate_limited';
  errorMessage: string;
  createdAt: string;
}

/** "12 min ago" style rendering for server timestamps. */
export function timeAgo(iso: string | undefined): string {
  if (!iso) return '';
  const s = Math.max(0, (Date.now() - new Date(iso).getTime()) / 1000);
  if (s < 60) return 'just now';
  const m = Math.floor(s / 60);
  if (m < 60) return `${m} min ago`;
  const h = Math.floor(m / 60);
  if (h < 24) return `${h} hr ago`;
  const d = Math.floor(h / 24);
  return d === 1 ? 'yesterday' : `${d} days ago`;
}
