'use client';

// The geo page's map (OpenFreeMap base, see base-map-layer.tsx) of Sri
// Lanka: a heat layer for the chosen analytics layer, or town circles for
// the opportunity view, plus optional business pins in their category's
// colour and icon (ringed red when flagged by bad reviews), each linking to
// the Vendors page. The initial view is always this fixed Sri Lanka
// overview — never the browser's own geolocation. Clicking empty map (not a
// marker) reports the point, for the page's area summary, drawn as a ring.

import L from 'leaflet';
import { useEffect, useRef } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { Circle, CircleMarker, MapContainer, Marker, Popup, useMap, useMapEvents } from 'react-leaflet';

import BaseMapLayer from './base-map-layer';

import type { Vendor, VendorCategory } from '@spotto/shared';
import { pinColor, pinIcon, pinIconName } from './category-pin';

const SRI_LANKA_CENTER: [number, number] = [7.8731, 80.7718];
const SRI_LANKA_ZOOM = 8;
const FLAG_RED = '#e11d48';

export type HeatPoint = { lat: number; lng: number; w: number };

/** A town drawn as a circle (the opportunity view). */
export type TownCircle = {
  id: string;
  name: string;
  lat: number;
  lng: number;
  /** 0..1, scales the circle. */
  size: number;
  color: string;
  label: string;
  detail: string;
};

// One DivIcon per look, shared by every pin that has it.
const pinCache = new Map<string, L.DivIcon>();

function categoryPin(category: VendorCategory | undefined, flagged: boolean): L.DivIcon {
  const color = pinColor(category);
  const key = `${color}|${pinIconName(category)}|${flagged}`;
  let pin = pinCache.get(key);
  if (!pin) {
    const Icon = pinIcon(category);
    const size = 22;
    const rings = [flagged ? `0 0 0 3px ${FLAG_RED}` : '', '0 1px 4px rgba(0,0,0,0.35)'].filter(Boolean).join(', ');
    const svg = renderToStaticMarkup(<Icon size={12} color="#fff" strokeWidth={2.4} />);
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

// leaflet.heat attaches itself to a global `L`, so it's loaded on demand
// once that global is set.
let heatPlugin: Promise<unknown> | null = null;
function loadHeatPlugin() {
  (window as unknown as { L: typeof L }).L = L;
  heatPlugin ??= import('leaflet.heat');
  return heatPlugin;
}

function HeatLayer({ points, gradient }: { points: HeatPoint[]; gradient: Record<number, string> }) {
  const map = useMap();
  const layer = useRef<L.HeatLayer | null>(null);
  useEffect(() => {
    let cancelled = false;
    loadHeatPlugin().then(() => {
      if (cancelled) return;
      // A square root keeps quiet areas visible next to a very busy one.
      const max = Math.max(1, ...points.map((p) => p.w));
      const data = points.map((p): L.HeatLatLngTuple => [p.lat, p.lng, Math.sqrt(p.w / max)]);
      if (!layer.current) {
        layer.current = L.heatLayer(data, { radius: 26, blur: 20, minOpacity: 0.3, maxZoom: 13, max: 1, gradient }).addTo(map);
      } else {
        layer.current.setOptions({ gradient });
        layer.current.setLatLngs(data);
      }
    });
    return () => {
      cancelled = true;
    };
  }, [points, gradient, map]);
  useEffect(() => () => {
    layer.current?.remove();
    layer.current = null;
  }, [map]);
  return null;
}

function ClickToInspect({ onClick }: { onClick: (lat: number, lng: number) => void }) {
  useMapEvents({
    click(e) {
      onClick(e.latlng.lat, e.latlng.lng);
    },
  });
  return null;
}

export default function GeoMap({
  heat,
  gradient,
  towns,
  vendors,
  categories,
  area,
  onMapClick,
}: {
  /** Heat points, or null to draw none. */
  heat: HeatPoint[] | null;
  gradient: Record<number, string>;
  towns: TownCircle[] | null;
  /** Business pins to overlay, or null for none. */
  vendors: Vendor[] | null;
  /** Category id → category, for each pin's colour and icon. */
  categories: Map<string, VendorCategory>;
  area: { lat: number; lng: number; radiusKm: number } | null;
  onMapClick: (lat: number, lng: number) => void;
}) {
  return (
    <MapContainer center={SRI_LANKA_CENTER} zoom={SRI_LANKA_ZOOM} style={{ width: '100%', height: '100%' }} scrollWheelZoom>
      <BaseMapLayer />
      <ClickToInspect onClick={onMapClick} />
      {heat && <HeatLayer points={heat} gradient={gradient} />}

      {towns?.map((t) => (
        <CircleMarker key={t.id} center={[t.lat, t.lng]} radius={8 + 22 * t.size}
          pathOptions={{ color: t.color, fillColor: t.color, fillOpacity: 0.2, weight: 1.5, opacity: 0.8 }}>
          <Popup>
            <strong>{t.name}</strong> · {t.label}
            <br />
            {t.detail}
          </Popup>
        </CircleMarker>
      ))}

      {vendors?.map((v) => {
        const category = categories.get(v.category);
        return (
          <Marker key={v.id} position={[v.lat, v.lng]} icon={categoryPin(category, !!v.flagged)} zIndexOffset={v.flagged ? 500 : 0}>
            <Popup>
              <strong>{v.name}</strong>
              <br />
              {category?.label ?? (v.category || 'Uncategorised')} · {v.town}
              <br />
              {v.flagged ? `⚠ Flagged — ${v.badReviewCount} bad reviews` : v.status}
              <br />
              <a href={`/vendors?id=${encodeURIComponent(v.id)}`}>Open in Vendors →</a>
            </Popup>
          </Marker>
        );
      })}

      {area && (
        <Circle center={[area.lat, area.lng]} radius={area.radiusKm * 1000}
          pathOptions={{ color: '#103d3a', weight: 2, dashArray: '6 5', fillColor: '#103d3a', fillOpacity: 0.06 }} />
      )}
    </MapContainer>
  );
}
