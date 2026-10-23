'use client';

// The map-pin picker used by registration and the Business Profile: the
// free Leaflet map in location-picker.tsx (CARTO tiles, no API key),
// loaded client-side only because Leaflet touches `window` at import
// time. Address search stays on Google Places (address-search-field.tsx)
// for its business auto-fill, but the map itself never depends on Google
// loading, so a blocked or failing Maps script can't leave it stuck on
// "Loading map…".

import dynamic from 'next/dynamic';

const LocationPicker = dynamic(() => import('./location-picker'), {
  ssr: false,
  loading: () => (
    <div className="w-full h-full flex items-center justify-center bg-sand text-muted font-semibold text-[13px]">
      Loading map…
    </div>
  ),
});

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
