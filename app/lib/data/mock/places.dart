// Mock content layer — 1:1 port of Prototype/design/data.jsx.

import '../models/models.dart';

const travelStyles = [
  TravelStyle(id: 'nomad', label: 'Digital Nomad', icon: 'Laptop', blurb: 'Fast Wi-Fi, good coffee, places to focus.'),
  TravelStyle(id: 'surf', label: 'Surf Traveler', icon: 'Waves', blurb: 'Breaks, board rentals, dawn patrol.'),
  TravelStyle(id: 'luxury', label: 'Luxury', icon: 'Gem', blurb: 'Refined stays, fine dining, sunset views.'),
  TravelStyle(id: 'backpacker', label: 'Backpacker', icon: 'Backpack', blurb: 'Cheap eats, hostels, good company.'),
];

const interests = [
  Interest(id: 'coffee', label: 'Coffee', icon: 'Coffee'),
  Interest(id: 'coworking', label: 'Coworking', icon: 'Laptop'),
  Interest(id: 'nightlife', label: 'Nightlife', icon: 'Martini'),
  Interest(id: 'surf', label: 'Surf', icon: 'Waves'),
  Interest(id: 'yoga', label: 'Yoga', icon: 'Flower2'),
  Interest(id: 'food', label: 'Local Food', icon: 'UtensilsCrossed'),
  Interest(id: 'beach', label: 'Beaches', icon: 'Palmtree'),
  Interest(id: 'music', label: 'Live Music', icon: 'Music'),
  Interest(id: 'hiking', label: 'Hiking', icon: 'Mountain'),
  Interest(id: 'vegan', label: 'Vegan / Plant-based', icon: 'Flower2'),
  Interest(id: 'halal', label: 'Halal Food', icon: 'UtensilsCrossed'),
  Interest(id: 'gluten-free', label: 'Gluten-Free', icon: 'UtensilsCrossed'),
  Interest(id: 'wifi', label: 'Fast Wi-Fi', icon: 'Wifi'),
  Interest(id: 'outlets', label: 'Power Outlets', icon: 'Zap'),
  Interest(id: 'quiet', label: 'Quiet Zone', icon: 'Volume2'),
];

const places = [
  Place(
    id: 'driftwood',
    name: 'Driftwood Coffee & Co-Work',
    type: 'Coworking Café',
    categoryId: 'coworking-cafe',
    icon: 'Coffee',
    color: 'teal',
    x: 33, y: 38,
    lat: 5.9731, lng: 80.4280,
    rating: 4.8,
    reviews: 212,
    distance: '400 m',
    priceLevel: r'$$',
    wifi: WifiInfo(score: 9.2, speed: '88 Mbps', label: 'Blazing'),
    hours: '7:00 AM – 9:00 PM',
    openNow: true,
    bestTimes: ['morning', 'afternoon'],
    deal: Deal(active: true, label: 'Free flat white with any day pass', tag: 'Perk'),
    tags: ['Coffee', 'Coworking', 'Fast Wi-Fi', 'Power outlets'],
    blurb: 'Airy garden café with a dedicated co-work loft, standing desks and the fastest fibre line in the bay. Sea-view seats on the upper deck.',
    seaView: true,
    photo: 'cowork',
    menu: [
      MenuSection(id: 's1', name: 'Breakfast', items: [
        MenuItem(id: 'm1', name: 'Coconut roti & curd bowl', price: 950, desc: 'Pol roti, kithul treacle, buffalo curd', icon: 'EggFried'),
        MenuItem(id: 'm2', name: 'Surfer’s big plate', price: 1450, desc: 'Eggs any way, sourdough, sambol, bacon', icon: 'Sandwich'),
      ]),
      MenuSection(id: 's2', name: 'Mains', items: [
        MenuItem(id: 'm3', name: 'Jackfruit curry & rice', price: 1200, desc: 'Young jak, red rice, papadam', icon: 'Soup'),
        MenuItem(id: 'm4', name: 'Grilled catch of the day', price: 1850, desc: 'From the bay — ask what came in', icon: 'Fish'),
      ]),
      MenuSection(id: 's3', name: 'Drinks', items: [
        MenuItem(id: 'm5', name: 'Single-origin pour-over', price: 750, desc: 'Ceylon arabica, brewed to order', icon: 'Coffee'),
        MenuItem(id: 'm6', name: 'King coconut', price: 350, desc: 'Straight off the cart, chilled', icon: 'GlassWater'),
      ]),
    ],
  ),
  Place(
    id: 'baysurf',
    name: 'Weligama Bay Surf School',
    type: 'Surf School',
    categoryId: 'surf-school',
    icon: 'Waves',
    color: 'coral',
    x: 62, y: 64,
    lat: 5.9711, lng: 80.4301,
    rating: 4.9,
    reviews: 540,
    distance: '650 m',
    priceLevel: r'$$',
    wifi: WifiInfo(score: 6.1, speed: '24 Mbps', label: 'Okay'),
    hours: '6:00 AM – 6:00 PM',
    openNow: true,
    bestTimes: ['morning', 'afternoon'],
    deal: Deal(active: true, label: '20% off beginner group lessons — today only', tag: '20% OFF'),
    tags: ['Surf', 'Board rental', 'Beginner friendly'],
    blurb: 'Right on the gentle beach-break that made Weligama famous for first-timers. Boards, rash guards and ISA-certified instructors.',
    seaView: true,
    photo: 'surf',
  ),
  Place(
    id: 'hangtime',
    name: 'Hangtime Rooftop & Sunset Bar',
    type: 'Sunset Bar',
    categoryId: 'bar-pub',
    icon: 'Martini',
    color: 'sun',
    x: 48, y: 22,
    lat: 5.9744, lng: 80.4291,
    rating: 4.7,
    reviews: 388,
    distance: '900 m',
    priceLevel: r'$$$',
    wifi: WifiInfo(score: 7.0, speed: '31 Mbps', label: 'Good'),
    hours: '4:00 PM – 1:00 AM',
    openNow: false,
    bestTimes: ['evening'],
    deal: Deal(active: true, label: 'Happy hour 5–7 PM · 2-for-1 arrack cocktails', tag: 'Happy Hr'),
    tags: ['Cocktails', 'Live DJ', 'Ocean view', 'Popular at sunset'],
    blurb: 'Palm-fringed rooftop facing due west over the bay — the spot locals point to when the sky turns gold. DJ sets Thu–Sun.',
    seaView: true,
    popularEvening: true,
    photo: 'sunset',
    menu: [
      MenuSection(id: 's1', name: 'Cocktails', items: [
        MenuItem(id: 'm1', name: 'Arrack sour', price: 1400, desc: 'Ceylon arrack, lime, kithul, egg white', icon: 'Martini'),
        MenuItem(id: 'm2', name: 'Bay sunset spritz', price: 1600, desc: 'Passionfruit, prosecco, soda', icon: 'Wine'),
      ]),
      MenuSection(id: 's2', name: 'Bites', items: [
        MenuItem(id: 'm3', name: 'Hot butter cuttlefish', price: 1750, desc: 'The rooftop classic, extra lime', icon: 'Flame'),
        MenuItem(id: 'm4', name: 'Isso vade plate', price: 950, desc: 'Prawn lentil fritters, tamarind dip', icon: 'Cookie'),
      ]),
    ],
  ),
  Place(
    id: 'claystudio',
    name: 'Clay & Co Pottery Studio',
    type: 'Pottery Workshop',
    categoryId: 'pottery-workshop',
    icon: 'Palette',
    color: 'clay',
    x: 19, y: 34,
    lat: 5.9734, lng: 80.4270,
    rating: 4.9,
    reviews: 96,
    distance: '300 m',
    priceLevel: r'$$',
    wifi: WifiInfo(score: 5.0, speed: '18 Mbps', label: 'Okay'),
    hours: '9:00 AM – 6:00 PM',
    openNow: true,
    bestTimes: ['morning', 'afternoon'],
    indoor: true,
    artisan: true,
    deal: Deal(active: true, label: 'Drop-in wheel-throwing taster — Rs 1,500', tag: 'Workshop'),
    tags: ['Pottery', 'Hands-on', 'Indoor', 'Local artisan'],
    blurb: 'A working clay studio tucked off the main road. Spin the wheel with resident potters — the perfect rainy-day plan when the surf is blown out.',
    seaView: false,
    photo: 'clay',
  ),
  Place(
    id: 'batik',
    name: 'Sea Salt Batik Atelier',
    type: 'Batik Atelier',
    categoryId: 'batik-atelier',
    icon: 'Brush',
    color: 'plum',
    x: 73, y: 30,
    lat: 5.9738, lng: 80.4309,
    rating: 4.8,
    reviews: 73,
    distance: '750 m',
    priceLevel: r'$$',
    wifi: WifiInfo(score: 5.5, speed: '20 Mbps', label: 'Okay'),
    hours: '9:30 AM – 5:30 PM',
    openNow: true,
    bestTimes: ['morning', 'afternoon'],
    indoor: true,
    artisan: true,
    deal: Deal.none,
    tags: ['Batik', 'Hand-dyed', 'Indoor', 'Local artisan'],
    blurb: 'Family atelier hand-waxing and dyeing batik the traditional way. Watch a panel come to life, or wax your own under the canopy.',
    seaView: false,
    photo: 'batik',
  ),
];

const timeSlots = [
  TimeSlot(id: 'morning', label: 'Morning', icon: 'Sunrise', range: '6–11 AM'),
  TimeSlot(id: 'afternoon', label: 'Afternoon', icon: 'Sun', range: '11–4 PM'),
  TimeSlot(id: 'evening', label: 'Evening', icon: 'Sunset', range: '4–10 PM'),
];

const weatherKinds = {
  'sunny': WeatherInfo(id: 'sunny', label: 'Sunny', icon: 'Sun', temp: '31°', tip: 'Perfect beach & surf weather'),
  'cloudy': WeatherInfo(id: 'cloudy', label: 'Cloudy', icon: 'CloudSun', temp: '29°', tip: 'Warm and a little hazy'),
  'rainy': WeatherInfo(id: 'rainy', label: 'Rainy', icon: 'CloudRain', temp: '26°', tip: 'Monsoon shower — swap to indoor picks'),
};

// `travelers`/`me` start as this hardcoded offline fallback, then get
// overwritten with real data (group members + nearby travellers resolved
// via Traveler.fromGroupMember/fromNearby/fromMe) once available — same
// offline-first pattern as tripCatalog in trip_catalog.dart. Kept around so
// tripWho() (trip_common.dart) and any other id → Traveler lookups work
// before the first successful fetch.
var travelers = [
  Traveler(id: 'mia', name: 'Mia', initial: 'M', color: 0xFFF0682F, x: 36, y: 33, lat: 5.9735, lng: 80.4282, at: 'Driftwood Coffee', sub: 'Surfing the main break', seededIn: true),
  Traveler(id: 'theo', name: 'Theo', initial: 'T', color: 0xFF7C5CFF, x: 60, y: 58, lat: 5.9715, lng: 80.4300, at: 'Bay Surf School', sub: 'On for sunset drinks', seededIn: true),
  Traveler(id: 'ines', name: 'Inès', initial: 'I', color: 0xFF0EA5E9, x: 50, y: 26, lat: 5.9741, lng: 80.4292, at: 'Hangtime Rooftop', sub: 'Café-hopping today', seededIn: true),
  Traveler(id: 'kai', name: 'Kai', initial: 'K', color: 0xFF0B6F86, x: 28, y: 64, lat: 5.9710, lng: 80.4276, at: 'Lazy Left', sub: 'Also in Weligama'),
  Traveler(id: 'sofia', name: 'Sofia', initial: 'S', color: 0xFFDB2777, x: 70, y: 38, lat: 5.9731, lng: 80.4307, at: 'Clay Studio', sub: 'From your Ella trip'),
  Traveler(id: 'noah', name: 'Noah', initial: 'N', color: 0xFFEAA200, x: 44, y: 48, lat: 5.9723, lng: 80.4288, at: 'Coconut Tree Hill', sub: '2 mutual travellers'),
];

var me = const Traveler(id: 'me', name: 'You', initial: 'LM', color: 0xFF0B6F86, at: 'Weligama town', sub: 'Trip host');

const groupSeed = Group(name: 'Weligama Crew', code: 'WLG-4K2', memberIds: ['mia', 'theo', 'ines']);

const vibes = [
  VibeTag(id: 'productive', label: 'Productive', icon: 'Laptop', color: 0xFF0B6F86),
  VibeTag(id: 'loud', label: 'Loud', icon: 'Volume2', color: 0xFFF0682F),
  VibeTag(id: 'crowded', label: 'Crowded', icon: 'Users', color: 0xFF6C8582),
];
