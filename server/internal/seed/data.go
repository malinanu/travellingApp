// Package seed loads the demo dataset ported from the prototype's data files
// (app/lib/data/mock/places.dart, Prototype/design/vendor-data.jsx,
// Prototype/design/admin-data.jsx). Relative-time strings become timestamps
// offset from now.
package seed

// FeedTemplate feeds the API's live-content generator goroutine.
type FeedTemplate struct {
	Vendor, Town, Type, Text, Status, Reason string
}

// FeedTemplates is FEED_TEMPLATES from admin-data.jsx.
var FeedTemplates = []FeedTemplate{
	{"Unawatuna Dive", "Unawatuna", "Offer", "Discover-scuba intro · 20% off", "ok", ""},
	{"Hiriketiya Surf", "Dickwella", "Offer", "Beginner group lesson deal", "ok", ""},
	{"Colombo Roasters", "Colombo", "Event", "Latte art throwdown · Friday", "ok", ""},
	{"Best Deals Lanka", "Hikkaduwa", "Offer", "FREE MONEY click here to win $$$", "flag", "Phishing pattern detected"},
	{"Sigiriya Sunrise", "Sigiriya", "Event", "Guided rock climb at dawn", "review", "Unverified safety cert"},
	{"Jaffna Palmyra", "Jaffna", "Offer", "Authentic crab curry · lunch set", "ok", ""},
}

// placeDriftwoodData …placeBatikData mirror the Dart Place model in
// app/lib/data/mock/places.dart (id/name/vendorId live on the row).
const placeDriftwoodData = `{
  "type": "Coworking Café", "icon": "Coffee", "color": "teal",
  "x": 33, "y": 38, "rating": 4.8, "reviews": 212, "distance": "400 m", "priceLevel": "$$",
  "wifi": {"score": 9.2, "speed": "88 Mbps", "label": "Blazing"},
  "hours": "7:00 AM – 9:00 PM", "openNow": true, "bestTimes": ["morning", "afternoon"],
  "deal": {"active": true, "label": "Free flat white with any day pass", "tag": "Perk"},
  "tags": ["Coffee", "Coworking", "Fast Wi-Fi", "Power outlets"],
  "blurb": "Airy garden café with a dedicated co-work loft, standing desks and the fastest fibre line in the bay. Sea-view seats on the upper deck.",
  "seaView": true, "photo": "cowork",
  "vibes": {"productive": 28, "loud": 3, "empty": 4},
  "menu": [
    {"id": "s1", "name": "Breakfast", "items": [
      {"id": "m1", "name": "Coconut roti & curd bowl", "price": 950, "desc": "Pol roti, kithul treacle, buffalo curd", "icon": "EggFried"},
      {"id": "m2", "name": "Surfer’s big plate", "price": 1450, "desc": "Eggs any way, sourdough, sambol, bacon", "icon": "Sandwich"}
    ]},
    {"id": "s2", "name": "Mains", "items": [
      {"id": "m3", "name": "Jackfruit curry & rice", "price": 1200, "desc": "Young jak, red rice, papadam", "icon": "Soup"},
      {"id": "m4", "name": "Grilled catch of the day", "price": 1850, "desc": "From the bay — ask what came in", "icon": "Fish"}
    ]},
    {"id": "s3", "name": "Drinks", "items": [
      {"id": "m5", "name": "Single-origin pour-over", "price": 750, "desc": "Ceylon arabica, brewed to order", "icon": "Coffee"},
      {"id": "m6", "name": "King coconut", "price": 350, "desc": "Straight off the cart, chilled", "icon": "GlassWater"}
    ]}
  ]
}`

const placeBaysurfData = `{
  "type": "Surf School", "icon": "Waves", "color": "coral",
  "x": 62, "y": 64, "rating": 4.9, "reviews": 540, "distance": "650 m", "priceLevel": "$$",
  "wifi": {"score": 6.1, "speed": "24 Mbps", "label": "Okay"},
  "hours": "6:00 AM – 6:00 PM", "openNow": true, "bestTimes": ["morning", "afternoon"],
  "deal": {"active": true, "label": "20% off beginner group lessons — today only", "tag": "20% OFF"},
  "tags": ["Surf", "Board rental", "Beginner friendly"],
  "blurb": "Right on the gentle beach-break that made Weligama famous for first-timers. Boards, rash guards and ISA-certified instructors.",
  "seaView": true, "photo": "surf",
  "vibes": {"productive": 5, "loud": 14, "empty": 6}
}`

const placeHangtimeData = `{
  "type": "Sunset Bar", "icon": "Martini", "color": "sun",
  "x": 48, "y": 22, "rating": 4.7, "reviews": 388, "distance": "900 m", "priceLevel": "$$$",
  "wifi": {"score": 7.0, "speed": "31 Mbps", "label": "Good"},
  "hours": "4:00 PM – 1:00 AM", "openNow": false, "bestTimes": ["evening"],
  "deal": {"active": true, "label": "Happy hour 5–7 PM · 2-for-1 arrack cocktails", "tag": "Happy Hr"},
  "tags": ["Cocktails", "Live DJ", "Ocean view", "Popular at sunset"],
  "blurb": "Palm-fringed rooftop facing due west over the bay — the spot locals point to when the sky turns gold. DJ sets Thu–Sun.",
  "seaView": true, "popularEvening": true, "photo": "sunset",
  "vibes": {"productive": 2, "loud": 31, "empty": 1},
  "menu": [
    {"id": "s1", "name": "Cocktails", "items": [
      {"id": "m1", "name": "Arrack sour", "price": 1400, "desc": "Ceylon arrack, lime, kithul, egg white", "icon": "Martini"},
      {"id": "m2", "name": "Bay sunset spritz", "price": 1600, "desc": "Passionfruit, prosecco, soda", "icon": "Wine"}
    ]},
    {"id": "s2", "name": "Bites", "items": [
      {"id": "m3", "name": "Hot butter cuttlefish", "price": 1750, "desc": "The rooftop classic, extra lime", "icon": "Flame"},
      {"id": "m4", "name": "Isso vade plate", "price": 950, "desc": "Prawn lentil fritters, tamarind dip", "icon": "Cookie"}
    ]}
  ]
}`

const placeClaystudioData = `{
  "type": "Pottery Workshop", "icon": "Palette", "color": "clay",
  "x": 19, "y": 34, "rating": 4.9, "reviews": 96, "distance": "300 m", "priceLevel": "$$",
  "wifi": {"score": 5.0, "speed": "18 Mbps", "label": "Okay"},
  "hours": "9:00 AM – 6:00 PM", "openNow": true, "bestTimes": ["morning", "afternoon"],
  "indoor": true, "artisan": true,
  "deal": {"active": true, "label": "Drop-in wheel-throwing taster — Rs 1,500", "tag": "Workshop"},
  "tags": ["Pottery", "Hands-on", "Indoor", "Local artisan"],
  "blurb": "A working clay studio tucked off the main road. Spin the wheel with resident potters — the perfect rainy-day plan when the surf is blown out.",
  "seaView": false, "photo": "clay",
  "vibes": {"productive": 12, "loud": 2, "empty": 9}
}`

const placeBatikData = `{
  "type": "Batik Atelier", "icon": "Brush", "color": "plum",
  "x": 73, "y": 30, "rating": 4.8, "reviews": 73, "distance": "750 m", "priceLevel": "$$",
  "wifi": {"score": 5.5, "speed": "20 Mbps", "label": "Okay"},
  "hours": "9:30 AM – 5:30 PM", "openNow": true, "bestTimes": ["morning", "afternoon"],
  "indoor": true, "artisan": true,
  "deal": {"active": false, "label": "", "tag": ""},
  "tags": ["Batik", "Hand-dyed", "Indoor", "Local artisan"],
  "blurb": "Family atelier hand-waxing and dyeing batik the traditional way. Watch a panel come to life, or wax your own under the canopy.",
  "seaView": false, "photo": "batik",
  "vibes": {"productive": 7, "loud": 1, "empty": 11}
}`

// driftwoodProfile is PROFILE from vendor-data.jsx.
const driftwoodProfile = `{
  "wifi": {"enabled": true, "score": 9.2, "speed": 88},
  "bestTimes": ["morning", "afternoon"],
  "seaView": true,
  "hours": {"open": "07:00", "close": "21:00"},
  "amenities": ["Power outlets", "Standing desks", "Air-con loft", "Specialty coffee", "Quiet zone", "Sea-view deck"],
  "blurb": "Airy garden café with a dedicated co-work loft, standing desks and the fastest fibre line in the bay. Sea-view seats on the upper deck."
}`

// driftwoodMenu is MENU from vendor-data.jsx (sections document).
const driftwoodMenu = `{"sections": [
  {"id": "s1", "name": "Breakfast", "items": [
    {"id": "m1", "name": "Coconut roti & curd bowl", "price": 950, "desc": "Pol roti, kithul treacle, buffalo curd", "img": null, "icon": "EggFried"},
    {"id": "m2", "name": "Surfer’s big plate", "price": 1450, "desc": "Eggs any way, sourdough, sambol, bacon", "img": null, "icon": "Sandwich"}
  ]},
  {"id": "s2", "name": "Mains", "items": [
    {"id": "m3", "name": "Jackfruit curry & rice", "price": 1200, "desc": "Young jak, red rice, papadam", "img": null, "icon": "Soup"},
    {"id": "m4", "name": "Grilled catch of the day", "price": 1850, "desc": "From the bay — ask what came in", "img": null, "icon": "Fish"}
  ]},
  {"id": "s3", "name": "Drinks", "items": [
    {"id": "m5", "name": "Single-origin pour-over", "price": 750, "desc": "Ceylon arabica, brewed to order", "img": null, "icon": "Coffee"},
    {"id": "m6", "name": "King coconut", "price": 350, "desc": "Straight off the cart, chilled", "img": null, "icon": "GlassWater"}
  ]}
]}`

// driftwoodDashboard bundles KPIS/WEEK/POPULAR/TREND from vendor-data.jsx.
const driftwoodDashboard = `{
  "kpis": [
    {"id": "offers", "label": "Active Offers", "value": 3, "icon": "Tag", "delta": null, "tone": "teal", "hint": "1 ending soon"},
    {"id": "redeem", "label": "Total Redemptions", "value": 248, "icon": "Ticket", "delta": 18, "tone": "coral", "hint": "this week"},
    {"id": "views", "label": "Profile Views", "value": 1940, "icon": "Eye", "delta": 12, "tone": "sun", "hint": "last 7 days"}
  ],
  "week": [
    {"day": "Mon", "views": 210, "redemptions": 24}, {"day": "Tue", "views": 248, "redemptions": 31},
    {"day": "Wed", "views": 196, "redemptions": 22}, {"day": "Thu", "views": 312, "redemptions": 44},
    {"day": "Fri", "views": 358, "redemptions": 52}, {"day": "Sat", "views": 401, "redemptions": 61},
    {"day": "Sun", "views": 285, "redemptions": 14}
  ],
  "popular": [
    {"h": 7, "v": 28}, {"h": 8, "v": 52}, {"h": 9, "v": 86}, {"h": 10, "v": 100},
    {"h": 11, "v": 78}, {"h": 12, "v": 60}, {"h": 13, "v": 48}, {"h": 14, "v": 66},
    {"h": 15, "v": 82}, {"h": 16, "v": 71}, {"h": 17, "v": 44}, {"h": 18, "v": 30},
    {"h": 19, "v": 22}, {"h": 20, "v": 14}
  ],
  "trend": [
    {"wk": "W1", "you": 38, "cat": 30}, {"wk": "W2", "you": 44, "cat": 33},
    {"wk": "W3", "you": 41, "cat": 36}, {"wk": "W4", "you": 58, "cat": 39},
    {"wk": "W5", "you": 72, "cat": 42}, {"wk": "W6", "you": 86, "cat": 47}
  ]
}`

// driftwoodGrowth bundles FLASH/STAMP_CARD/BTS_CLIPS/BOOST from vendor-data.jsx.
const driftwoodGrowth = `{
  "flash": {"radiusKm": 3, "inRange": 142, "nomads": 58, "surfers": 44, "lastSentMins": 184},
  "stampCard": {
    "reward": "Free flat white", "stampsNeeded": 6,
    "buckets": [
      {"away": 1, "count": 23, "label": "One stamp away"},
      {"away": 2, "count": 41, "label": "Two away"},
      {"away": 3, "count": 67, "label": "Three+ away"}
    ],
    "totalHolders": 214, "redeemedThisMonth": 38
  },
  "btsClips": [
    {"id": "b1", "title": "Pulling the morning espresso", "secs": 9, "views": 1240, "grad": "linear-gradient(135deg,#0e8c8a,#0a5f6e)", "icon": "Coffee"},
    {"id": "b2", "title": "Roasting Ceylon beans", "secs": 10, "views": 880, "grad": "linear-gradient(135deg,#c2683f,#7a3b22)", "icon": "Flame"},
    {"id": "b3", "title": "Latte art, slow-mo", "secs": 7, "views": 2010, "grad": "linear-gradient(135deg,#0b4a54,#0b6f86)", "icon": "Sparkles"}
  ],
  "boost": {
    "event": {"name": "Friday Sunset Sessions", "when": "Fri · 5–9 PM", "dj": "Live DJ + arrack cocktails"},
    "tiers": [
      {"id": "local", "label": "Local map top spot", "reach": "~1,200 travelers", "days": 3, "price": 12, "icon": "MapPin"},
      {"id": "category", "label": "Category feature", "reach": "~3,400 travelers", "days": 5, "price": 24, "icon": "Star", "best": true},
      {"id": "push", "label": "Push + map + category", "reach": "~6,800 travelers", "days": 7, "price": 39, "icon": "Megaphone"}
    ]
  }
}`

// driftwoodReports reuses the dashboard series for the Reports page charts.
const driftwoodReports = `{
  "week": [
    {"day": "Mon", "views": 210, "redemptions": 24}, {"day": "Tue", "views": 248, "redemptions": 31},
    {"day": "Wed", "views": 196, "redemptions": 22}, {"day": "Thu", "views": 312, "redemptions": 44},
    {"day": "Fri", "views": 358, "redemptions": 52}, {"day": "Sat", "views": 401, "redemptions": 61},
    {"day": "Sun", "views": 285, "redemptions": 14}
  ],
  "trend": [
    {"wk": "W1", "you": 38, "cat": 30}, {"wk": "W2", "you": 44, "cat": 33},
    {"wk": "W3", "you": 41, "cat": 36}, {"wk": "W4", "you": 58, "cat": 39},
    {"wk": "W5", "you": 72, "cat": 42}, {"wk": "W6", "you": 86, "cat": 47}
  ],
  "popular": [
    {"h": 7, "v": 28}, {"h": 8, "v": 52}, {"h": 9, "v": 86}, {"h": 10, "v": 100},
    {"h": 11, "v": 78}, {"h": 12, "v": 60}, {"h": 13, "v": 48}, {"h": 14, "v": 66},
    {"h": 15, "v": 82}, {"h": 16, "v": 71}, {"h": 17, "v": 44}, {"h": 18, "v": 30},
    {"h": 19, "v": 22}, {"h": 20, "v": 14}
  ]
}`
