'use client';

// Google Maps-powered map-pin editor for a vendor's business location —
// click or drag to move the pin. Address search now lives outside this
// component (see components/address-search-field.tsx, used by callers
// alongside this picker) rather than as a floating box on top of the map,
// so there's a single search entry point instead of two. Only rendered
// when NEXT_PUBLIC_GOOGLE_MAPS_API_KEY is configured — see
// components/location-picker-auto.tsx, which falls back to the free
// Leaflet location-picker.tsx otherwise. Must only load client-side
// (touches `window`/injects a script tag), same reason that picker is
// dynamic-imported with ssr:false.

import { useEffect, useRef, useState } from 'react';

import { loadGoogleMaps } from '../lib/google-maps';

export default function LocationPickerGoogle({
  lat, lng, onChange, recenterToken,
}: {
  lat: number;
  lng: number;
  onChange: (lat: number, lng: number) => void;
  // Bump this (e.g. from an address-search selection) to pan/zoom the map
  // to the current lat/lng props. Plain lat/lng prop changes are otherwise
  // ignored post-mount — see the mount effect below — so a search result
  // needs this explicit signal to actually move the map.
  recenterToken?: number;
}) {
  const mapDivRef = useRef<HTMLDivElement>(null);
  // onChange is re-created every render on the caller's side (an inline
  // arrow function in profile/page.tsx) — capturing it in a ref lets the
  // map-setup effect below run exactly once without needing onChange in
  // its dependency array.
  const onChangeRef = useRef(onChange);
  onChangeRef.current = onChange;
  const latRef = useRef(lat);
  latRef.current = lat;
  const lngRef = useRef(lng);
  lngRef.current = lng;
  const mapRef = useRef<google.maps.Map | null>(null);
  const markerRef = useRef<google.maps.Marker | null>(null);
  const [ready, setReady] = useState(false);

  useEffect(() => {
    let cancelled = false;
    loadGoogleMaps().then((g) => {
      if (cancelled || !mapDivRef.current) return;

      const map = new g.maps.Map(mapDivRef.current, {
        center: { lat, lng },
        zoom: 15,
        streetViewControl: false,
        mapTypeControl: false,
        fullscreenControl: false,
      });
      const marker = new g.maps.Marker({ position: { lat, lng }, map, draggable: true });
      mapRef.current = map;
      markerRef.current = marker;

      marker.addListener('dragend', () => {
        const pos = marker.getPosition();
        if (pos) onChangeRef.current(pos.lat(), pos.lng());
      });
      map.addListener('click', (e: google.maps.MapMouseEvent) => {
        if (!e.latLng) return;
        marker.setPosition(e.latLng);
        onChangeRef.current(e.latLng.lat(), e.latLng.lng());
      });

      setReady(true);
    });
    return () => {
      cancelled = true;
    };
    // Deliberately empty deps: the map/marker are created once on mount.
    // Subsequent lat/lng prop changes are driven by this component's own
    // onChange calls, not external re-centers, so re-running this on every
    // lat/lng change would fight the user's own pan/zoom — recenterToken
    // (below) is the explicit escape hatch for a genuine external jump.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    if (recenterToken == null || !mapRef.current || !markerRef.current) return;
    const pos = { lat: latRef.current, lng: lngRef.current };
    mapRef.current.panTo(pos);
    mapRef.current.setZoom(16);
    markerRef.current.setPosition(pos);
    // Reads lat/lng fresh via refs when recenterToken changes, rather than
    // depending on lat/lng directly — see the mount effect's comment.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [recenterToken]);

  return (
    <div className="relative w-full h-full">
      <div ref={mapDivRef} className="w-full h-full" />
      {!ready && (
        <div className="absolute inset-0 flex items-center justify-center bg-sand text-muted font-semibold text-[13px]">
          Loading map…
        </div>
      )}
    </div>
  );
}
