'use client';

// A "Town/area" text input upgraded with Google Places Autocomplete —
// searching an address geocodes it to a precise lat/lng (reported via
// onLocationSelect, for the map below to jump to) while the field itself
// keeps showing/storing just the short town/locality name, not the full
// street address. That keeps this in sync with every other place that
// already assumes a short "Weligama"-style town (admin vendor list/
// search/queue/geo-map chips, the vendor dashboard greeting, account and
// reports pages) — the map pin (lat/lng), not this text, is what actually
// drives precise traveler directions.
//
// Deliberately uses the classic input-bindable google.maps.places.
// Autocomplete widget rather than the newer PlaceAutocompleteElement used
// elsewhere in this app (registration's business search, previously also
// embedded in location-picker-google.tsx) — that element manages its own
// internal shadow-DOM input with no way to pre-fill it with an existing
// saved town, which a plain <input> handles for free.

import { MapPin } from 'lucide-react';
import { useEffect, useRef } from 'react';

import type { DayHours, Weekday } from '@spotto/shared';
import { loadGoogleMaps } from '../lib/google-maps';
import { categoryFromGoogleTypes, hoursFromLegacyPeriods } from '../lib/google-place-mapping';

export interface PlaceDetails {
  placeId?: string;
  phone?: string;
  website?: string;
  category?: string;
  hours?: Partial<Record<Weekday, DayHours>> | null;
}

// Same priority order as townFromAddressComponents in app/login/page.tsx,
// adapted to the classic Autocomplete widget's PlaceResult shape
// (address_components / long_name) rather than the new Place class's
// (addressComponents / longText) — small enough to not be worth sharing.
const TOWN_COMPONENT_PRIORITY = ['locality', 'sublocality', 'administrative_area_level_2', 'administrative_area_level_1'];

function townFromLegacyAddressComponents(components: google.maps.GeocoderAddressComponent[] | undefined): string {
  for (const type of TOWN_COMPONENT_PRIORITY) {
    const match = components?.find((c) => c.types.includes(type));
    if (match) return match.long_name;
  }
  return '';
}

export default function AddressSearchField({
  label = 'Address', value, onChange, onLocationSelect, onPlaceDetails, onBlur, placeholder, autoFocus, variant = 'field',
}: {
  label?: string;
  value: string;
  onChange: (town: string) => void;
  onLocationSelect: (lat: number, lng: number) => void;
  // Fired alongside onLocationSelect/onChange when the matched result is an
  // actual Google Maps business listing (not just a bare address) with
  // phone/website/hours/category data attached — undefined/null fields mean
  // Google didn't have that particular piece of data for this place.
  onPlaceDetails?: (details: PlaceDetails) => void;
  onBlur?: () => void;
  placeholder?: string;
  autoFocus?: boolean;
  // 'field' matches the registration wizard's rounded pill Field component
  // (icon + chunky bordered box); 'plain' matches the profile page's
  // simpler shared `.input` class used by its other business-identity
  // fields (e.g. Business name right next to this one).
  variant?: 'field' | 'plain';
}) {
  const hasKey = !!process.env.NEXT_PUBLIC_GOOGLE_MAPS_API_KEY;
  const inputRef = useRef<HTMLInputElement>(null);
  const onChangeRef = useRef(onChange);
  onChangeRef.current = onChange;
  const onLocationSelectRef = useRef(onLocationSelect);
  onLocationSelectRef.current = onLocationSelect;
  const onPlaceDetailsRef = useRef(onPlaceDetails);
  onPlaceDetailsRef.current = onPlaceDetails;

  useEffect(() => {
    if (!hasKey || !inputRef.current) return;
    let cancelled = false;
    let autocomplete: google.maps.places.Autocomplete | null = null;
    loadGoogleMaps().then(async (g) => {
      if (cancelled || !inputRef.current) return;
      const { Autocomplete } = await g.maps.importLibrary('places') as google.maps.PlacesLibrary;
      autocomplete = new Autocomplete(inputRef.current, {
        fields: [
          'place_id', 'formatted_address', 'geometry', 'address_components',
          'international_phone_number', 'website', 'opening_hours', 'types',
        ],
        componentRestrictions: { country: 'lk' },
      });
      autocomplete.addListener('place_changed', () => {
        const place = autocomplete!.getPlace();
        const loc = place.geometry?.location;
        if (loc) onLocationSelectRef.current(loc.lat(), loc.lng());
        const town = townFromLegacyAddressComponents(place.address_components) || place.formatted_address || '';
        if (town) onChangeRef.current(town);
        onPlaceDetailsRef.current?.({
          placeId: place.place_id,
          phone: place.international_phone_number,
          website: place.website,
          category: categoryFromGoogleTypes(place.types),
          hours: hoursFromLegacyPeriods(place.opening_hours?.periods),
        });
      });
    });
    return () => {
      cancelled = true;
      if (autocomplete) google.maps.event.clearInstanceListeners(autocomplete);
    };
    // hasKey never changes at runtime — this sets up the widget once.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [hasKey]);

  if (variant === 'plain') {
    return (
      <label className="block">
        <span className="block text-[12.5px] font-extrabold mb-1.5 text-muted">{label}</span>
        <input
          ref={inputRef}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          onBlur={onBlur}
          autoFocus={autoFocus}
          placeholder={placeholder ?? 'Search for your address…'}
          className="input"
        />
      </label>
    );
  }

  return (
    <label className="block">
      <span className="block text-[12.5px] font-extrabold mb-1.5 text-ink-soft">{label}</span>
      <div className="flex items-center gap-2.5 rounded-2xl px-3.5 h-[50px] bg-white border-[1.5px] border-line">
        <MapPin size={18} className="text-muted shrink-0" />
        <input
          ref={inputRef}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          onBlur={onBlur}
          autoFocus={autoFocus}
          placeholder={placeholder ?? 'Search for your address…'}
          className="flex-1 bg-transparent outline-none text-[15px] font-bold w-full text-ink"
        />
      </div>
    </label>
  );
}
