'use client';

// Picks between the Google Maps-powered location-picker-google.tsx and the
// free Leaflet/OpenStreetMap location-picker.tsx (identical lat/lng/onChange
// props on both) — and unlike the old build-time-only branch this also
// swaps to Leaflet at *runtime* if Google signals gm_authFailure (bad/
// restricted key, an API not enabled, or billing not set up), instead of
// leaving Google's own "Oops! Something went wrong" overlay stuck on
// screen. See lib/google-maps.ts for how that failure is detected.

import dynamic from 'next/dynamic';
import { useEffect, useState } from 'react';

import { onGoogleMapsAuthFailure } from '../lib/google-maps';

const mapPickerLoading = () => (
  <div className="w-full h-full flex items-center justify-center bg-sand text-muted font-semibold text-[13px]">
    Loading map…
  </div>
);

const GoogleLocationPicker = dynamic(() => import('./location-picker-google'), { ssr: false, loading: mapPickerLoading });
const LeafletLocationPicker = dynamic(() => import('./location-picker'), { ssr: false, loading: mapPickerLoading });

export default function LocationPickerAuto({
  lat, lng, onChange, recenterToken,
}: {
  lat: number;
  lng: number;
  onChange: (lat: number, lng: number) => void;
  // Bump to pan/re-center the map — e.g. after an address search resolves
  // a new location. See location-picker-google.tsx / location-picker.tsx.
  recenterToken?: number;
}) {
  const hasKey = !!process.env.NEXT_PUBLIC_GOOGLE_MAPS_API_KEY;
  const [authFailed, setAuthFailed] = useState(false);

  useEffect(() => {
    if (!hasKey) return;
    return onGoogleMapsAuthFailure(() => setAuthFailed(true));
  }, [hasKey]);

  if (!hasKey || authFailed) {
    return <LeafletLocationPicker lat={lat} lng={lng} onChange={onChange} recenterToken={recenterToken} />;
  }
  return <GoogleLocationPicker lat={lat} lng={lng} onChange={onChange} recenterToken={recenterToken} />;
}
