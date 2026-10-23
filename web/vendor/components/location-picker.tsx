'use client';

// Draggable map-pin editor for correcting a vendor's business/map location —
// reuses the same free CARTO Voyager tiles (OpenStreetMap data) and
// CircleMarker pattern as web/admin/components/vendor-map.tsx (no API key,
// no Leaflet default-icon-asset headaches). Must only be
// loaded client-side (Leaflet touches `window` at import time) — see the
// dynamic(..., { ssr: false }) import in app/profile/page.tsx.

import { useEffect, useRef } from 'react';
import type { Map as LeafletMap } from 'leaflet';
import { CircleMarker, MapContainer, TileLayer, useMapEvents } from 'react-leaflet';

function ClickToMove({ onMove }: { onMove: (lat: number, lng: number) => void }) {
  useMapEvents({
    click(e) {
      onMove(e.latlng.lat, e.latlng.lng);
    },
  });
  return null;
}

export default function LocationPicker({
  lat, lng, onChange, recenterToken,
}: {
  lat: number;
  lng: number;
  onChange: (lat: number, lng: number) => void;
  // Bump this (e.g. from an address-search selection) to pan the map to
  // the current lat/lng props — MapContainer's own `center` prop only
  // sets the *initial* view in react-leaflet, it doesn't recenter on
  // change, so an external jump needs this explicit signal instead.
  recenterToken?: number;
}) {
  // React Strict Mode's dev-only double-mount doesn't clear Leaflet's
  // internal `_leaflet_id` binding on the container div between the two
  // synthetic mounts, which makes the second one throw "Map container is
  // already initialized." Explicitly removing the map instance on unmount
  // (react-leaflet v4 forwards MapContainer's ref to the Leaflet Map) frees
  // that binding so the remount succeeds.
  const mapRef = useRef<LeafletMap | null>(null);
  useEffect(() => () => {
    mapRef.current?.remove();
    mapRef.current = null;
  }, []);

  const latRef = useRef(lat);
  latRef.current = lat;
  const lngRef = useRef(lng);
  lngRef.current = lng;
  useEffect(() => {
    if (recenterToken == null || !mapRef.current) return;
    mapRef.current.setView([latRef.current, lngRef.current], 16);
    // Reads lat/lng fresh via refs when recenterToken changes.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [recenterToken]);

  // CircleMarker (unlike Marker) has no native dragging in Leaflet, so the
  // pin is repositioned by clicking anywhere on the map — same interaction
  // vendor-map.tsx already uses for its investigation pin.
  return (
    <MapContainer ref={mapRef} center={[lat, lng]} zoom={15} style={{ width: '100%', height: '100%' }} scrollWheelZoom>
      <TileLayer
        attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors &copy; <a href="https://carto.com/attributions">CARTO</a>'
        url="https://{s}.basemaps.cartocdn.com/rastertiles/voyager/{z}/{x}/{y}{r}.png"
        subdomains="abcd"
        maxZoom={20}
      />
      <ClickToMove onMove={onChange} />
      <CircleMarker
        center={[lat, lng]}
        radius={10}
        pathOptions={{ color: '#fff', weight: 2, fillColor: '#0b6f86', fillOpacity: 0.95 }}
      />
    </MapContainer>
  );
}
