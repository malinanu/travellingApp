'use client';

// Real Leaflet map (OpenFreeMap base, see base-map-layer.tsx) of Sri Lanka — replaces the old stylised
// SVG silhouette. Shows a circle per town sized by recent traveller
// activity and coloured by activity per live business, plus a pin per vendor in
// its category's colour and icon, the same as the traveller app's map,
// ringed red when flagged by bad reviews. The map's initial view is always
// this fixed Sri Lanka overview — never the browser's own geolocation; it
// flies to a business when the page picks one (`focus`).
// Clicking empty map area (not a marker) drops an investigation pin so a
// super-admin can check an arbitrary spot.

import L from 'leaflet';
import { useEffect } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { CircleMarker, MapContainer, Marker, Popup, useMap, useMapEvents } from 'react-leaflet';

import BaseMapLayer from './base-map-layer';

import type { Region, Vendor, VendorCategory } from '@spotto/shared';
import { pinColor, pinIcon, pinIconName } from './category-pin';
import { regionTone } from './region-tone';

const SRI_LANKA_CENTER: [number, number] = [7.8731, 80.7718];
const SRI_LANKA_ZOOM = 8;
const FOCUS_ZOOM = 16;
const FLAG_RED = '#e11d48';

/** Where the map should fly to; a new object flies again, even to the same spot. */
export type MapFocus = { lat: number; lng: number };

// One DivIcon per look, shared by every pin that has it — hundreds of pins
// use only a few dozen category/flag/selected combinations.
const pinCache = new Map<string, L.DivIcon>();

function categoryPin(category: VendorCategory | undefined, flagged: boolean, selected: boolean): L.DivIcon {
  const color = pinColor(category);
  const key = `${color}|${pinIconName(category)}|${flagged}|${selected}`;
  let pin = pinCache.get(key);
  if (!pin) {
    const Icon = pinIcon(category);
    const size = selected ? 34 : 26;
    const rings = [
      selected ? '0 0 0 3px #103d3a' : '',
      flagged ? `0 0 0 ${selected ? 6 : 3}px ${FLAG_RED}` : '',
      '0 1px 4px rgba(0,0,0,0.35)',
    ].filter(Boolean).join(', ');
    const svg = renderToStaticMarkup(<Icon size={selected ? 17 : 14} color="#fff" strokeWidth={2.4} />);
    pin = L.divIcon({
      className: 'vendor-pin',
      iconSize: [size, size],
      iconAnchor: [size / 2, size / 2],
      popupAnchor: [0, -size / 2],
      html: `<div style="width:${size}px;height:${size}px;border-radius:50%;background:${color};border:2px solid #fff;` +
        `box-shadow:${rings};display:flex;align-items:center;justify-content:center;box-sizing:border-box">${svg}</div>`,
    });
    pinCache.set(key, pin);
  }
  return pin;
}

function FlyTo({ focus }: { focus: MapFocus | null }) {
  const map = useMap();
  useEffect(() => {
    if (focus) map.flyTo([focus.lat, focus.lng], Math.max(map.getZoom(), FOCUS_ZOOM), { duration: 0.8 });
  }, [focus, map]);
  return null;
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
  categories,
  selectedVendorId,
  onSelectVendor,
  investigationPin,
  onInvestigate,
  focus,
}: {
  regions: Region[];
  vendors: Vendor[];
  /** Category id → category, for each pin's colour and icon. */
  categories: Map<string, VendorCategory>;
  selectedVendorId: string | null;
  onSelectVendor: (id: string) => void;
  investigationPin: { lat: number; lng: number } | null;
  onInvestigate: (lat: number, lng: number) => void;
  focus: MapFocus | null;
}) {
  const maxActivity = Math.max(1, ...regions.map((r) => r.travelers));
  return (
    <MapContainer
      center={SRI_LANKA_CENTER}
      zoom={SRI_LANKA_ZOOM}
      style={{ width: '100%', height: '100%' }}
      scrollWheelZoom
    >
      <BaseMapLayer />
      <ClickToInvestigate onInvestigate={onInvestigate} />
      <FlyTo focus={focus} />

      {regions.filter((r) => r.travelers > 0 || r.vendors > 0).map((r) => {
        const tone = regionTone(r.travelers, r.vendors);
        return (
          <CircleMarker
            key={r.id}
            center={[r.lat, r.lng]}
            radius={8 + (20 * r.travelers) / maxActivity}
            pathOptions={{ color: tone.ring, fillColor: tone.ring, fillOpacity: 0.15, weight: 1.5, opacity: 0.7 }}
          >
            <Popup>
              <strong>{r.name}</strong> · {tone.label}
              <br />
              {r.travelers} traveller {r.travelers === 1 ? 'action' : 'actions'} · {r.vendors} live {r.vendors === 1 ? 'business' : 'businesses'}
              <br />
              <span style={{ opacity: 0.7 }}>Check-ins, reviews and saves in the last 30 days</span>
            </Popup>
          </CircleMarker>
        );
      })}

      {vendors.map((v) => {
        const category = categories.get(v.category);
        const isSelected = v.id === selectedVendorId;
        return (
          <Marker
            key={v.id}
            position={[v.lat, v.lng]}
            icon={categoryPin(category, !!v.flagged, isSelected)}
            zIndexOffset={isSelected ? 1000 : v.flagged ? 500 : 0}
            eventHandlers={{ click: () => onSelectVendor(v.id) }}
          >
            <Popup>
              <strong>{v.name}</strong>
              <br />
              {category?.label ?? (v.category || 'Uncategorised')} · {v.town}
              <br />
              {v.flagged ? `⚠ Flagged — ${v.badReviewCount} bad reviews` : v.status}
            </Popup>
          </Marker>
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
