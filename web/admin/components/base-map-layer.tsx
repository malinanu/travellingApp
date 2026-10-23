'use client';

// Base map for the Leaflet maps: OpenFreeMap's free vector tiles
// (OpenStreetMap data, no API key, no usage limits) drawn through
// MapLibre GL inside Leaflet, so markers, clicks and popups stay plain
// react-leaflet. OSM's own tile servers block app traffic and CARTO's
// basemaps now need an API key, which is why it isn't a raster TileLayer.
// Client-only (MapLibre needs WebGL and `window`), like every map here.

import 'maplibre-gl/dist/maplibre-gl.css';

import { maplibreGL } from '@maplibre/maplibre-gl-leaflet';
import { useEffect } from 'react';
import { useMap } from 'react-leaflet';

const STYLE_URL = 'https://tiles.openfreemap.org/styles/liberty';
const ATTRIBUTION =
  '<a href="https://openfreemap.org" target="_blank" rel="noreferrer">OpenFreeMap</a> ' +
  '<a href="https://www.openmaptiles.org/" target="_blank" rel="noreferrer">&copy; OpenMapTiles</a> ' +
  'Data from <a href="https://www.openstreetmap.org/copyright" target="_blank" rel="noreferrer">OpenStreetMap</a>';

export default function BaseMapLayer() {
  const map = useMap();
  useEffect(() => {
    const layer = maplibreGL({ style: STYLE_URL, attributionControl: false });
    // The GL layer has no Leaflet attribution of its own; credit the
    // tiles and data in Leaflet's attribution control instead.
    layer.getAttribution = () => ATTRIBUTION;
    layer.addTo(map);
    return () => {
      layer.remove();
    };
  }, [map]);
  return null;
}
