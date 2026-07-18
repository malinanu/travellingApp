// Hand-maintained TS mirror of the Go API's JSON shapes (see server/internal/domain).

export type Role = 'traveller' | 'vendor' | 'admin';
export type VendorStatus = 'pending' | 'approved' | 'rejected' | 'edits' | 'suspended';
export type OfferStatus = 'draft' | 'live' | 'ending' | 'ended';
export type ClaimStatus = 'claimed' | 'redeemed';
export type FeedStatus = 'ok' | 'review' | 'flag';

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
  totpEnabled: boolean;
  phone: string;
  shareLocation: boolean;
  notifyRedemptions: boolean;
  notifyReviews: boolean;
  notifyPayouts: boolean;
  notifyGrowthTips: boolean;
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
  menu: { sections?: MenuSection[] } | null;
  submittedAt: string;
  lat: number;
  lng: number;
  badReviewCount?: number;
  flagged?: boolean;
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
}

export interface MenuSection {
  id: string;
  name: string;
  items: MenuItem[];
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
  redeemedAt?: string;
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

export interface FeedItem {
  id: number;
  vendor: string;
  town: string;
  type: string;
  text: string;
  status: FeedStatus;
  reason?: string;
  resolved?: string;
  createdAt: string;
}

export interface Region {
  id: string;
  name: string;
  x: number;
  y: number;
  lat: number;
  lng: number;
  travelers: number;
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

export interface AudioGuide {
  id: string;
  site: string;
  region: string;
  lat: number;
  lng: number;
  lang: string;
  published: boolean;
  plays: number;
  depths: string[];
  duration: Record<string, string>;
}

export interface AdminKPIs {
  pendingVendors: number;
  feedItems: number;
  mrr: number;
  activeSubs: number;
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
