'use client';

// Real Leaflet + OpenStreetMap map of Sri Lanka — replaces the old stylised
// SVG silhouette. Shows region demand/supply bubbles (same gap-coloring/
// thresholds as before, now at real coordinates) plus individual vendor
// markers color-coded by status, red when flagged by bad reviews. The
// map's initial view is always this fixed Sri Lanka overview — never the
// browser's own geolocation. Clicking empty map area (not a marker) drops
// an investigation pin so a super-admin can check an arbitrary spot.

import { CircleMarker, MapContainer, Popup, TileLayer, useMapEvents } from 'react-leaflet';

import type { Region, Vendor } from '@spotto/shared';
import { regionTone } from './region-tone';

const SRI_LANKA_CENTER: [number, number] = [7.8731, 80.7718];
const SRI_LANKA_ZOOM = 8;

function vendorColor(v: Vendor): string {
  if (v.flagged) return '#e11d48'; // red — bad-review flag
  if (v.status === 'pending') return '#d97706'; // amber
  if (v.status === 'suspended') return '#7c3aed'; // purple — admin action taken
  if (v.status === 'rejected' || v.status === 'edits') return '#64748b'; // muted
  return '#0b6f86'; // teal — approved & healthy
}

function ClickToInvestigate({ onInvestigate }: { onInvestigate: (lat: number, lng: number) => void }) {
  useMapEvents({
    click(e) {
      onInvestigate(e.latlng.lat, e.latlng.lng);
    },
  });
  return null;
}

export default function VendorMap({
  regions,
  vendors,
  selectedVendorId,
  onSelectVendor,
  investigationPin,
  onInvestigate,
}: {
  regions: Region[];
  vendors: Vendor[];
  selectedVendorId: string | null;
  onSelectVendor: (id: string) => void;
  investigationPin: { lat: number; lng: number } | null;
  onInvestigate: (lat: number, lng: number) => void;
}) {
  return (
    <MapContainer
      center={SRI_LANKA_CENTER}
      zoom={SRI_LANKA_ZOOM}
      style={{ width: '100%', height: '100%' }}
      scrollWheelZoom
    >
      <TileLayer
        attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors &copy; <a href="https://carto.com/attributions">CARTO</a>'
        url="https://{s}.basemaps.cartocdn.com/rastertiles/voyager/{z}/{x}/{y}{r}.png"
        subdomains="abcd"
        maxZoom={20}
      />
      <ClickToInvestigate onInvestigate={onInvestigate} />

      {regions.map((r) => {
        const gap = r.travelers - r.vendors;
        const tone = regionTone(gap);
        const demandR = 8 + (r.travelers / 100) * 20;
        return (
          <CircleMarker
            key={r.id}
            center={[r.lat, r.lng]}
            radius={demandR}
            pathOptions={{ color: tone.ring, fillColor: tone.ring, fillOpacity: 0.15, weight: 1.5, opacity: 0.7 }}
          >
            <Popup>
              <strong>{r.name}</strong>
              <br />
              {tone.label} ({gap > 0 ? '+' : ''}
              {gap})
              <br />
              {r.travelers} travelers · {r.vendors} vendors
            </Popup>
          </CircleMarker>
        );
      })}

      {vendors.map((v) => {
        const color = vendorColor(v);
        const isSelected = v.id === selectedVendorId;
        return (
          <CircleMarker
            key={v.id}
            center={[v.lat, v.lng]}
            radius={isSelected ? 10 : 7}
            pathOptions={{
              color: '#fff',
              weight: 2,
              fillColor: color,
              fillOpacity: 0.95,
            }}
            eventHandlers={{ click: () => onSelectVendor(v.id) }}
          >
            <Popup>
              <strong>{v.name}</strong>
              <br />
              {v.category} · {v.town}
              <br />
              {v.flagged ? `⚠ Flagged — ${v.badReviewCount} bad reviews` : v.status}
            </Popup>
          </CircleMarker>
        );
      })}

      {investigationPin && (
        <CircleMarker
          center={[investigationPin.lat, investigationPin.lng]}
          radius={12}
          pathOptions={{ color: '#103d3a', weight: 2, dashArray: '4 3', fillColor: '#fff', fillOpacity: 0.3 }}
        >
          <Popup>Investigating this area</Popup>
        </CircleMarker>
      )}
    </MapContainer>
  );
}
