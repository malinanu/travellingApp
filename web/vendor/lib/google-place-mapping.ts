// Small, self-contained helpers for turning a classic Google Places
// Autocomplete PlaceResult (as used by components/address-search-field.tsx)
// into Spotto's own category/hours shapes. Deliberately separate from the
// similarly-named helpers in app/login/page.tsx's registration Step 1
// business search:
//   - categoryFromGoogleTypes here returns undefined when nothing matches,
//     rather than defaulting to 'cafe' — registration always needs *some*
//     category pre-selected, but this module only ever suggests a category
//     when it's actually confident a real business (not just a bare
//     address) was matched, per the "only fill empty fields" auto-fill
//     behavior it feeds (see address-search-field.tsx's onPlaceDetails).
//   - hoursFromLegacyPeriods parses the classic Autocomplete widget's
//     PlaceOpeningHoursPeriod shape (`{ day, time: "0900" }`, a 4-digit
//     string), which differs from the newer Place API's `{ day, hour,
//     minute }` numeric shape that login/page.tsx's hoursFromPeriods
//     parses — different enough that sharing one function isn't practical.

import type { DayHours, Weekday } from '@spotto/shared';

const GOOGLE_TYPE_TO_CATEGORY: [string, string][] = [
  ['cafe', 'cafe'],
  ['restaurant', 'restaurant'],
  ['bar', 'bar'], ['night_club', 'bar'],
  ['bakery', 'Bakery'],
  ['lodging', 'Hotel'],
  ['spa', 'Spa & Wellness'],
  ['gym', 'Yoga / Wellness'],
  ['tourist_attraction', 'Tours / Activity'], ['travel_agency', 'Tours / Activity'],
  ['meal_takeaway', 'Street Food'],
  ['art_gallery', 'craft'],
  ['store', 'shop'], ['clothing_store', 'shop'],
];

export function categoryFromGoogleTypes(types: string[] | undefined): string | undefined {
  for (const [googleType, spottoCategory] of GOOGLE_TYPE_TO_CATEGORY) {
    if (types?.includes(googleType)) return spottoCategory;
  }
  return undefined;
}

// Google numbers days Sunday=0..Saturday=6 in PlaceOpeningHoursPeriod, same
// convention server/internal/http/vendor.go's weekdayKeys already uses.
const GOOGLE_WEEKDAYS: Weekday[] = ['sunday', 'monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday'];

function formatLegacyTime(time: string): string {
  return `${time.slice(0, 2)}:${time.slice(2, 4)}`;
}

export function hoursFromLegacyPeriods(
  periods: google.maps.places.PlaceOpeningHoursPeriod[] | undefined,
): Partial<Record<Weekday, DayHours>> | null {
  if (!periods || periods.length === 0) return null;
  const hours: Partial<Record<Weekday, DayHours>> = {};
  for (const p of periods) {
    if (!p.open) continue;
    const open = formatLegacyTime(p.open.time);
    const close = p.close ? formatLegacyTime(p.close.time) : open;
    hours[GOOGLE_WEEKDAYS[p.open.day]] = { open, close };
  }
  return Object.keys(hours).length > 0 ? hours : null;
}
