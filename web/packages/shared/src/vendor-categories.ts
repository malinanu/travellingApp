// Canonical vendor-category taxonomy — the single source of truth for both
// web/vendor and web/admin, mirroring server/internal/seed/data.go (the
// real, DB-backed taxonomy served at GET /categories) and
// app/lib/data/vendor_categories.dart exactly (id/label/segment/icon name/
// relevantVibeDimensions/bookingType) — keep all three in sync if any
// change. This file previously held an entirely different, disjoint
// taxonomy (hospitality/activity/independent/events/brand ids like
// "cafe"/"coworking-cafe") that vendor onboarding and admin vendor editing
// wrote into vendors.category — a value the app's own category lookup
// (populated at runtime from GET /categories) could never resolve, so
// those vendors' map pins always fell back to the generic pin. Replaced
// 2026-09-23 to close that gap.
//
// This package has no UI-framework dependency, so `icon` is a Lucide icon
// *name* (string), not a component — each app resolves it locally
// (web/vendor/lib/business-categories.ts owns the lucide-react import).

export type VendorSegment =
  | 'food_and_drink'
  | 'things_to_do'
  | 'shopping'
  | 'services'
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
  { id: 'food_and_drink', label: 'Food & Drink' },
  { id: 'things_to_do', label: 'Things to do' },
  { id: 'shopping', label: 'Shopping' },
  { id: 'services', label: 'Services' },
  { id: 'brand', label: 'Brand Partner' },
];

/** Place.color pin-tone key per segment — mirrors
 * server/internal/domain/vendor_category.go's PinColorForSegment and
 * app/lib/core/theme/palette.dart's pinTones exactly. Keep all three in
 * sync if either changes. */
export const PIN_TONE_COLORS: Record<VendorSegment, string> = {
  food_and_drink: '#F0682F', // coral
  things_to_do: '#EAA200', // sun
  shopping: '#7C5CFF', // plum
  services: '#C2683F', // clay
  brand: '#0B6F86', // teal
  future: '#0B6F86', // teal
};

export type BookingType = 'session' | 'table' | 'stay' | 'rental';

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
  // Food & Drink
  { id: 'restaurants', label: 'Restaurants', segment: 'food_and_drink', icon: 'UtensilsCrossed', relevantVibeDimensions: ['loud', 'crowded'], bookingType: 'table' },
  { id: 'takeout', label: 'Takeout', segment: 'food_and_drink', icon: 'ShoppingBag', relevantVibeDimensions: ['crowded'] },
  { id: 'bars', label: 'Bars', segment: 'food_and_drink', icon: 'Martini', relevantVibeDimensions: ['loud', 'crowded'], bookingType: 'table' },
  { id: 'delivery', label: 'Delivery', segment: 'food_and_drink', icon: 'Truck', relevantVibeDimensions: [] },
  { id: 'coffee', label: 'Coffee', segment: 'food_and_drink', icon: 'Coffee', relevantVibeDimensions: ['productive', 'loud', 'crowded'], bookingType: 'table' },

  // Things to do
  { id: 'parks', label: 'Parks', segment: 'things_to_do', icon: 'TreePine', relevantVibeDimensions: ['crowded'] },
  { id: 'gyms', label: 'Gyms', segment: 'things_to_do', icon: 'Dumbbell', relevantVibeDimensions: ['loud', 'crowded'], bookingType: 'session' },
  { id: 'art', label: 'Art', segment: 'things_to_do', icon: 'Image', relevantVibeDimensions: ['crowded'] },
  { id: 'attractions', label: 'Attractions', segment: 'things_to_do', icon: 'Ticket', relevantVibeDimensions: ['crowded'], bookingType: 'session' },
  { id: 'nightlife', label: 'Nightlife', segment: 'things_to_do', icon: 'Music', relevantVibeDimensions: ['loud', 'crowded'], bookingType: 'table' },
  { id: 'live-music', label: 'Live music', segment: 'things_to_do', icon: 'Mic', relevantVibeDimensions: ['loud', 'crowded'], bookingType: 'table' },
  { id: 'movies', label: 'Movies', segment: 'things_to_do', icon: 'Film', relevantVibeDimensions: ['crowded'], bookingType: 'session' },
  { id: 'museums', label: 'Museums', segment: 'things_to_do', icon: 'Landmark', relevantVibeDimensions: ['crowded'], bookingType: 'session' },
  { id: 'libraries', label: 'Libraries', segment: 'things_to_do', icon: 'Library', relevantVibeDimensions: ['productive', 'crowded'] },

  // Shopping
  { id: 'groceries', label: 'Groceries', segment: 'shopping', icon: 'ShoppingCart', relevantVibeDimensions: ['crowded'] },
  { id: 'beauty-supplies', label: 'Beauty supplies', segment: 'shopping', icon: 'Gem', relevantVibeDimensions: ['crowded'] },
  { id: 'car-dealers', label: 'Car dealers', segment: 'shopping', icon: 'Car', relevantVibeDimensions: ['crowded'] },
  { id: 'home-garden', label: 'Home & garden', segment: 'shopping', icon: 'Home', relevantVibeDimensions: ['crowded'] },
  { id: 'apparel', label: 'Apparel', segment: 'shopping', icon: 'Shirt', relevantVibeDimensions: ['crowded'] },
  { id: 'shopping-centers', label: 'Shopping centers', segment: 'shopping', icon: 'Store', relevantVibeDimensions: ['crowded'] },
  { id: 'electronics', label: 'Electronics', segment: 'shopping', icon: 'Tv', relevantVibeDimensions: ['crowded'] },
  { id: 'sporting-goods', label: 'Sporting goods', segment: 'shopping', icon: 'Trophy', relevantVibeDimensions: ['crowded'] },
  { id: 'convenience-stores', label: 'Convenience stores', segment: 'shopping', icon: 'Package', relevantVibeDimensions: ['crowded'] },

  // Services
  { id: 'hotels', label: 'Hotels', segment: 'services', icon: 'Building2', relevantVibeDimensions: ['crowded'], bookingType: 'stay' },
  { id: 'atms', label: 'ATMs', segment: 'services', icon: 'Banknote', relevantVibeDimensions: ['crowded'] },
  { id: 'beauty-salons', label: 'Beauty salons', segment: 'services', icon: 'Scissors', relevantVibeDimensions: ['crowded'], bookingType: 'session' },
  { id: 'car-rental', label: 'Car rental', segment: 'services', icon: 'Key', relevantVibeDimensions: ['crowded'], bookingType: 'rental' },
  { id: 'car-wash', label: 'Car wash', segment: 'services', icon: 'Droplets', relevantVibeDimensions: ['crowded'] },
  { id: 'dry-cleaning', label: 'Dry cleaning', segment: 'services', icon: 'WashingMachine', relevantVibeDimensions: ['crowded'] },
  { id: 'charging-stations', label: 'Charging stations', segment: 'services', icon: 'Zap', relevantVibeDimensions: ['crowded'] },
  { id: 'gas', label: 'Gas', segment: 'services', icon: 'Fuel', relevantVibeDimensions: ['crowded'] },
  { id: 'hospitals-clinics', label: 'Hospitals & clinics', segment: 'services', icon: 'Stethoscope', relevantVibeDimensions: ['crowded'], bookingType: 'session' },
  { id: 'mail-shipping', label: 'Mail & shipping', segment: 'services', icon: 'Mail', relevantVibeDimensions: ['crowded'] },
  { id: 'parking', label: 'Parking', segment: 'services', icon: 'SquareParking', relevantVibeDimensions: ['crowded'] },
  { id: 'pharmacies', label: 'Pharmacies', segment: 'services', icon: 'Pill', relevantVibeDimensions: ['crowded'] },

  // Brand
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
