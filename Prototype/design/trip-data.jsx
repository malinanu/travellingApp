// trip-data.jsx — Trip planner catalog + seed trip + distance helpers.
// Catalog = Spotto curated places (with real-ish coords) + more Weligama spots
// so search can find hotels, beaches, bars etc. Exports to window.

// ── searchable place catalog ───────────────────────────────────
// styles: which travel styles this suits (for suggestions)
const TRIP_CATALOG = [
  // Spotto curated (linked to PLACES so tapping opens the business drawer)
  { id: 'driftwood',  spottoId: 'driftwood',  name: 'Driftwood Coffee & Co-Work', type: 'Coworking Café', icon: 'Coffee',   lat: 5.9762, lng: 80.4221, cost: 1200, styles: ['nomad'],            hint: '8:00' },
  { id: 'baysurf',    spottoId: 'baysurf',    name: 'Weligama Bay Surf School',   type: 'Surf School',    icon: 'Waves',    lat: 5.9716, lng: 80.4308, cost: 3500, styles: ['surf','backpacker'], hint: '6:30' },
  { id: 'hangtime',   spottoId: 'hangtime',   name: 'Hangtime Rooftop & Sunset Bar', type: 'Sunset Bar',  icon: 'Martini',  lat: 5.9789, lng: 80.4275, cost: 2800, styles: ['nomad','luxury'],    hint: '17:30' },
  { id: 'claystudio', spottoId: 'claystudio', name: 'Clay & Co Pottery Studio',   type: 'Pottery Workshop', icon: 'Palette', lat: 5.9773, lng: 80.4189, cost: 1500, styles: ['backpacker'],       hint: '10:00' },
  { id: 'batik',      spottoId: 'batik',      name: 'Sea Salt Batik Atelier',     type: 'Batik Atelier',  icon: 'Brush',    lat: 5.9747, lng: 80.4368, cost: 1000, styles: ['luxury'],            hint: '11:00' },
  // wider Weligama area
  { id: 'bayvilla',   name: 'Bay Villa Boutique Stay',   type: 'Hotel',          icon: 'BedDouble', lat: 5.9731, lng: 80.4254, cost: 9500, styles: ['luxury','nomad'],    hint: '14:00' },
  { id: 'palmhostel', name: 'Palm & Board Hostel',       type: 'Hostel',         icon: 'Backpack',  lat: 5.9758, lng: 80.4332, cost: 2200, styles: ['backpacker','surf'], hint: '13:00' },
  { id: 'welibeach',  name: 'Weligama Beach Point',      type: 'Beach',          icon: 'Palmtree',  lat: 5.9722, lng: 80.4286, cost: 0,    styles: ['surf','backpacker','nomad','luxury'], hint: '7:00' },
  { id: 'mirissa',    name: 'Mirissa Beach',             type: 'Beach',          icon: 'Palmtree',  lat: 5.9445, lng: 80.4595, cost: 0,    styles: ['surf','backpacker','luxury'], hint: '15:00' },
  { id: 'coconuthill',name: 'Coconut Tree Hill',         type: 'Viewpoint',      icon: 'Mountain',  lat: 5.9417, lng: 80.4632, cost: 0,    styles: ['backpacker','luxury','nomad'], hint: '17:00' },
  { id: 'kottu',      name: 'Smoky Kottu Shack',         type: 'Street Food',    icon: 'UtensilsCrossed', lat: 5.9755, lng: 80.4312, cost: 800, styles: ['backpacker','surf'], hint: '19:30' },
  { id: 'reefdine',   name: 'Reef & Vine Dining',        type: 'Restaurant',     icon: 'Wine',      lat: 5.9740, lng: 80.4260, cost: 5200, styles: ['luxury'],            hint: '19:00' },
  { id: 'secretroot', name: 'Secret Root Spa',           type: 'Spa & Wellness', icon: 'Flower2',   lat: 5.9768, lng: 80.4338, cost: 4200, styles: ['luxury','nomad'],    hint: '16:00' },
  { id: 'taprobane',  name: 'Taprobane Island Viewpoint',type: 'Viewpoint',      icon: 'Mountain',  lat: 5.9735, lng: 80.4203, cost: 0,    styles: ['nomad','luxury','backpacker'], hint: '8:30' },
  { id: 'oldlight',   name: 'Old Light Gallery Bar',     type: 'Bar & Gallery',  icon: 'Martini',   lat: 5.9701, lng: 80.4179, cost: 2400, styles: ['nomad','backpacker'], hint: '21:00' },
  { id: 'morningmkt', name: 'Weligama Morning Market',   type: 'Local Market',   icon: 'ShoppingBasket', lat: 5.9769, lng: 80.4297, cost: 500, styles: ['backpacker','nomad'], hint: '6:30' },
];

// ── seed trip (Weligama weekend, mirrors hotel → café → bar → beach) ──
// startDate drives the calendar; collaborators + `by` demo shared planning.
const TRIP_SEED = {
  name: 'Weligama Weekend',
  code: 'WLG-7QF',
  startDate: '2026-06-20',
  collaborators: ['theo', 'mia'],
  days: [
    { id: 'd1', stops: [
      { id: 's1', placeId: 'bayvilla',  time: 14 * 60 },                 // 2:00 PM check-in
      { id: 's2', placeId: 'driftwood', time: 16 * 60, by: 'theo' },     // 4:00 PM café (added by Theo)
      { id: 's3', placeId: 'hangtime',  time: 17 * 60 + 30 },            // 5:30 PM sunset bar
      { id: 's4', placeId: 'kottu',     time: 19 * 60 + 30 },            // 7:30 PM dinner
    ]},
    { id: 'd2', stops: [
      { id: 's5', placeId: 'welibeach', time: 7 * 60 },                  // 7:00 AM beach
      { id: 's6', placeId: 'baysurf',   time: 8 * 60, by: 'mia' },       // 8:00 AM surf (added by Mia)
      { id: 's7', placeId: 'coconuthill', time: 17 * 60 },               // 5:00 PM viewpoint
    ]},
  ],
};

// ── helpers ────────────────────────────────────────────────────
function tripHaversineKm(a, b) {
  const R = 6371, rad = (d) => (d * Math.PI) / 180;
  const dLat = rad(b.lat - a.lat), dLng = rad(b.lng - a.lng);
  const h = Math.sin(dLat / 2) ** 2 + Math.cos(rad(a.lat)) * Math.cos(rad(b.lat)) * Math.sin(dLng / 2) ** 2;
  return 2 * R * Math.asin(Math.sqrt(h));
}

// leg between two catalog places: walk if short, tuk-tuk otherwise
function tripLeg(from, to) {
  const km = tripHaversineKm(from, to);
  const walk = km <= 1.2;
  const mins = Math.max(1, Math.round(walk ? km * 14 : km * 3 + 4));
  const dist = km < 1 ? `${Math.round(km * 1000)} m` : `${km.toFixed(1)} km`;
  return {
    km, dist, mins,
    mode: walk ? 'Walk' : 'Tuk-tuk',
    icon: walk ? 'Footprints' : 'CarTaxiFront',
    gmaps: `https://www.google.com/maps/dir/?api=1&origin=${from.lat},${from.lng}&destination=${to.lat},${to.lng}&travelmode=${walk ? 'walking' : 'driving'}`,
  };
}

// whole-day route in Google Maps (origin → waypoints → destination)
function tripDayGmaps(stops) {
  const pts = stops.map((s) => tripPlace(s)).filter(Boolean);
  if (pts.length < 2) return null;
  const o = pts[0], d = pts[pts.length - 1];
  const way = pts.slice(1, -1).map((p) => `${p.lat},${p.lng}`).join('|');
  return `https://www.google.com/maps/dir/?api=1&origin=${o.lat},${o.lng}&destination=${d.lat},${d.lng}${way ? `&waypoints=${encodeURIComponent(way)}` : ''}&travelmode=driving`;
}

// resolve a stop → catalog place (custom stops carry their own data)
function tripPlace(stop) {
  if (stop.custom) return stop.custom;
  return TRIP_CATALOG.find((p) => p.id === stop.placeId) || null;
}

function tripFmtTime(mins) {
  let h = Math.floor(mins / 60), m = mins % 60;
  const ap = h >= 12 ? 'PM' : 'AM';
  h = h % 12 || 12;
  return `${h}:${String(m).padStart(2, '0')} ${ap}`;
}

function tripFmtRs(n) {
  if (!n) return 'Free';
  return `Rs ${n.toLocaleString('en-US')}`;
}

// date for day index i, derived from the trip's startDate
function tripDayDate(startDate, i) {
  const base = (startDate || new Date().toISOString().slice(0, 10)) + 'T00:00:00';
  const d = new Date(base);
  d.setDate(d.getDate() + i);
  return d;
}

Object.assign(window, { TRIP_CATALOG, TRIP_SEED, tripLeg, tripDayGmaps, tripPlace, tripFmtTime, tripFmtRs, tripHaversineKm, tripDayDate });
