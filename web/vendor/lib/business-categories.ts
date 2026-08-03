// Vendor portal's category picker (Business profile page + registration
// wizard). The canonical taxonomy (ids/labels/segments) now lives in
// @spotto/shared's vendor-categories.ts — this file just resolves each
// category's icon *name* to a real lucide-react component, since the
// shared package has no UI-framework dependency.
import {
  Anchor, Backpack, BedSingle, Building2, ChefHat, Coffee, Compass, Cookie,
  Flower2, GlassWater, Hammer, HeartHandshake, Laptop, Leaf, Martini, Music,
  Palette, Palmtree, PartyPopper, PawPrint, Plane, Ship, Shirt, Soup,
  Sparkles, UtensilsCrossed, Waves, Wine, LucideIcon,
} from 'lucide-react';
import {
  VendorCategory, VENDOR_SEGMENTS, VENDOR_CATEGORIES,
  categoryById, categoriesBySegment, onboardableCategories,
} from '@spotto/shared';

export type { VendorCategory };
export { VENDOR_SEGMENTS, VENDOR_CATEGORIES, categoryById, categoriesBySegment, onboardableCategories };

/** Re-export under the old name for existing call sites during rollout. */
export const BUSINESS_CATEGORIES = onboardableCategories();

const ICON_COMPONENTS: Record<string, LucideIcon> = {
  Anchor, Backpack, BedSingle, Building2, ChefHat, Coffee, Compass, Cookie,
  Flower2, GlassWater, Hammer, HeartHandshake, Laptop, Leaf, Martini, Music,
  Palette, Palmtree, PartyPopper, PawPrint, Plane, Ship, Shirt, Soup,
  Sparkles, UtensilsCrossed, Waves, Wine,
};

/** Resolves a category's icon name to its lucide-react component, falling
 * back to a generic icon for anything unrecognized. */
export function categoryIconComponent(category: VendorCategory): LucideIcon {
  return ICON_COMPONENTS[category.icon] ?? Compass;
}

/** Now a plain id comparison — categories are canonical ids after the
 * taxonomy redesign, no more label/case tolerance needed. */
export function isSelectedCategory(current: string | undefined, t: VendorCategory) {
  return (current ?? '').toLowerCase().trim() === t.id.toLowerCase();
}
