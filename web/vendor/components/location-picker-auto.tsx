'use client';

// The map-pin picker used by registration and the Business Profile: the
// free Leaflet map in location-picker.tsx (CARTO tiles, no API key),
// loaded client-side only because Leaflet touches `window` at import
// time. Address search stays on Google Places (address-search-field.tsx)
// for its business auto-fill, but the map itself never depends on Google
// loading, so a blocked or failing Maps script can't leave it stuck on
// "Loading map…".

import dynamic from 'next/dynamic';

// One loader shared by the component and preloadLocationPicker, so both
// fetch the same chunk files (two separate import() calls get bundled with
// different chunk lists).
const loadLocationPicker = () => import('./location-picker');

const LocationPicker = dynamic(loadLocationPicker, {
  ssr: false,
  loading: () => (
    <div className="w-full h-full flex items-center justify-center bg-sand text-muted font-semibold text-[13px]">
      Loading map…
    </div>
  ),
});

// Fetches the map code ahead of time, so a page opened before a deploy still
// has it when the map finally renders (the deploy removes the old file —
// see app/error.tsx). Failures are left for the real render to surface.
export function preloadLocationPicker() {
  loadLocationPicker().catch(() => {});
}

export default function LocationPickerAuto(props: {
  lat: number;
  lng: number;
  onChange: (lat: number, lng: number) => void;
  // Bump to pan/re-center the map — e.g. after an address search resolves
  // a new location. See location-picker.tsx.
  recenterToken?: number;
}) {
  return <LocationPicker {...props} />;
}
