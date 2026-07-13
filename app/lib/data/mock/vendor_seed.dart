// Vendor portal demo data for "Driftwood Coffee & Co-Work".
// 1:1 port of Prototype/design/vendor-data.jsx (models + seed together —
// these types are only used by the vendor features).

import '../models/models.dart' show MenuSection, MenuItem;

class VendorBusiness {
  final String name;
  final String type;
  final String typeId; // one of businessTypes ids
  final String area;
  final String initials;
  final String owner;
  const VendorBusiness({
    required this.name,
    required this.type,
    required this.typeId,
    required this.area,
    required this.initials,
    required this.owner,
  });

  VendorBusiness copyWith({String? name, String? type, String? typeId, String? area, String? initials, String? owner}) =>
      VendorBusiness(
        name: name ?? this.name,
        type: type ?? this.type,
        typeId: typeId ?? this.typeId,
        area: area ?? this.area,
        initials: initials ?? this.initials,
        owner: owner ?? this.owner,
      );

  Map<String, dynamic> toJson() =>
      {'name': name, 'type': type, 'typeId': typeId, 'area': area, 'initials': initials, 'owner': owner};
  factory VendorBusiness.fromJson(Map<String, dynamic> json) => VendorBusiness(
        name: json['name'] as String,
        type: json['type'] as String,
        typeId: json['typeId'] as String,
        area: json['area'] as String,
        initials: json['initials'] as String,
        owner: json['owner'] as String,
      );
}

class BusinessType {
  final String id;
  final String label;
  final String icon;
  final bool food; // unlocks the Menu tab
  const BusinessType({required this.id, required this.label, required this.icon, this.food = false});
}

class Kpi {
  final String id;
  final String label;
  final int value;
  final String icon;
  final int? delta;
  final String tone; // teal/coral/sun
  final String hint;
  const Kpi({
    required this.id,
    required this.label,
    required this.value,
    required this.icon,
    this.delta,
    required this.tone,
    required this.hint,
  });
}

class Offer {
  final String id;
  final String title;
  final String type;
  final String value;
  final String status; // live/ending/draft/ended
  final String start;
  final String end;
  final int claimed;
  final int limit;
  final int redeemed;
  const Offer({
    required this.id,
    required this.title,
    required this.type,
    required this.value,
    required this.status,
    required this.start,
    required this.end,
    required this.claimed,
    required this.limit,
    required this.redeemed,
  });

  Offer copyWith({String? title, String? type, String? value, String? status, String? start, String? end, int? claimed, int? limit, int? redeemed}) => Offer(
        id: id,
        title: title ?? this.title,
        type: type ?? this.type,
        value: value ?? this.value,
        status: status ?? this.status,
        start: start ?? this.start,
        end: end ?? this.end,
        claimed: claimed ?? this.claimed,
        limit: limit ?? this.limit,
        redeemed: redeemed ?? this.redeemed,
      );
}

class DiscountType {
  final String id;
  final String label;
  final String icon;
  final String hint;
  const DiscountType({required this.id, required this.label, required this.icon, required this.hint});
}

class DayTraffic {
  final String day;
  final int views;
  final int redemptions;
  const DayTraffic({required this.day, required this.views, required this.redemptions});
}

class HourFootfall {
  final int h;
  final int v; // relative footfall 0-100
  const HourFootfall({required this.h, required this.v});
}

class WeekTrend {
  final String wk;
  final int you;
  final int cat;
  const WeekTrend({required this.wk, required this.you, required this.cat});
}

class Redemption {
  final String id;
  final String code;
  final String name;
  final String offer;
  final String time;
  final String style;
  const Redemption({
    required this.id,
    required this.code,
    required this.name,
    required this.offer,
    required this.time,
    required this.style,
  });
}

class ValidCode {
  final String name;
  final String country;
  final String style;
  final String offer;
  final String value;
  const ValidCode({
    required this.name,
    required this.country,
    required this.style,
    required this.offer,
    required this.value,
  });
}

class VendorProfile {
  final bool wifiEnabled;
  final double wifiScore;
  final int wifiSpeed;
  final List<String> bestTimes;
  final bool seaView;
  final String opens;
  final String closes;
  final List<String> amenities;
  final String blurb;
  const VendorProfile({
    required this.wifiEnabled,
    required this.wifiScore,
    required this.wifiSpeed,
    required this.bestTimes,
    required this.seaView,
    required this.opens,
    required this.closes,
    required this.amenities,
    required this.blurb,
  });

  VendorProfile copyWith({
    bool? wifiEnabled,
    double? wifiScore,
    int? wifiSpeed,
    List<String>? bestTimes,
    bool? seaView,
    String? opens,
    String? closes,
    List<String>? amenities,
    String? blurb,
  }) =>
      VendorProfile(
        wifiEnabled: wifiEnabled ?? this.wifiEnabled,
        wifiScore: wifiScore ?? this.wifiScore,
        wifiSpeed: wifiSpeed ?? this.wifiSpeed,
        bestTimes: bestTimes ?? this.bestTimes,
        seaView: seaView ?? this.seaView,
        opens: opens ?? this.opens,
        closes: closes ?? this.closes,
        amenities: amenities ?? this.amenities,
        blurb: blurb ?? this.blurb,
      );
}

// ── seed data ──

const vendorSeed = VendorBusiness(
  name: 'Driftwood Coffee & Co-Work',
  type: 'Coworking Café',
  typeId: 'cafe',
  area: 'Weligama',
  initials: 'DC',
  owner: 'Tharindu Perera',
);

const businessTypes = [
  BusinessType(id: 'restaurant', label: 'Restaurant', icon: 'Utensils', food: true),
  BusinessType(id: 'cafe', label: 'Café', icon: 'Coffee', food: true),
  BusinessType(id: 'bar', label: 'Bar & Sunset', icon: 'Martini', food: true),
  BusinessType(id: 'surf', label: 'Surf & Water', icon: 'Waves'),
  BusinessType(id: 'craft', label: 'Craft & Studio', icon: 'Palette'),
  BusinessType(id: 'shop', label: 'Shop', icon: 'Store'),
];

const vendorMenuSeed = [
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
];

const kpis = [
  Kpi(id: 'offers', label: 'Active Offers', value: 3, icon: 'Tag', tone: 'teal', hint: '1 ending soon'),
  Kpi(id: 'redeem', label: 'Total Redemptions', value: 248, icon: 'Ticket', delta: 18, tone: 'coral', hint: 'this week'),
  Kpi(id: 'views', label: 'Profile Views', value: 1940, icon: 'Eye', delta: 12, tone: 'sun', hint: 'last 7 days'),
];

const offersSeed = [
  Offer(id: 'o1', title: 'Free flat white with any day pass', type: 'Freebie', value: 'Free coffee', status: 'live', start: 'Today 07:00', end: 'Today 17:00', claimed: 34, limit: 50, redeemed: 21),
  Offer(id: 'o2', title: '20% off all single-origin pour-overs', type: 'Percentage', value: '20% off', status: 'live', start: 'Mon', end: 'Sun', claimed: 88, limit: 200, redeemed: 61),
  Offer(id: 'o3', title: 'Buy 5 day-passes, get 1 free', type: 'Bundle', value: '6-for-5', status: 'ending', start: 'This week', end: 'Today 23:59', claimed: 47, limit: 50, redeemed: 39),
  Offer(id: 'o4', title: 'Rainy-day bundle: coffee + covered desk', type: 'Bundle', value: '2-for-1', status: 'draft', start: '—', end: '—', claimed: 0, limit: 40, redeemed: 0),
  Offer(id: 'o5', title: 'Full-moon night: free ginger tea', type: 'Freebie', value: 'Free tea', status: 'ended', start: 'Last Fri', end: 'Last Sat', claimed: 50, limit: 50, redeemed: 44),
];

const discountTypes = [
  DiscountType(id: 'pct', label: 'Percentage off', icon: 'Percent', hint: 'e.g. 20% off'),
  DiscountType(id: 'amount', label: 'Amount off', icon: 'BadgeDollarSign', hint: 'e.g. Rs 500 off'),
  DiscountType(id: 'freebie', label: 'Freebie', icon: 'Gift', hint: 'e.g. free coffee'),
  DiscountType(id: 'bundle', label: 'Bundle', icon: 'Package', hint: 'e.g. 2-for-1'),
];

const weekTraffic = [
  DayTraffic(day: 'Mon', views: 210, redemptions: 24),
  DayTraffic(day: 'Tue', views: 248, redemptions: 31),
  DayTraffic(day: 'Wed', views: 196, redemptions: 22),
  DayTraffic(day: 'Thu', views: 312, redemptions: 44),
  DayTraffic(day: 'Fri', views: 358, redemptions: 52),
  DayTraffic(day: 'Sat', views: 401, redemptions: 61),
  DayTraffic(day: 'Sun', views: 285, redemptions: 14),
];

const popularTimes = [
  HourFootfall(h: 7, v: 28), HourFootfall(h: 8, v: 52), HourFootfall(h: 9, v: 86),
  HourFootfall(h: 10, v: 100), HourFootfall(h: 11, v: 78), HourFootfall(h: 12, v: 60),
  HourFootfall(h: 13, v: 48), HourFootfall(h: 14, v: 66), HourFootfall(h: 15, v: 82),
  HourFootfall(h: 16, v: 71), HourFootfall(h: 17, v: 44), HourFootfall(h: 18, v: 30),
  HourFootfall(h: 19, v: 22), HourFootfall(h: 20, v: 14),
];

const trend = [
  WeekTrend(wk: 'W1', you: 38, cat: 30),
  WeekTrend(wk: 'W2', you: 44, cat: 33),
  WeekTrend(wk: 'W3', you: 41, cat: 36),
  WeekTrend(wk: 'W4', you: 58, cat: 39),
  WeekTrend(wk: 'W5', you: 72, cat: 42),
  WeekTrend(wk: 'W6', you: 86, cat: 47),
];

const redemptionsSeed = [
  Redemption(id: 'r1', code: 'SPOTTO-DRIF', name: 'Lena (Germany)', offer: 'Free flat white', time: '2 min ago', style: 'Digital Nomad'),
  Redemption(id: 'r2', code: 'SPOTTO-9KQ2', name: 'Marco (Italy)', offer: '20% off pour-over', time: '18 min ago', style: 'Surf Traveler'),
  Redemption(id: 'r3', code: 'SPOTTO-7HX1', name: 'Aiko (Japan)', offer: 'Free flat white', time: '41 min ago', style: 'Digital Nomad'),
  Redemption(id: 'r4', code: 'SPOTTO-2BVT', name: 'Sam (Australia)', offer: 'Day-pass bundle', time: '1 hr ago', style: 'Backpacker'),
  Redemption(id: 'r5', code: 'SPOTTO-MR48', name: 'Chloé (France)', offer: '20% off pour-over', time: '2 hr ago', style: 'Luxury'),
];

const validCodes = {
  'SPOTTO-X4K9': ValidCode(name: 'Tom R.', country: 'UK', style: 'Surf Traveler', offer: '20% off all single-origin pour-overs', value: '20% off'),
  'SPOTTO-88Q2': ValidCode(name: 'Sara P.', country: 'Spain', style: 'Digital Nomad', offer: 'Buy 5 day-passes, get 1 free', value: '6-for-5'),
};

// SPOTTO-DRIF is seeded in the prototype's VALID_CODES too, but in the Flutter
// app the traveller can genuinely claim Driftwood's deal, so it validates via
// the claims store instead. Kept here for parity when no claim exists.
const validCodesLegacy = {
  'SPOTTO-DRIF': ValidCode(name: 'Lena M.', country: 'Germany', style: 'Digital Nomad', offer: 'Free flat white with any day pass', value: 'Free coffee'),
};

const vendorProfileSeed = VendorProfile(
  wifiEnabled: true,
  wifiScore: 9.2,
  wifiSpeed: 88,
  bestTimes: ['morning', 'afternoon'],
  seaView: true,
  opens: '07:00',
  closes: '21:00',
  amenities: ['Power outlets', 'Standing desks', 'Air-con loft', 'Specialty coffee', 'Quiet zone', 'Sea-view deck'],
  blurb: 'Airy garden café with a dedicated co-work loft, standing desks and the fastest fibre line in the bay. Sea-view seats on the upper deck.',
);

/// Bell-dropdown notification (session-only, like the prototype).
class VendorNotif {
  final String id;
  final String icon;
  final String tone; // accent/sun/muted
  final String title;
  final String sub;
  final String time;
  final bool unread;
  const VendorNotif({
    required this.id,
    required this.icon,
    required this.tone,
    required this.title,
    required this.sub,
    required this.time,
    required this.unread,
  });

  VendorNotif copyWith({bool? unread}) => VendorNotif(
      id: id, icon: icon, tone: tone, title: title, sub: sub, time: time, unread: unread ?? this.unread);
}

const vendorNotifsSeed = [
  VendorNotif(id: 'n1', icon: 'Ticket', tone: 'accent', title: 'Lena redeemed "Free flat white"', sub: 'Scanned at the counter', time: '2 min ago', unread: true),
  VendorNotif(id: 'n2', icon: 'Star', tone: 'sun', title: 'New 5-star review from Marco', sub: '"Best pour-over on the south coast"', time: '38 min ago', unread: true),
  VendorNotif(id: 'n3', icon: 'Clock', tone: 'sun', title: '"Buy 5 day-passes" ends tonight', sub: '47 of 50 claimed — almost gone', time: '2 hr ago', unread: true),
  VendorNotif(id: 'n4', icon: 'TrendingUp', tone: 'accent', title: 'Profile views up 12% this week', sub: 'Your evening photos are working', time: 'Yesterday', unread: false),
  VendorNotif(id: 'n5', icon: 'Wallet', tone: 'muted', title: 'Premium plan renewed', sub: '\$24.00 · receipt emailed', time: '3 days ago', unread: false),
];

// ── Growth Tools data ──

class FlashSaleInfo {
  final int radiusKm;
  final int inRange;
  final int nomads;
  final int surfers;
  final int lastSentMins;
  const FlashSaleInfo({required this.radiusKm, required this.inRange, required this.nomads, required this.surfers, required this.lastSentMins});
}

const flash = FlashSaleInfo(radiusKm: 3, inRange: 142, nomads: 58, surfers: 44, lastSentMins: 184);

class StampBucket {
  final int away;
  final int count;
  final String label;
  const StampBucket({required this.away, required this.count, required this.label});
}

class StampCardInfo {
  final String reward;
  final int stampsNeeded;
  final List<StampBucket> buckets;
  final int totalHolders;
  final int redeemedThisMonth;
  const StampCardInfo({required this.reward, required this.stampsNeeded, required this.buckets, required this.totalHolders, required this.redeemedThisMonth});
}

const stampCard = StampCardInfo(
  reward: 'Free flat white',
  stampsNeeded: 6,
  buckets: [
    StampBucket(away: 1, count: 23, label: 'One stamp away'),
    StampBucket(away: 2, count: 41, label: 'Two away'),
    StampBucket(away: 3, count: 67, label: 'Three+ away'),
  ],
  totalHolders: 214,
  redeemedThisMonth: 38,
);

class BtsClip {
  final String id;
  final String title;
  final int secs;
  final int views;
  final int gradStart; // ARGB — prototype used CSS gradients
  final int gradEnd;
  final String icon;
  const BtsClip({required this.id, required this.title, required this.secs, required this.views, required this.gradStart, required this.gradEnd, required this.icon});
}

const btsClips = [
  BtsClip(id: 'b1', title: 'Pulling the morning espresso', secs: 9, views: 1240, gradStart: 0xFF0E8C8A, gradEnd: 0xFF0A5F6E, icon: 'Coffee'),
  BtsClip(id: 'b2', title: 'Roasting Ceylon beans', secs: 10, views: 880, gradStart: 0xFFC2683F, gradEnd: 0xFF7A3B22, icon: 'Flame'),
  BtsClip(id: 'b3', title: 'Latte art, slow-mo', secs: 7, views: 2010, gradStart: 0xFF0B4A54, gradEnd: 0xFF0B6F86, icon: 'Sparkles'),
];

class BoostTier {
  final String id;
  final String label;
  final String reach;
  final int days;
  final int price; // USD
  final String icon;
  final bool best;
  const BoostTier({required this.id, required this.label, required this.reach, required this.days, required this.price, required this.icon, this.best = false});
}

class BoostEvent {
  final String name;
  final String when;
  final String dj;
  const BoostEvent({required this.name, required this.when, required this.dj});
}

const boostEvent = BoostEvent(name: 'Friday Sunset Sessions', when: 'Fri · 5–9 PM', dj: 'Live DJ + arrack cocktails');

const boostTiers = [
  BoostTier(id: 'local', label: 'Local map top spot', reach: '~1,200 travelers', days: 3, price: 12, icon: 'MapPin'),
  BoostTier(id: 'category', label: 'Category feature', reach: '~3,400 travelers', days: 5, price: 24, icon: 'Star', best: true),
  BoostTier(id: 'push', label: 'Push + map + category', reach: '~6,800 travelers', days: 7, price: 39, icon: 'Megaphone'),
];
