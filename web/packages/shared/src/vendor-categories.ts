// Canonical vendor-category taxonomy — the single source of truth for both
// web/vendor and web/admin, mirroring the business plan's 6 stakeholder
// segments. Mirrors server/internal/domain/vendor_category.go and
// app/lib/data/vendor_categories.dart exactly (id/label/segment/icon name/
// relevantVibeDimensions) — keep all three in sync if any change.
//
// This package has no UI-framework dependency, so `icon` is a Lucide icon
// *name* (string), not a component — each app resolves it locally
// (web/vendor/lib/business-categories.ts owns the lucide-react import).
//
// Replaces what used to be 5 independently hardcoded, drifting category
// lists across this codebase.

export type VendorSegment =
  | 'hospitality'
  | 'activity'
  | 'independent'
  | 'events'
  | 'brand'
  | 'future';

export interface VendorSegmentInfo {
  id: VendorSegment;
  label: string;
}

// "future" (accommodation/flight/insurance/transfer/booking platforms) is
// deliberately excluded — affiliate-only per the business plan, not an
// active onboarding type. See onboardableCategories().
export const VENDOR_SEGMENTS: VendorSegmentInfo[] = [
  { id: 'hospitality', label: 'Hospitality' },
  { id: 'activity', label: 'Activities & Experiences' },
  { id: 'independent', label: 'Independent & Local' },
  { id: 'events', label: 'Events' },
  { id: 'brand', label: 'Brand Partner' },
];

export type BookingType = 'session' | 'table' | 'stay';

export interface VendorCategory {
  id: string;
  label: string;
  segment: VendorSegment;
  /** Lucide icon name — resolve via a local icon-name→component map. */
  icon: string;
  /** Subset of "productive"/"loud"/"crowded" meaningful for this category
   * (e.g. "productive" doesn't mean anything for a beach bar). Omitted for
   * non-venue categories (brand partners) where vibe is irrelevant. */
  relevantVibeDimensions?: string[];
  /** Which real-booking flow this category's vendors use, if any — undefined
   * means not bookable (walk-in businesses, brand partners, event
   * categories — Events isn't a shipped feature yet). Mirrors
   * server/internal/domain/vendor_category.go's BookingType exactly. */
  bookingType?: BookingType;
}

export const VENDOR_CATEGORIES: VendorCategory[] = [
  // Hospitality vendors — cafes, restaurants, bars, hotels, beach clubs.
  { id: 'cafe', label: 'Café', segment: 'hospitality', icon: 'Coffee', relevantVibeDimensions: ['productive', 'loud', 'crowded'], bookingType: 'table' },
  { id: 'coworking-cafe', label: 'Coworking Café', segment: 'hospitality', icon: 'Laptop', relevantVibeDimensions: ['productive', 'loud', 'crowded'], bookingType: 'table' },
  { id: 'restaurant', label: 'Restaurant', segment: 'hospitality', icon: 'UtensilsCrossed', relevantVibeDimensions: ['loud', 'crowded'], bookingType: 'table' },
  { id: 'street-food', label: 'Street Food', segment: 'hospitality', icon: 'Soup', relevantVibeDimensions: ['crowded'] },
  { id: 'bakery', label: 'Bakery', segment: 'hospitality', icon: 'Cookie', relevantVibeDimensions: ['crowded'] },
  { id: 'bar-pub', label: 'Bar & Pub', segment: 'hospitality', icon: 'Martini', relevantVibeDimensions: ['loud', 'crowded'], bookingType: 'table' },
  { id: 'club', label: 'Club / Nightlife', segment: 'hospitality', icon: 'Music', relevantVibeDimensions: ['loud', 'crowded'], bookingType: 'table' },
  { id: 'hotel', label: 'Hotel', segment: 'hospitality', icon: 'Building2', relevantVibeDimensions: ['crowded'], bookingType: 'stay' },
  { id: 'hostel', label: 'Hostel / Guesthouse', segment: 'hospitality', icon: 'BedSingle', relevantVibeDimensions: ['loud', 'crowded'], bookingType: 'stay' },
  { id: 'resort', label: 'Resort', segment: 'hospitality', icon: 'Palmtree', relevantVibeDimensions: ['crowded'], bookingType: 'stay' },
  { id: 'beach-club', label: 'Beach Club', segment: 'hospitality', icon: 'Waves', relevantVibeDimensions: ['loud', 'crowded'], bookingType: 'table' },

  // Activity and experience providers — surf/dive schools, yoga, tours.
  { id: 'surf-school', label: 'Surf School', segment: 'activity', icon: 'Waves', relevantVibeDimensions: ['crowded'], bookingType: 'session' },
  { id: 'dive-school', label: 'Diving / Dive Shop', segment: 'activity', icon: 'Anchor', relevantVibeDimensions: ['crowded'], bookingType: 'session' },
  { id: 'yoga-studio', label: 'Yoga Studio', segment: 'activity', icon: 'Flower2', relevantVibeDimensions: ['loud', 'crowded'], bookingType: 'session' },
  { id: 'spa-wellness', label: 'Spa & Wellness', segment: 'activity', icon: 'Sparkles', relevantVibeDimensions: ['loud', 'crowded'], bookingType: 'session' },
  { id: 'cooking-class', label: 'Cooking Class', segment: 'activity', icon: 'ChefHat', relevantVibeDimensions: ['crowded'], bookingType: 'session' },
  { id: 'pottery-workshop', label: 'Pottery Workshop', segment: 'activity', icon: 'Hammer', relevantVibeDimensions: ['crowded'], bookingType: 'session' },
  { id: 'batik-atelier', label: 'Batik / Textile Atelier', segment: 'activity', icon: 'Shirt', relevantVibeDimensions: ['crowded'], bookingType: 'session' },
  { id: 'spice-trail', label: 'Spice Trail', segment: 'activity', icon: 'Leaf', relevantVibeDimensions: ['crowded'], bookingType: 'session' },
  { id: 'safari-wildlife', label: 'Safari & Wildlife', segment: 'activity', icon: 'PawPrint', relevantVibeDimensions: ['crowded'], bookingType: 'session' },
  { id: 'boat-trip', label: 'Boat Trip', segment: 'activity', icon: 'Ship', relevantVibeDimensions: ['crowded'], bookingType: 'session' },
  { id: 'city-tour', label: 'City / Cultural Tour', segment: 'activity', icon: 'Compass', relevantVibeDimensions: ['crowded'], bookingType: 'session' },

  // Independent local providers — guides, artisans, tailors, solo hosts.
  { id: 'tour-guide', label: 'Tour / Walking Guide', segment: 'independent', icon: 'Compass', relevantVibeDimensions: ['crowded'], bookingType: 'session' },
  { id: 'artisan-shop', label: 'Local Artisan / Craft Shop', segment: 'independent', icon: 'Palette', relevantVibeDimensions: ['crowded'] },
  { id: 'tailor-clothing', label: 'Tailor / Local Clothing', segment: 'independent', icon: 'Shirt', relevantVibeDimensions: ['crowded'] },
  { id: 'experience-host', label: 'Independent Experience Host', segment: 'independent', icon: 'Backpack', relevantVibeDimensions: ['crowded'], bookingType: 'session' },

  // Event promoters — festivals, parties, competitions, community events.
  { id: 'festival-event', label: 'Festival', segment: 'events', icon: 'PartyPopper', relevantVibeDimensions: ['loud', 'crowded'] },
  { id: 'party-promoter', label: 'Party Promotion', segment: 'events', icon: 'PartyPopper', relevantVibeDimensions: ['loud', 'crowded'] },
  { id: 'surf-competition', label: 'Surf Competition', segment: 'events', icon: 'Waves', relevantVibeDimensions: ['loud', 'crowded'] },
  { id: 'community-charity-event', label: 'Community / Charity Event', segment: 'events', icon: 'HeartHandshake', relevantVibeDimensions: ['crowded'] },
  { id: 'music-night', label: 'Music Night', segment: 'events', icon: 'Music', relevantVibeDimensions: ['loud', 'crowded'] },

  // Brand partners — sponsors, not physical venues, so vibe is irrelevant.
  { id: 'beverage-brand', label: 'Beverage Brand', segment: 'brand', icon: 'GlassWater' },
  { id: 'alcohol-brand', label: 'Beer & Alcohol Brand', segment: 'brand', icon: 'Wine' },
  { id: 'food-brand', label: 'Food Brand', segment: 'brand', icon: 'UtensilsCrossed' },
  { id: 'lifestyle-brand', label: 'Lifestyle Brand', segment: 'brand', icon: 'Sparkles' },
  { id: 'tourism-company', label: 'Tourism Company', segment: 'brand', icon: 'Plane' },

  // "future" segment deliberately has no categories — see VENDOR_SEGMENTS.
];

export function categoryById(id: string | undefined): VendorCategory | undefined {
  const norm = (id ?? '').toLowerCase().trim();
  return VENDOR_CATEGORIES.find((c) => c.id.toLowerCase() === norm);
}

export function categoriesBySegment(segment: VendorSegment): VendorCategory[] {
  return VENDOR_CATEGORIES.filter((c) => c.segment === segment);
}

/** Every category outside "future" — what a registration/profile picker or
 * admin dropdown should offer. */
export function onboardableCategories(): VendorCategory[] {
  return VENDOR_CATEGORIES.filter((c) => c.segment !== 'future');
}

/** Whether this category's vendors can take real bookings through the
 * platform — drives the vendor portal's Services/Bookings nav visibility. */
export function isBookableCategory(category: string | undefined): boolean {
  return !!categoryById(category)?.bookingType;
}
