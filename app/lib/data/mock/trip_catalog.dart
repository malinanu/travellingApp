// Trip planner catalog + seed trip + distance helpers.
// 1:1 port of Prototype/design/trip-data.jsx. `tripCatalog` below starts as
// this hardcoded offline fallback, then PlacesNotifier overwrites it with
// real /places data (mapped via TripCatalogItem.fromPlace) once fetched —
// same offline-first pattern as seed.places/tripSeed/groupSeed elsewhere.

import 'dart:math' as math;

import 'package:intl/intl.dart';

import '../models/models.dart' show Place;

class TripCatalogItem {
  final String id;
  final String? spottoId; // links to a Place for opening the business drawer
  final String name;
  final String type;
  final String icon;
  final double lat;
  final double lng;
  final int cost; // Rs
  final List<String> styles;
  final String hint; // suggested time "HH:mm"
  const TripCatalogItem({
    required this.id,
    this.spottoId,
    required this.name,
    required this.type,
    required this.icon,
    required this.lat,
    required this.lng,
    required this.cost,
    required this.styles,
    required this.hint,
  });

  /// Maps a real /places entry onto the trip-planner shape. `spottoId` is
  /// only set for `curated` places (the original Spotto businesses with a
  /// full menu/wifi/deal/reviews) — plain points of interest don't get a
  /// business-drawer link.
  factory TripCatalogItem.fromPlace(Place p) => TripCatalogItem(
        id: p.id,
        spottoId: p.curated ? p.id : null,
        name: p.name,
        type: p.type,
        icon: p.icon,
        lat: p.lat,
        lng: p.lng,
        cost: p.cost,
        styles: p.styles,
        hint: p.hint,
      );
}

/// A one-off stop typed in by the user (not in the catalog).
class CustomStopInfo {
  final String name;
  final String type;
  final String icon;
  final double lat;
  final double lng;
  final int cost;
  const CustomStopInfo({
    required this.name,
    this.type = 'Custom stop',
    this.icon = 'MapPin',
    this.lat = 5.9749,
    this.lng = 80.4293,
    this.cost = 0,
  });

  Map<String, dynamic> toJson() =>
      {'name': name, 'type': type, 'icon': icon, 'lat': lat, 'lng': lng, 'cost': cost};
  factory CustomStopInfo.fromJson(Map<String, dynamic> json) => CustomStopInfo(
        name: json['name'] as String,
        type: json['type'] as String? ?? 'Custom stop',
        icon: json['icon'] as String? ?? 'MapPin',
        lat: (json['lat'] as num?)?.toDouble() ?? 5.9749,
        lng: (json['lng'] as num?)?.toDouble() ?? 80.4293,
        cost: json['cost'] as int? ?? 0,
      );
}

class TripStop {
  final String id;
  final String placeId; // '' for custom stops
  final int time; // minutes since midnight
  final String? by; // collaborator who added it
  final CustomStopInfo? custom;
  const TripStop({required this.id, required this.placeId, required this.time, this.by, this.custom});

  TripStop copyWith({String? id, String? placeId, int? time, String? by}) => TripStop(
      id: id ?? this.id,
      placeId: placeId ?? this.placeId,
      time: time ?? this.time,
      by: by ?? this.by,
      custom: custom);

  Map<String, dynamic> toJson() => {
        'id': id,
        'placeId': placeId,
        'time': time,
        if (by != null) 'by': by,
        if (custom != null) 'custom': custom!.toJson(),
      };
  factory TripStop.fromJson(Map<String, dynamic> json) => TripStop(
        id: json['id'] as String,
        placeId: json['placeId'] as String? ?? '',
        time: json['time'] as int,
        by: json['by'] as String?,
        custom: json['custom'] == null
            ? null
            : CustomStopInfo.fromJson(json['custom'] as Map<String, dynamic>),
      );
}

class TripDay {
  final String id;
  final List<TripStop> stops;
  const TripDay({required this.id, required this.stops});

  TripDay copyWith({List<TripStop>? stops}) => TripDay(id: id, stops: stops ?? this.stops);

  Map<String, dynamic> toJson() => {'id': id, 'stops': stops.map((s) => s.toJson()).toList()};
  factory TripDay.fromJson(Map<String, dynamic> json) => TripDay(
        id: json['id'] as String,
        stops: (json['stops'] as List).map((s) => TripStop.fromJson(s as Map<String, dynamic>)).toList(),
      );
}

class Trip {
  final String name;
  final String code;
  final String startDate; // yyyy-MM-dd
  final List<String> collaborators;
  final List<TripDay> days;
  const Trip({
    required this.name,
    required this.code,
    required this.startDate,
    required this.collaborators,
    required this.days,
  });

  Trip copyWith({String? name, String? startDate, List<String>? collaborators, List<TripDay>? days}) => Trip(
        name: name ?? this.name,
        code: code,
        startDate: startDate ?? this.startDate,
        collaborators: collaborators ?? this.collaborators,
        days: days ?? this.days,
      );

  Map<String, dynamic> toJson() => {
        'name': name,
        'code': code,
        'startDate': startDate,
        'collaborators': collaborators,
        'days': days.map((d) => d.toJson()).toList(),
      };
  factory Trip.fromJson(Map<String, dynamic> json) => Trip(
        name: json['name'] as String,
        code: json['code'] as String,
        startDate: json['startDate'] as String,
        collaborators: (json['collaborators'] as List).cast<String>(),
        days: (json['days'] as List).map((d) => TripDay.fromJson(d as Map<String, dynamic>)).toList(),
      );
}

class Leg {
  final double km;
  final String dist;
  final int mins;
  final String mode; // 'Walk' | 'Tuk-tuk'
  final String icon; // 'Footprints' | 'CarTaxiFront'
  final String gmaps;
  const Leg({
    required this.km,
    required this.dist,
    required this.mins,
    required this.mode,
    required this.icon,
    required this.gmaps,
  });
}

var tripCatalog = [
  // Spotto curated (linked to PLACES so tapping opens the business drawer)
  TripCatalogItem(id: 'driftwood', spottoId: 'driftwood', name: 'Driftwood Coffee & Co-Work', type: 'Coworking Café', icon: 'Coffee', lat: 5.9762, lng: 80.4221, cost: 1200, styles: ['nomad'], hint: '8:00'),
  TripCatalogItem(id: 'baysurf', spottoId: 'baysurf', name: 'Weligama Bay Surf School', type: 'Surf School', icon: 'Waves', lat: 5.9716, lng: 80.4308, cost: 3500, styles: ['surf', 'backpacker'], hint: '6:30'),
  TripCatalogItem(id: 'hangtime', spottoId: 'hangtime', name: 'Hangtime Rooftop & Sunset Bar', type: 'Sunset Bar', icon: 'Martini', lat: 5.9789, lng: 80.4275, cost: 2800, styles: ['nomad', 'luxury'], hint: '17:30'),
  TripCatalogItem(id: 'claystudio', spottoId: 'claystudio', name: 'Clay & Co Pottery Studio', type: 'Pottery Workshop', icon: 'Palette', lat: 5.9773, lng: 80.4189, cost: 1500, styles: ['backpacker'], hint: '10:00'),
  TripCatalogItem(id: 'batik', spottoId: 'batik', name: 'Sea Salt Batik Atelier', type: 'Batik Atelier', icon: 'Brush', lat: 5.9747, lng: 80.4368, cost: 1000, styles: ['luxury'], hint: '11:00'),
  // wider Weligama area
  TripCatalogItem(id: 'bayvilla', name: 'Bay Villa Boutique Stay', type: 'Hotel', icon: 'BedDouble', lat: 5.9731, lng: 80.4254, cost: 9500, styles: ['luxury', 'nomad'], hint: '14:00'),
  TripCatalogItem(id: 'palmhostel', name: 'Palm & Board Hostel', type: 'Hostel', icon: 'Backpack', lat: 5.9758, lng: 80.4332, cost: 2200, styles: ['backpacker', 'surf'], hint: '13:00'),
  TripCatalogItem(id: 'welibeach', name: 'Weligama Beach Point', type: 'Beach', icon: 'Palmtree', lat: 5.9722, lng: 80.4286, cost: 0, styles: ['surf', 'backpacker', 'nomad', 'luxury'], hint: '7:00'),
  TripCatalogItem(id: 'mirissa', name: 'Mirissa Beach', type: 'Beach', icon: 'Palmtree', lat: 5.9445, lng: 80.4595, cost: 0, styles: ['surf', 'backpacker', 'luxury'], hint: '15:00'),
  TripCatalogItem(id: 'coconuthill', name: 'Coconut Tree Hill', type: 'Viewpoint', icon: 'Mountain', lat: 5.9417, lng: 80.4632, cost: 0, styles: ['backpacker', 'luxury', 'nomad'], hint: '17:00'),
  TripCatalogItem(id: 'kottu', name: 'Smoky Kottu Shack', type: 'Street Food', icon: 'UtensilsCrossed', lat: 5.9755, lng: 80.4312, cost: 800, styles: ['backpacker', 'surf'], hint: '19:30'),
  TripCatalogItem(id: 'reefdine', name: 'Reef & Vine Dining', type: 'Restaurant', icon: 'Wine', lat: 5.9740, lng: 80.4260, cost: 5200, styles: ['luxury'], hint: '19:00'),
  TripCatalogItem(id: 'secretroot', name: 'Secret Root Spa', type: 'Spa & Wellness', icon: 'Flower2', lat: 5.9768, lng: 80.4338, cost: 4200, styles: ['luxury', 'nomad'], hint: '16:00'),
  TripCatalogItem(id: 'taprobane', name: 'Taprobane Island Viewpoint', type: 'Viewpoint', icon: 'Mountain', lat: 5.9735, lng: 80.4203, cost: 0, styles: ['nomad', 'luxury', 'backpacker'], hint: '8:30'),
  TripCatalogItem(id: 'oldlight', name: 'Old Light Gallery Bar', type: 'Bar & Gallery', icon: 'Martini', lat: 5.9701, lng: 80.4179, cost: 2400, styles: ['nomad', 'backpacker'], hint: '21:00'),
  TripCatalogItem(id: 'morningmkt', name: 'Weligama Morning Market', type: 'Local Market', icon: 'ShoppingBasket', lat: 5.9769, lng: 80.4297, cost: 500, styles: ['backpacker', 'nomad'], hint: '6:30'),
];

const tripSeed = Trip(
  name: 'Weligama Weekend',
  code: 'WLG-7QF',
  startDate: '2026-06-20',
  collaborators: ['theo', 'mia'],
  days: [
    TripDay(id: 'd1', stops: [
      TripStop(id: 's1', placeId: 'bayvilla', time: 14 * 60), // 2:00 PM check-in
      TripStop(id: 's2', placeId: 'driftwood', time: 16 * 60, by: 'theo'),
      TripStop(id: 's3', placeId: 'hangtime', time: 17 * 60 + 30),
      TripStop(id: 's4', placeId: 'kottu', time: 19 * 60 + 30),
    ]),
    TripDay(id: 'd2', stops: [
      TripStop(id: 's5', placeId: 'welibeach', time: 7 * 60),
      TripStop(id: 's6', placeId: 'baysurf', time: 8 * 60, by: 'mia'),
      TripStop(id: 's7', placeId: 'coconuthill', time: 17 * 60),
    ]),
  ],
);

double tripHaversineKm(TripCatalogItem a, TripCatalogItem b) {
  const r = 6371.0;
  double rad(double d) => d * math.pi / 180;
  final dLat = rad(b.lat - a.lat), dLng = rad(b.lng - a.lng);
  final h = math.pow(math.sin(dLat / 2), 2) +
      math.cos(rad(a.lat)) * math.cos(rad(b.lat)) * math.pow(math.sin(dLng / 2), 2);
  return 2 * r * math.asin(math.sqrt(h));
}

/// Leg between two catalog places: walk if short, tuk-tuk otherwise.
Leg tripLeg(TripCatalogItem from, TripCatalogItem to) {
  final km = tripHaversineKm(from, to);
  final walk = km <= 1.2;
  final mins = math.max(1, (walk ? km * 14 : km * 3 + 4).round());
  final dist = km < 1 ? '${(km * 1000).round()} m' : '${km.toStringAsFixed(1)} km';
  return Leg(
    km: km,
    dist: dist,
    mins: mins,
    mode: walk ? 'Walk' : 'Tuk-tuk',
    icon: walk ? 'Footprints' : 'CarTaxiFront',
    gmaps: 'https://www.google.com/maps/dir/?api=1&origin=${from.lat},${from.lng}'
        '&destination=${to.lat},${to.lng}&travelmode=${walk ? 'walking' : 'driving'}',
  );
}

TripCatalogItem? tripPlace(TripStop stop) {
  final c = stop.custom;
  if (c != null) {
    return TripCatalogItem(
        id: stop.id,
        name: c.name,
        type: c.type,
        icon: c.icon,
        lat: c.lat,
        lng: c.lng,
        cost: c.cost,
        styles: const [],
        hint: '9:00');
  }
  for (final p in tripCatalog) {
    if (p.id == stop.placeId) return p;
  }
  return null;
}

/// Whole-day route in Google Maps (origin → waypoints → destination).
String? tripDayGmaps(List<TripStop> stops) {
  final pts = stops.map(tripPlace).whereType<TripCatalogItem>().toList();
  if (pts.length < 2) return null;
  final o = pts.first, d = pts.last;
  final way = pts.sublist(1, pts.length - 1).map((p) => '${p.lat},${p.lng}').join('|');
  return 'https://www.google.com/maps/dir/?api=1&origin=${o.lat},${o.lng}'
      '&destination=${d.lat},${d.lng}'
      '${way.isNotEmpty ? '&waypoints=${Uri.encodeComponent(way)}' : ''}&travelmode=driving';
}

String tripFmtTime(int mins) {
  var h = mins ~/ 60;
  final m = mins % 60;
  final ap = h >= 12 ? 'PM' : 'AM';
  h = h % 12;
  if (h == 0) h = 12;
  return '$h:${m.toString().padLeft(2, '0')} $ap';
}

String tripFmtRs(int n) {
  if (n == 0) return 'Free';
  return 'Rs ${NumberFormat('#,##0', 'en_US').format(n)}';
}

/// Date for day index [i], derived from the trip's start date.
DateTime tripDayDate(String startDate, int i) =>
    DateTime.parse(startDate).add(Duration(days: i));

/// Local (not UTC) yyyy-MM-dd so the picker's "today" matches the default.
String todayIso() {
  final d = DateTime.now();
  return '${d.year}-${d.month.toString().padLeft(2, '0')}-${d.day.toString().padLeft(2, '0')}';
}

/// `TRP-` + 3 random base36 chars, like the prototype.
String newTripCode() {
  const chars = '0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ';
  final r = math.Random();
  return 'TRP-${List.generate(3, (_) => chars[r.nextInt(chars.length)]).join()}';
}

int _tripUid = 100;

/// Unique id for new stops/days (prototype's `newId`).
String tripNewId(String prefix) =>
    '$prefix${++_tripUid}_${DateTime.now().millisecondsSinceEpoch % 10000}';

/// An invitable person shown on the trip-share sheet — built from a real
/// GroupMember (see trip_share_sheet.dart), not a hardcoded contacts list.
class TripContact {
  final String id;
  final String name;
  final String sub; // email, shown under the name
  final String initial;
  final int color; // ARGB
  const TripContact({
    required this.id,
    required this.name,
    required this.sub,
    required this.initial,
    required this.color,
  });
}
