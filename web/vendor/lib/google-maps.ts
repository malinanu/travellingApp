// Shared Google Maps JS API loader — both components/address-search-field.tsx
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
    // A failed load (blocked script, offline) shouldn't stick: the next
    // caller gets a fresh attempt instead of the same rejected promise.
    loaderPromise.catch(() => { loaderPromise = null; });
  }
  return loaderPromise;
}
