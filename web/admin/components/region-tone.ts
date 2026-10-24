// Pure helper for the geo page's town circles and Opportunity leaderboard,
// split out from vendor-map.tsx so pages that only need this (e.g.
// geo/page.tsx's leaderboard) don't statically import a module that touches
// `window` at load time via react-leaflet — Next 16's Turbopack evaluates
// static imports eagerly during prerendering, unlike the previous webpack
// build, so that eager evaluation used to crash the build with "window is
// not defined" even though the map itself was correctly lazy-loaded via
// next/dynamic.
//
// activity is traveller actions (check-ins, reviews, saves) in a region in
// the last 30 days and businesses its live businesses (GET /admin/regions),
// so the score is demand per business.

export function regionScore(activity: number, businesses: number): number {
  return activity / Math.max(1, businesses);
}

export function regionTone(activity: number, businesses: number) {
  const perBusiness = regionScore(activity, businesses);
  if (activity === 0) return { ring: '#94a3b8', label: 'No activity yet' };
  if (perBusiness >= 10) return { ring: '#e11d48', label: 'Critical gap' };
  if (perBusiness >= 5) return { ring: '#d97706', label: 'Under-served' };
  if (businesses >= 5 && perBusiness < 0.5) return { ring: '#0b6f86', label: 'Saturated' };
  return { ring: '#64748b', label: 'Balanced' };
}
