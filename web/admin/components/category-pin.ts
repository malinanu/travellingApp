// A business's map-pin look — its category group's colour and the
// category's icon — shared by the geo map's pins (vendor-map.tsx) and the
// geo page's lists and details, so the two can't drift apart. No Leaflet
// here, for the same reason as region-tone.ts: the page imports this
// statically, while the map is lazy-loaded.

import * as Icons from 'lucide-react';

import { PIN_TONE_COLORS, type VendorCategory } from '@spotto/shared';

/** Pins whose category group or icon the map doesn't know. */
export const OTHER_PIN_COLOR = '#64748b';

const LucideIcons = Icons as unknown as Record<string, Icons.LucideIcon>;

/** Whether a category draws with its own group colour (else the grey "Other"). */
export function knownPinGroup(category: VendorCategory | undefined): boolean {
  return !!category && category.segment in PIN_TONE_COLORS;
}

export function pinColor(category: VendorCategory | undefined): string {
  return category && knownPinGroup(category) ? PIN_TONE_COLORS[category.segment] : OTHER_PIN_COLOR;
}

/** The category's Lucide icon name, or 'Store' when it has none we know. */
export function pinIconName(category: VendorCategory | undefined): string {
  return category?.icon && category.icon in LucideIcons ? category.icon : 'Store';
}

export function pinIcon(category: VendorCategory | undefined): Icons.LucideIcon {
  return LucideIcons[pinIconName(category)] ?? Icons.Store;
}
