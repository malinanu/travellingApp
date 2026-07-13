// data.jsx — Spotto content layer (places, prefs). Exports to window.

const TRAVEL_STYLES = [
  { id: 'nomad',      label: 'Digital Nomad', icon: 'Laptop',    blurb: 'Fast Wi-Fi, good coffee, places to focus.' },
  { id: 'surf',       label: 'Surf Traveler', icon: 'Waves',     blurb: 'Breaks, board rentals, dawn patrol.' },
  { id: 'luxury',     label: 'Luxury',        icon: 'Gem',       blurb: 'Refined stays, fine dining, sunset views.' },
  { id: 'backpacker', label: 'Backpacker',    icon: 'Backpack',  blurb: 'Cheap eats, hostels, good company.' },
];

const INTERESTS = [
  { id: 'coffee',    label: 'Coffee',     icon: 'Coffee' },
  { id: 'coworking', label: 'Coworking',  icon: 'Laptop' },
  { id: 'nightlife', label: 'Nightlife',  icon: 'Martini' },
  { id: 'surf',      label: 'Surf',       icon: 'Waves' },
  { id: 'yoga',      label: 'Yoga',       icon: 'Flower2' },
  { id: 'food',      label: 'Local Food', icon: 'UtensilsCrossed' },
  { id: 'beach',     label: 'Beaches',    icon: 'Palmtree' },
  { id: 'music',     label: 'Live Music', icon: 'Music' },
];

// x / y are percentages on the map canvas (0–100)
const PLACES = [
  {
    id: 'driftwood',
    name: 'Driftwood Coffee & Co-Work',
    type: 'Coworking Café',
    icon: 'Coffee',
    color: 'teal',
    x: 33, y: 38,
    rating: 4.8,
    reviews: 212,
    distance: '400 m',
    priceLevel: '$$',
    wifi: { score: 9.2, speed: '88 Mbps', label: 'Blazing' },
    hours: '7:00 AM – 9:00 PM',
    openNow: true,
    bestTimes: ['morning', 'afternoon'],
    deal: { active: true, label: 'Free flat white with any day pass', tag: 'Perk' },
    tags: ['Coffee', 'Coworking', 'Fast Wi-Fi', 'Power outlets'],
    blurb: 'Airy garden café with a dedicated co-work loft, standing desks and the fastest fibre line in the bay. Sea-view seats on the upper deck.',
    seaView: true,
    photo: 'cowork',
    menu: [
      { id: 's1', name: 'Breakfast', items: [
        { id: 'm1', name: 'Coconut roti & curd bowl', price: 950,  desc: 'Pol roti, kithul treacle, buffalo curd', icon: 'EggFried' },
        { id: 'm2', name: 'Surfer’s big plate',  price: 1450, desc: 'Eggs any way, sourdough, sambol, bacon', icon: 'Sandwich' },
      ]},
      { id: 's2', name: 'Mains', items: [
        { id: 'm3', name: 'Jackfruit curry & rice',   price: 1200, desc: 'Young jak, red rice, papadam',         icon: 'Soup' },
        { id: 'm4', name: 'Grilled catch of the day', price: 1850, desc: 'From the bay — ask what came in',      icon: 'Fish' },
      ]},
      { id: 's3', name: 'Drinks', items: [
        { id: 'm5', name: 'Single-origin pour-over',  price: 750,  desc: 'Ceylon arabica, brewed to order',      icon: 'Coffee' },
        { id: 'm6', name: 'King coconut',             price: 350,  desc: 'Straight off the cart, chilled',       icon: 'GlassWater' },
      ]},
    ],
  },
  {
    id: 'baysurf',
    name: 'Weligama Bay Surf School',
    type: 'Surf School',
    icon: 'Waves',
    color: 'coral',
    x: 62, y: 64,
    rating: 4.9,
    reviews: 540,
    distance: '650 m',
    priceLevel: '$$',
    wifi: { score: 6.1, speed: '24 Mbps', label: 'Okay' },
    hours: '6:00 AM – 6:00 PM',
    openNow: true,
    bestTimes: ['morning', 'afternoon'],
    deal: { active: true, label: '20% off beginner group lessons — today only', tag: '20% OFF' },
    tags: ['Surf', 'Board rental', 'Beginner friendly'],
    blurb: 'Right on the gentle beach-break that made Weligama famous for first-timers. Boards, rash guards and ISA-certified instructors.',
    seaView: true,
    photo: 'surf',
  },
  {
    id: 'hangtime',
    name: 'Hangtime Rooftop & Sunset Bar',
    type: 'Sunset Bar',
    icon: 'Martini',
    color: 'sun',
    x: 48, y: 22,
    rating: 4.7,
    reviews: 388,
    distance: '900 m',
    priceLevel: '$$$',
    wifi: { score: 7.0, speed: '31 Mbps', label: 'Good' },
    hours: '4:00 PM – 1:00 AM',
    openNow: false,
    bestTimes: ['evening'],
    deal: { active: true, label: 'Happy hour 5–7 PM · 2-for-1 arrack cocktails', tag: 'Happy Hr' },
    tags: ['Cocktails', 'Live DJ', 'Ocean view', 'Popular at sunset'],
    blurb: 'Palm-fringed rooftop facing due west over the bay — the spot locals point to when the sky turns gold. DJ sets Thu–Sun.',
    seaView: true,
    popularEvening: true,
    photo: 'sunset',
    menu: [
      { id: 's1', name: 'Cocktails', items: [
        { id: 'm1', name: 'Arrack sour',        price: 1400, desc: 'Ceylon arrack, lime, kithul, egg white', icon: 'Martini' },
        { id: 'm2', name: 'Bay sunset spritz',  price: 1600, desc: 'Passionfruit, prosecco, soda',           icon: 'Wine' },
      ]},
      { id: 's2', name: 'Bites', items: [
        { id: 'm3', name: 'Hot butter cuttlefish', price: 1750, desc: 'The rooftop classic, extra lime',     icon: 'Flame' },
        { id: 'm4', name: 'Isso vade plate',       price: 950,  desc: 'Prawn lentil fritters, tamarind dip', icon: 'Cookie' },
      ]},
    ],
  },
  {
    id: 'claystudio',
    name: 'Clay & Co Pottery Studio',
    type: 'Pottery Workshop',
    icon: 'Palette',
    color: 'clay',
    x: 19, y: 34,
    rating: 4.9,
    reviews: 96,
    distance: '300 m',
    priceLevel: '$$',
    wifi: { score: 5.0, speed: '18 Mbps', label: 'Okay' },
    hours: '9:00 AM – 6:00 PM',
    openNow: true,
    bestTimes: ['morning', 'afternoon'],
    indoor: true, artisan: true,
    deal: { active: true, label: 'Drop-in wheel-throwing taster — Rs 1,500', tag: 'Workshop' },
    tags: ['Pottery', 'Hands-on', 'Indoor', 'Local artisan'],
    blurb: 'A working clay studio tucked off the main road. Spin the wheel with resident potters — the perfect rainy-day plan when the surf is blown out.',
    seaView: false,
    photo: 'clay',
  },
  {
    id: 'batik',
    name: 'Sea Salt Batik Atelier',
    type: 'Batik Atelier',
    icon: 'Brush',
    color: 'plum',
    x: 73, y: 30,
    rating: 4.8,
    reviews: 73,
    distance: '750 m',
    priceLevel: '$$',
    wifi: { score: 5.5, speed: '20 Mbps', label: 'Okay' },
    hours: '9:30 AM – 5:30 PM',
    openNow: true,
    bestTimes: ['morning', 'afternoon'],
    indoor: true, artisan: true,
    deal: { active: false },
    tags: ['Batik', 'Hand-dyed', 'Indoor', 'Local artisan'],
    blurb: 'Family atelier hand-waxing and dyeing batik the traditional way. Watch a panel come to life, or wax your own under the canopy.',
    seaView: false,
    photo: 'batik',
  },
];

const TIMES = [
  { id: 'morning',   label: 'Morning',   icon: 'Sunrise', range: '6–11 AM' },
  { id: 'afternoon', label: 'Afternoon', icon: 'Sun',     range: '11–4 PM' },
  { id: 'evening',   label: 'Evening',   icon: 'Sunset',  range: '4–10 PM' },
];

const WEATHER = {
  sunny:  { label: 'Sunny',  icon: 'Sun',       temp: '31°', tip: 'Perfect beach & surf weather' },
  cloudy: { label: 'Cloudy', icon: 'CloudSun',  temp: '29°', tip: 'Warm and a little hazy' },
  rainy:  { label: 'Rainy',  icon: 'CloudRain', temp: '26°', tip: 'Monsoon shower — swap to indoor picks' },
};

// traveller roster for Group Mode (x/y are map percentages).
// `in` = seeded into the group; the rest are nearby travellers you can add.
const TRAVELERS = [
  { id: 'mia',   name: 'Mia',    initial: 'M', color: '#f0682f', x: 36, y: 33, at: 'Driftwood Coffee', sub: 'Surfing the main break', in: true },
  { id: 'theo',  name: 'Theo',   initial: 'T', color: '#7c5cff', x: 60, y: 58, at: 'Bay Surf School',   sub: 'On for sunset drinks',   in: true },
  { id: 'ines',  name: 'Inès',   initial: 'I', color: '#0ea5e9', x: 50, y: 26, at: 'Hangtime Rooftop',  sub: 'Café-hopping today',     in: true },
  { id: 'kai',   name: 'Kai',    initial: 'K', color: '#0b6f86', x: 28, y: 64, at: 'Lazy Left',         sub: 'Also in Weligama',       in: false },
  { id: 'sofia', name: 'Sofia',  initial: 'S', color: '#db2777', x: 70, y: 38, at: 'Clay Studio',       sub: 'From your Ella trip',    in: false },
  { id: 'noah',  name: 'Noah',   initial: 'N', color: '#eaa200', x: 44, y: 48, at: 'Coconut Tree Hill', sub: '2 mutual travellers',    in: false },
];

// "you" — the current account, shown as the group host
const ME = { id: 'me', name: 'You', initial: 'LM', color: '#0b6f86', at: 'Weligama town', sub: 'Trip host' };

// the active group: name, shareable code, and who's in it
const GROUP_SEED = {
  name: 'Weligama Crew',
  code: 'WLG-4K2',
  memberIds: ['mia', 'theo', 'ines'],
};

// kept for backward-compat (map avatars now come from group state)
const FRIENDS = TRAVELERS.filter((t) => t.in);

// current-vibe tags travelers can crowd-source
const VIBES = [
  { id: 'productive', label: 'Productive', icon: 'Laptop',  color: '#0b6f86' },
  { id: 'loud',       label: 'Loud',       icon: 'Volume2', color: '#f0682f' },
  { id: 'empty',      label: 'Empty',      icon: 'Wind',    color: '#6c8582' },
];

// seed vibe vote counts per place
const VIBE_SEED = {
  driftwood: { productive: 28, loud: 3,  empty: 4 },
  baysurf:   { productive: 5,  loud: 14, empty: 6 },
  hangtime:  { productive: 2,  loud: 31, empty: 1 },
  claystudio:{ productive: 12, loud: 2,  empty: 9 },
  batik:     { productive: 7,  loud: 1,  empty: 11 },
};

Object.assign(window, { TRAVELERS, ME, GROUP_SEED, TRAVEL_STYLES, INTERESTS, PLACES, TIMES, WEATHER, FRIENDS, VIBES, VIBE_SEED });
