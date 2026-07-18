'use client';

// Google Maps-powered map-pin editor for a vendor's business location, with
// an address search box (Places Autocomplete) so an owner can type an
// address and have it geocode straight to a pin, instead of only
// click/drag placement. Only rendered when NEXT_PUBLIC_GOOGLE_MAPS_API_KEY
// is configured — see the conditional dynamic import in app/profile/
// page.tsx, which falls back to the free Leaflet location-picker.tsx
// otherwise. Must only load client-side (touches `window`/injects a
// script tag), same reason the Leaflet picker is dynamic-imported with
// ssr:false.

import { useEffect, useRef, useState } from 'react';

import { loadGoogleMaps } from '../lib/google-maps';

export default function LocationPickerGoogle({
  lat, lng, onChange,
}: {
  lat: number;
  lng: number;
  onChange: (lat: number, lng: number) => void;
}) {
  const mapDivRef = useRef<HTMLDivElement>(null);
  const searchContainerRef = useRef<HTMLDivElement>(null);
  // onChange is re-created every render on the caller's side (an inline
  // arrow function in profile/page.tsx) — capturing it in a ref lets the
  // map-setup effect below run exactly once without needing onChange in
  // its dependency array.
  const onChangeRef = useRef(onChange);
  onChangeRef.current = onChange;
  const [ready, setReady] = useState(false);

  useEffect(() => {
    let cancelled = false;
    loadGoogleMaps().then(async (g) => {
      if (cancelled || !mapDivRef.current) return;

      const map = new g.maps.Map(mapDivRef.current, {
        center: { lat, lng },
        zoom: 15,
        streetViewControl: false,
        mapTypeControl: false,
        fullscreenControl: false,
      });
      const marker = new g.maps.Marker({ position: { lat, lng }, map, draggable: true });

      marker.addListener('dragend', () => {
        const pos = marker.getPosition();
        if (pos) onChangeRef.current(pos.lat(), pos.lng());
      });
      map.addListener('click', (e: google.maps.MapMouseEvent) => {
        if (!e.latLng) return;
        marker.setPosition(e.latLng);
        onChangeRef.current(e.latLng.lat(), e.latLng.lng());
      });

      if (searchContainerRef.current) {
        // Every vendor in this app is Sri Lanka-based today (matches the
        // same assumption the public-holiday feature already makes).
        const { PlaceAutocompleteElement } = await g.maps.importLibrary('places') as google.maps.PlacesLibrary;
        const autocomplete = new PlaceAutocompleteElement({ includedRegionCodes: ['lk'] });
        autocomplete.classList.add('w-full');
        searchContainerRef.current.appendChild(autocomplete);
        autocomplete.addEventListener('gmp-select', async ({ placePrediction }) => {
          const place = placePrediction.toPlace();
          const { place: detailedPlace } = await place.fetchFields({ fields: ['location'] });
          const loc = detailedPlace.location;
          if (!loc) return;
          map.setCenter(loc);
          map.setZoom(16);
          marker.setPosition(loc);
          onChangeRef.current(loc.lat(), loc.lng());
        });
      }

      if (!cancelled) setReady(true);
    });
    return () => {
      cancelled = true;
    };
    // Deliberately empty deps: the map/marker are created once on mount.
    // Subsequent lat/lng prop changes are driven by this component's own
    // onChange calls, not external re-centers, so re-running this on every
    // lat/lng change would fight the user's own pan/zoom/search.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return (
    <div className="relative w-full h-full">
      <div className="absolute top-2 left-2 right-2 z-10">
        <div ref={searchContainerRef} className="[&_gmp-place-autocomplete]:shadow-md [&_gmp-place-autocomplete]:rounded-xl" />
      </div>
      <div ref={mapDivRef} className="w-full h-full" />
      {!ready && (
        <div className="absolute inset-0 flex items-center justify-center bg-sand text-muted font-semibold text-[13px]">
          Loading map…
        </div>
      )}
    </div>
  );
}
