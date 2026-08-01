// Shared Google Maps JS API loader — both components/location-picker-google.tsx
// and app/login/page.tsx's registration "find your business" search need
// the same script loaded, so this caches one load promise at module scope
// rather than each caller invoking its own Loader (the @googlemaps/js-api-loader
// Loader class de-dupes safely on its own, but sharing one promise avoids
// any redundant setup work and keeps both call sites in lockstep on version/
// libraries options).

import { Loader } from '@googlemaps/js-api-loader';

let loaderPromise: Promise<typeof google> | null = null;

export function loadGoogleMaps(): Promise<typeof google> {
  if (!loaderPromise) {
    const loader = new Loader({
      apiKey: process.env.NEXT_PUBLIC_GOOGLE_MAPS_API_KEY ?? '',
      version: 'weekly',
      libraries: ['places'],
    });
    loaderPromise = loader.load().then(() => google);
  }
  return loaderPromise;
}

// A bad/restricted key, a disabled API, or missing billing doesn't reject
// loadGoogleMaps()'s promise — the script tag itself loads fine (200 OK),
// it's the API's own credential check that fails, asynchronously, after
// the map is already being set up. Google's JS API reports that failure
// by calling window.gm_authFailure if it's defined, which is the only
// reliable hook for it (there's no promise rejection or thrown error to
// catch). Consumers subscribe here to fall back to the free Leaflet
// picker instead of leaving Google's own "Oops!" overlay on screen.
type AuthFailureListener = () => void;
const authFailureListeners = new Set<AuthFailureListener>();

function ensureAuthFailureHook() {
  if (typeof window === 'undefined') return;
  const w = window as unknown as { gm_authFailure?: () => void };
  if (w.gm_authFailure) return;
  w.gm_authFailure = () => authFailureListeners.forEach((fn) => fn());
}

export function onGoogleMapsAuthFailure(listener: AuthFailureListener): () => void {
  ensureAuthFailureHook();
  authFailureListeners.add(listener);
  return () => authFailureListeners.delete(listener);
}
