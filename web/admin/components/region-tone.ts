// Pure gap-coloring helper, split out from vendor-map.tsx so pages that only
// need this (e.g. geo/page.tsx's leaderboard) don't statically import a
// module that touches `window` at load time via react-leaflet — Next 16's
// Turbopack evaluates static imports eagerly during prerendering, unlike
// the previous webpack build, so that eager evaluation used to crash the
// build with "window is not defined" even though the map itself was
// correctly lazy-loaded via next/dynamic.

export function regionTone(gap: number) {
  if (gap >= 28) return { ring: '#e11d48', label: 'Critical gap' };
  if (gap >= 12) return { ring: '#d97706', label: 'Under-served' };
  if (gap <= -10) return { ring: '#0b6f86', label: 'Saturated' };
  return { ring: '#64748b', label: 'Balanced' };
}
