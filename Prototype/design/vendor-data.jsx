// vendor-data.jsx — Spotto Vendor Portal demo data for "Driftwood Coffee & Co-Work".

const VENDOR = {
  name: 'Driftwood Coffee & Co-Work',
  type: 'Coworking Café',
  typeId: 'cafe',            // one of BUSINESS_TYPES ids
  area: 'Weligama',
  initials: 'DC',
  owner: 'Tharindu Perera',
};

// business categories — `food: true` unlocks the Menu tab
const BUSINESS_TYPES = [
  { id: 'restaurant', label: 'Restaurant',     icon: 'Utensils', food: true },
  { id: 'cafe',       label: 'Café',           icon: 'Coffee',   food: true },
  { id: 'bar',        label: 'Bar & Sunset',   icon: 'Martini',  food: true },
  { id: 'surf',       label: 'Surf & Water',   icon: 'Waves' },
  { id: 'craft',      label: 'Craft & Studio', icon: 'Palette' },
  { id: 'shop',       label: 'Shop',           icon: 'Store' },
];

// sectioned menu — items: price in Rs (number), img = uploaded object URL (session-only),
// icon = tinted fallback tile when there's no photo
const MENU = [
  { id: 's1', name: 'Breakfast', items: [
    { id: 'm1', name: 'Coconut roti & curd bowl', price: 950,  desc: 'Pol roti, kithul treacle, buffalo curd', img: null, icon: 'EggFried' },
    { id: 'm2', name: 'Surfer’s big plate',  price: 1450, desc: 'Eggs any way, sourdough, sambol, bacon', img: null, icon: 'Sandwich' },
  ]},
  { id: 's2', name: 'Mains', items: [
    { id: 'm3', name: 'Jackfruit curry & rice',   price: 1200, desc: 'Young jak, red rice, papadam',           img: null, icon: 'Soup' },
    { id: 'm4', name: 'Grilled catch of the day', price: 1850, desc: 'From the bay — ask what came in',   img: null, icon: 'Fish' },
  ]},
  { id: 's3', name: 'Drinks', items: [
    { id: 'm5', name: 'Single-origin pour-over',  price: 750,  desc: 'Ceylon arabica, brewed to order',        img: null, icon: 'Coffee' },
    { id: 'm6', name: 'King coconut',             price: 350,  desc: 'Straight off the cart, chilled',         img: null, icon: 'GlassWater' },
  ]},
];

const KPIS = [
  { id: 'offers',  label: 'Active Offers',     value: 3,      icon: 'Tag',        delta: null,    tone: 'teal',  hint: '1 ending soon' },
  { id: 'redeem',  label: 'Total Redemptions', value: 248,    icon: 'Ticket',     delta: +18,     tone: 'coral', hint: 'this week' },
  { id: 'views',   label: 'Profile Views',     value: 1940,   icon: 'Eye',        delta: +12,     tone: 'sun',   hint: 'last 7 days' },
];

// active offers w/ claim progress
const OFFERS = [
  { id: 'o1', title: 'Free flat white with any day pass', type: 'Freebie', value: 'Free coffee',
    status: 'live', start: 'Today 07:00', end: 'Today 17:00', claimed: 34, limit: 50, redeemed: 21 },
  { id: 'o2', title: '20% off all single-origin pour-overs', type: 'Percentage', value: '20% off',
    status: 'live', start: 'Mon', end: 'Sun', claimed: 88, limit: 200, redeemed: 61 },
  { id: 'o3', title: 'Buy 5 day-passes, get 1 free', type: 'Bundle', value: '6-for-5',
    status: 'ending', start: 'This week', end: 'Today 23:59', claimed: 47, limit: 50, redeemed: 39 },
  { id: 'o4', title: 'Rainy-day bundle: coffee + covered desk', type: 'Bundle', value: '2-for-1',
    status: 'draft', start: '—', end: '—', claimed: 0, limit: 40, redeemed: 0 },
  { id: 'o5', title: 'Full-moon night: free ginger tea', type: 'Freebie', value: 'Free tea',
    status: 'ended', start: 'Last Fri', end: 'Last Sat', claimed: 50, limit: 50, redeemed: 44 },
];

const DISCOUNT_TYPES = [
  { id: 'pct',     label: 'Percentage off', icon: 'Percent',     hint: 'e.g. 20% off' },
  { id: 'amount',  label: 'Amount off',     icon: 'BadgeDollarSign', hint: 'e.g. Rs 500 off' },
  { id: 'freebie', label: 'Freebie',        icon: 'Gift',        hint: 'e.g. free coffee' },
  { id: 'bundle',  label: 'Bundle',         icon: 'Package',     hint: 'e.g. 2-for-1' },
];

// 7-day traffic: views + redemptions
const WEEK = [
  { day: 'Mon', views: 210, redemptions: 24 },
  { day: 'Tue', views: 248, redemptions: 31 },
  { day: 'Wed', views: 196, redemptions: 22 },
  { day: 'Thu', views: 312, redemptions: 44 },
  { day: 'Fri', views: 358, redemptions: 52 },
  { day: 'Sat', views: 401, redemptions: 61 },
  { day: 'Sun', views: 285, redemptions: 14 },
];

// popular times — by hour (7:00–20:00), value = relative footfall 0–100
const POPULAR = [
  { h: 7, v: 28 }, { h: 8, v: 52 }, { h: 9, v: 86 }, { h: 10, v: 100 },
  { h: 11, v: 78 }, { h: 12, v: 60 }, { h: 13, v: 48 }, { h: 14, v: 66 },
  { h: 15, v: 82 }, { h: 16, v: 71 }, { h: 17, v: 44 }, { h: 18, v: 30 },
  { h: 19, v: 22 }, { h: 20, v: 14 },
];

// your redemptions vs Weligama café category index, indexed weeks
const TREND = [
  { wk: 'W1', you: 38, cat: 30 },
  { wk: 'W2', you: 44, cat: 33 },
  { wk: 'W3', you: 41, cat: 36 },
  { wk: 'W4', you: 58, cat: 39 },
  { wk: 'W5', you: 72, cat: 42 },
  { wk: 'W6', you: 86, cat: 47 },
];

// recent redemption feed
const REDEMPTIONS = [
  { id: 'r1', code: 'SPOTTO-DRIF', name: 'Lena (Germany)',     offer: 'Free flat white', time: '2 min ago',  style: 'Digital Nomad' },
  { id: 'r2', code: 'SPOTTO-9KQ2', name: 'Marco (Italy)',       offer: '20% off pour-over', time: '18 min ago', style: 'Surf Traveler' },
  { id: 'r3', code: 'SPOTTO-7HX1', name: 'Aiko (Japan)',        offer: 'Free flat white', time: '41 min ago', style: 'Digital Nomad' },
  { id: 'r4', code: 'SPOTTO-2BVT', name: 'Sam (Australia)',     offer: 'Day-pass bundle', time: '1 hr ago',   style: 'Backpacker' },
  { id: 'r5', code: 'SPOTTO-MR48', name: 'Chloé (France)',      offer: '20% off pour-over', time: '2 hr ago',  style: 'Luxury' },
];

// valid codes the scanner accepts (others fail)
const VALID_CODES = {
  'SPOTTO-DRIF': { name: 'Lena M.', country: 'Germany', style: 'Digital Nomad', offer: 'Free flat white with any day pass', value: 'Free coffee' },
  'SPOTTO-X4K9': { name: 'Tom R.', country: 'UK',       style: 'Surf Traveler', offer: '20% off all single-origin pour-overs', value: '20% off' },
  'SPOTTO-88Q2': { name: 'Sara P.', country: 'Spain',   style: 'Digital Nomad', offer: 'Buy 5 day-passes, get 1 free', value: '6-for-5' },
};

const PROFILE = {
  wifi: { enabled: true, score: 9.2, speed: 88 },
  bestTimes: ['morning', 'afternoon'],
  seaView: true,
  hours: { open: '07:00', close: '21:00' },
  amenities: ['Power outlets', 'Standing desks', 'Air-con loft', 'Specialty coffee', 'Quiet zone', 'Sea-view deck'],
  blurb: 'Airy garden café with a dedicated co-work loft, standing desks and the fastest fibre line in the bay. Sea-view seats on the upper deck.',
};

// ── Growth Tools data ──

// travelers currently within range for a flash sale (simulated GEO radius)
const FLASH = { radiusKm: 3, inRange: 142, nomads: 58, surfers: 44, lastSentMins: 184 };

// digital stamp card: how close travelers are to the reward
const STAMP_CARD = {
  reward: 'Free flat white', stampsNeeded: 6,
  buckets: [
    { away: 1, count: 23, label: 'One stamp away' },
    { away: 2, count: 41, label: 'Two away' },
    { away: 3, count: 67, label: 'Three+ away' },
  ],
  totalHolders: 214, redeemedThisMonth: 38,
};

// behind-the-scenes clips
const BTS_CLIPS = [
  { id: 'b1', title: 'Pulling the morning espresso', secs: 9,  views: 1240, grad: 'linear-gradient(135deg,#0e8c8a,#0a5f6e)', icon: 'Coffee' },
  { id: 'b2', title: 'Roasting Ceylon beans',        secs: 10, views: 880,  grad: 'linear-gradient(135deg,#c2683f,#7a3b22)', icon: 'Flame' },
  { id: 'b3', title: 'Latte art, slow-mo',            secs: 7,  views: 2010, grad: 'linear-gradient(135deg,#0b4a54,#0b6f86)', icon: 'Sparkles' },
];

// sponsored boost options
const BOOST = {
  event: { name: 'Friday Sunset Sessions', when: 'Fri · 5–9 PM', dj: 'Live DJ + arrack cocktails' },
  tiers: [
    { id: 'local',   label: 'Local map top spot',   reach: '~1,200 travelers', days: 3, price: 12, icon: 'MapPin' },
    { id: 'category',label: 'Category feature',      reach: '~3,400 travelers', days: 5, price: 24, icon: 'Star', best: true },
    { id: 'push',    label: 'Push + map + category', reach: '~6,800 travelers', days: 7, price: 39, icon: 'Megaphone' },
  ],
};

Object.assign(window, {
  VENDOR, KPIS, OFFERS, DISCOUNT_TYPES, WEEK, POPULAR, TREND, REDEMPTIONS, VALID_CODES, PROFILE,
  FLASH, STAMP_CARD, BTS_CLIPS, BOOST, BUSINESS_TYPES, MENU,
});
