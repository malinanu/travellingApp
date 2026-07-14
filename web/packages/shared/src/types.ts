// Hand-maintained TS mirror of the Go API's JSON shapes (see server/internal/domain).

export type Role = 'traveller' | 'vendor' | 'admin';
export type VendorStatus = 'pending' | 'approved' | 'rejected' | 'edits' | 'suspended';
export type OfferStatus = 'draft' | 'live' | 'ending' | 'ended';
export type ClaimStatus = 'claimed' | 'redeemed';
export type FeedStatus = 'ok' | 'review' | 'flag';

export interface User {
  id: string;
  email: string;
  role: Role;
  name: string;
  country: string;
  travelStyle: string;
  interests: string[] | null;
  vendorId?: string;
  createdAt: string;
  vendorStatus?: VendorStatus; // present on GET /me for vendor users
  totpEnabled: boolean;
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

export interface VendorProfile {
  wifi?: { enabled: boolean; score: number; speed: number };
  bestTimes?: string[];
  seaView?: boolean;
  hours?: { open: string; close: string };
  amenities?: string[];
  blurb?: string;
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
  claimed: number;
  redeemed: number;
}

export interface Claim {
  id: string;
  code: string;
  placeId: string;
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
  flash?: { radiusKm: number; inRange: number; nomads: number; surfers: number; lastSentMins: number };
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
