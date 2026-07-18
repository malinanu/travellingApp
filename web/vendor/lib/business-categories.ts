// Shared business-type taxonomy for the vendor portal's category picker
// (Business profile page + registration wizard). `id` is stored verbatim
// as `Vendor.category` and MUST exactly match (case-insensitively) the
// literal strings this app's data actually contains — including the
// original 6 lowercase ids new signups have always used (restaurant/cafe/
// bar/surf/craft/shop) and the richer free-text categories real/seeded
// vendor applications carry (server/internal/gendata/gendata.go's
// `vendorCategories`: Cafe, Coworking Café, Surf School, Restaurant, Bar,
// Yoga / Wellness, Spa & Wellness, Hostel, Hotel, Pottery Workshop, Batik
// Atelier, Tours / Activity, Street Food, Bakery, Dive Shop). Without this,
// a vendor whose category came from that richer set (e.g. "Spa & Wellness")
// showed no button as selected at all, since none of the original 6
// options could ever match it.
import {
  Anchor, BedSingle, Building2, Coffee, Compass, Croissant, Flower2, Hammer,
  Laptop, Martini, Palette, Shirt, Soup, Sparkles, Store, Utensils, Waves,
  LucideIcon,
} from 'lucide-react';

export interface BusinessCategory {
  id: string;
  label: string;
  icon: LucideIcon;
}

export const BUSINESS_CATEGORIES: BusinessCategory[] = [
  { id: 'restaurant', label: 'Restaurant', icon: Utensils },
  { id: 'cafe', label: 'Café', icon: Coffee },
  { id: 'bar', label: 'Bar & Sunset', icon: Martini },
  { id: 'surf', label: 'Surf & Water', icon: Waves },
  { id: 'craft', label: 'Craft & Studio', icon: Palette },
  { id: 'shop', label: 'Shop', icon: Store },
  { id: 'Coworking Café', label: 'Coworking Café', icon: Laptop },
  { id: 'Surf School', label: 'Surf School', icon: Waves },
  { id: 'Yoga / Wellness', label: 'Yoga / Wellness', icon: Flower2 },
  { id: 'Spa & Wellness', label: 'Spa & Wellness', icon: Sparkles },
  { id: 'Hostel', label: 'Hostel', icon: BedSingle },
  { id: 'Hotel', label: 'Hotel', icon: Building2 },
  { id: 'Pottery Workshop', label: 'Pottery Workshop', icon: Hammer },
  { id: 'Batik Atelier', label: 'Batik Atelier', icon: Shirt },
  { id: 'Tours / Activity', label: 'Tours / Activity', icon: Compass },
  { id: 'Street Food', label: 'Street Food', icon: Soup },
  { id: 'Bakery', label: 'Bakery', icon: Croissant },
  { id: 'Dive Shop', label: 'Dive Shop', icon: Anchor },
];

/** Case-insensitive match against a category's id or label — tolerant of
 * both the lowercase ids new signups store and the free-text casing
 * gendata/admin-assigned categories use. */
export function isSelectedCategory(current: string | undefined, t: BusinessCategory) {
  const c = (current ?? '').toLowerCase().trim();
  return c === t.id.toLowerCase() || c === t.label.toLowerCase();
}
