// admin-data.jsx — Spotto Admin Command Center demo data.

// ── Vendor approval queue ──
const VENDOR_QUEUE = [
  { id: 'V-2041', name: 'Ahangama Yoga Studio', cat: 'Yoga / Wellness', town: 'Ahangama', owner: 'Nimali Fernando',
    submitted: '12 min ago', priority: 'high', docs: true, photos: 6, wifi: 7.4,
    flags: [], note: 'Sunrise rooftop shala, 200hr certified instructors. Requesting Premium.',
    region: 'south' },
  { id: 'V-2040', name: 'Hiriketiya Poke Bar', cat: 'Restaurant', town: 'Dickwella', owner: 'Sea & Co Ltd',
    submitted: '38 min ago', priority: 'normal', docs: true, photos: 4, wifi: 6.1,
    flags: ['Duplicate address?'], note: 'Beachfront poke & smoothies. Shares plot with registered surf shop.',
    region: 'south' },
  { id: 'V-2039', name: 'Ella Rock Trail Guides', cat: 'Tours / Activity', town: 'Ella',
    owner: 'Kasun Bandara', submitted: '1 hr ago', priority: 'normal', docs: false, photos: 2, wifi: null,
    flags: ['Missing business license'], note: 'Guided sunrise hikes. License upload pending.', region: 'hill' },
  { id: 'V-2038', name: 'Fort Print & Coffee', cat: 'Coworking Café', town: 'Galle Fort',
    owner: 'Dilan Jayawardene', submitted: '2 hr ago', priority: 'high', docs: true, photos: 9, wifi: 9.0,
    flags: [], note: 'Heritage-building café with fibre internet. Strong nomad fit.', region: 'south' },
  { id: 'V-2037', name: 'Arugam Board Co.', cat: 'Surf School', town: 'Arugam Bay',
    owner: 'Wave Theory', submitted: '3 hr ago', priority: 'normal', docs: true, photos: 5, wifi: 5.2,
    flags: [], note: 'Board rental + lessons on the east coast point break.', region: 'east' },
  { id: 'V-2036', name: 'Kandy Spice Kitchen', cat: 'Restaurant', town: 'Kandy',
    owner: 'Heritage Foods', submitted: '5 hr ago', priority: 'low', docs: true, photos: 3, wifi: 6.8,
    flags: ['Low photo quality'], note: 'Family curry house near the temple.', region: 'hill' },
];

// ── Live content feed (seed + generated) ──
const FEED_SEED = [
  { id: 'C901', vendor: 'Driftwood Coffee', town: 'Weligama', type: 'Offer', text: 'Free flat white with any day pass', status: 'ok',     t: 'now' },
  { id: 'C900', vendor: 'Hangtime Rooftop', town: 'Weligama', type: 'Event', text: 'Full-moon DJ set · Saturday 9PM', status: 'ok',     t: '1m' },
  { id: 'C899', vendor: 'Lucky Tuk Tours',  town: 'Galle',    type: 'Offer', text: 'CHEAPEST TOURS!!! 90% OFF CALL NOW', status: 'flag',  t: '2m', reason: 'Spammy formatting · unverifiable discount' },
  { id: 'C898', vendor: 'Mirissa Whale Co', town: 'Mirissa',  type: 'Offer', text: 'Dawn whale-watching · 15% off groups', status: 'ok',  t: '3m' },
  { id: 'C897', vendor: 'Ella Gin Bar',     town: 'Ella',     type: 'Event', text: 'Live acoustic night · free entry', status: 'review', t: '4m', reason: 'New vendor · first post' },
];

const FEED_TEMPLATES = [
  { vendor: 'Unawatuna Dive', town: 'Unawatuna', type: 'Offer', text: 'Discover-scuba intro · 20% off', status: 'ok' },
  { vendor: 'Hiriketiya Surf', town: 'Dickwella', type: 'Offer', text: 'Beginner group lesson deal', status: 'ok' },
  { vendor: 'Colombo Roasters', town: 'Colombo', type: 'Event', text: 'Latte art throwdown · Friday', status: 'ok' },
  { vendor: 'Best Deals Lanka', town: 'Hikkaduwa', type: 'Offer', text: 'FREE MONEY click here to win $$$', status: 'flag', reason: 'Phishing pattern detected' },
  { vendor: 'Sigiriya Sunrise', town: 'Sigiriya', type: 'Event', text: 'Guided rock climb at dawn', status: 'review', reason: 'Unverified safety cert' },
  { vendor: 'Jaffna Palmyra', town: 'Jaffna', type: 'Offer', text: 'Authentic crab curry · lunch set', status: 'ok' },
];

// ── Geo regions: travelers vs vendors (gap = demand minus supply) ──
const REGIONS = [
  { id: 'colombo',  name: 'Colombo',     x: 25, y: 56, travelers: 88, vendors: 96 },
  { id: 'negombo',  name: 'Negombo',     x: 23, y: 49, travelers: 54, vendors: 60 },
  { id: 'kandy',    name: 'Kandy',       x: 50, y: 47, travelers: 71, vendors: 64 },
  { id: 'sigiriya', name: 'Sigiriya',    x: 54, y: 34, travelers: 66, vendors: 28 },
  { id: 'ella',     name: 'Ella',        x: 61, y: 60, travelers: 82, vendors: 41 },
  { id: 'nuwara',   name: 'Nuwara Eliya',x: 54, y: 56, travelers: 58, vendors: 39 },
  { id: 'galle',    name: 'Galle',       x: 31, y: 82, travelers: 90, vendors: 78 },
  { id: 'weligama', name: 'Weligama',    x: 39, y: 86, travelers: 95, vendors: 70 },
  { id: 'mirissa',  name: 'Mirissa',     x: 44, y: 88, travelers: 84, vendors: 52 },
  { id: 'arugam',   name: 'Arugam Bay',  x: 80, y: 61, travelers: 76, vendors: 22 },
  { id: 'trinco',   name: 'Trincomalee', x: 70, y: 27, travelers: 61, vendors: 18 },
  { id: 'jaffna',   name: 'Jaffna',      x: 45, y: 8,  travelers: 44, vendors: 12 },
];

// ── Subscriptions ──
const SUBS = [
  { id: 'V-1001', vendor: 'Driftwood Coffee & Co-Work', town: 'Weligama', tier: 'Premium', mrr: 49, renews: 'in 4 days',  state: 'soon',   since: 'Jan 2024' },
  { id: 'V-1002', vendor: 'Hangtime Rooftop & Bar',     town: 'Weligama', tier: 'Premium', mrr: 49, renews: 'in 21 days', state: 'active', since: 'Mar 2024' },
  { id: 'V-1003', vendor: 'Weligama Bay Surf School',   town: 'Weligama', tier: 'Premium', mrr: 49, renews: 'in 12 days', state: 'active', since: 'Feb 2024' },
  { id: 'V-1004', vendor: 'Mirissa Whale Co.',          town: 'Mirissa',  tier: 'Free',    mrr: 0,  renews: '—',          state: 'trial',  since: 'Aug 2024' },
  { id: 'V-1005', vendor: 'Ella Gin Bar',               town: 'Ella',     tier: 'Free',    mrr: 0,  renews: '—',          state: 'free',   since: 'Sep 2024' },
  { id: 'V-1006', vendor: 'Fort Print & Coffee',        town: 'Galle',    tier: 'Premium', mrr: 49, renews: 'in 2 days',  state: 'soon',   since: 'Apr 2024' },
  { id: 'V-1007', vendor: 'Unawatuna Dive Center',      town: 'Unawatuna',tier: 'Premium', mrr: 49, renews: 'overdue',    state: 'overdue',since: 'Nov 2023' },
  { id: 'V-1008', vendor: 'Kandy Spice Kitchen',        town: 'Kandy',    tier: 'Free',    mrr: 0,  renews: '—',          state: 'free',   since: 'Jul 2024' },
  { id: 'V-1009', vendor: 'Arugam Board Co.',           town: 'Arugam Bay',tier: 'Free',   mrr: 0,  renews: '—',          state: 'trial',  since: 'Oct 2024' },
  { id: 'V-1010', vendor: 'Colombo Roasters',           town: 'Colombo',  tier: 'Premium', mrr: 49, renews: 'in 30 days', state: 'active', since: 'Dec 2023' },
];

// ── Audio guides CMS ──
const AUDIO_GUIDES = [
  { id: 'AG-01', site: 'Galle Fort Ramparts', region: 'Galle', lat: 6.0269, lng: 80.2168,
    depths: ['Quick', 'Standard', 'In-depth'], lang: 'EN', published: true, plays: 3120, duration: { Quick: '4m', Standard: '11m', 'In-depth': '28m' } },
  { id: 'AG-02', site: 'Sigiriya Rock Fortress', region: 'Sigiriya', lat: 7.9570, lng: 80.7603,
    depths: ['Quick', 'Standard'], lang: 'EN', published: true, plays: 2410, duration: { Quick: '5m', Standard: '14m' } },
  { id: 'AG-03', site: 'Temple of the Tooth', region: 'Kandy', lat: 7.2936, lng: 80.6413,
    depths: ['Standard', 'In-depth'], lang: 'EN', published: false, plays: 0, duration: { Standard: '12m', 'In-depth': '31m' } },
  { id: 'AG-04', site: 'Dutch Hospital Precinct', region: 'Colombo', lat: 6.9330, lng: 79.8430,
    depths: ['Quick'], lang: 'EN', published: true, plays: 880, duration: { Quick: '3m' } },
];

const DEPTH_INFO = {
  Quick:     { mins: '3–5 min',  desc: 'Highlights only — for travelers on the move.', icon: 'Zap' },
  Standard:  { mins: '10–15 min',desc: 'The essential story and key landmarks.',       icon: 'Headphones' },
  'In-depth':{ mins: '25–35 min',desc: 'Full history, characters and hidden details.', icon: 'BookOpen' },
};

// ── KPIs for header ──
const ADMIN_KPIS = [
  { id: 'pending', label: 'Pending approvals', value: VENDOR_QUEUE.length, icon: 'UserCheck', tone: 'amber' },
  { id: 'flagged', label: 'Flagged content',   value: 7,                    icon: 'Flag',      tone: 'red' },
  { id: 'live',    label: 'Live offers',        value: 412,                  icon: 'Radio',     tone: 'teal' },
  { id: 'mrr',     label: 'Platform MRR',       value: '$6.1k',              icon: 'TrendingUp',tone: 'green' },
];

Object.assign(window, {
  VENDOR_QUEUE, FEED_SEED, FEED_TEMPLATES, REGIONS, SUBS, AUDIO_GUIDES, DEPTH_INFO, ADMIN_KPIS,
});
