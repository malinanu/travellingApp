// Trip planner models, catalog and distance helpers. Ported from
// Prototype/design/trip-data.jsx. `tripCatalog` is filled from real /places
// data by PlacesNotifier (mapped via TripCatalogItem.fromPlace).

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

/// A saved social-inspiration link attached to a trip.
/// Supports Instagram, TikTok, YouTube, and any generic website URL.
class InspirationLink {
  final String id;
  final String url;
  final String title; // user-entered label (auto-extracted from hostname on add)
  final String platform; // 'instagram' | 'tiktok' | 'youtube' | 'web'
  final String note; // optional personal note
  final String addedAt; // ISO-8601
  const InspirationLink({
    required this.id,
    required this.url,
    required this.title,
    required this.platform,
    this.note = '',
    required this.addedAt,
  });

  InspirationLink copyWith({String? note, String? title}) => InspirationLink(
        id: id,
        url: url,
        title: title ?? this.title,
        platform: platform,
        note: note ?? this.note,
        addedAt: addedAt,
      );

  Map<String, dynamic> toJson() => {
        'id': id,
        'url': url,
        'title': title,
        'platform': platform,
        'note': note,
        'addedAt': addedAt,
      };

  factory InspirationLink.fromJson(Map<String, dynamic> json) => InspirationLink(
        id: json['id'] as String,
        url: json['url'] as String,
        title: json['title'] as String? ?? '',
        platform: json['platform'] as String? ?? 'web',
        note: json['note'] as String? ?? '',
        addedAt: json['addedAt'] as String? ?? '',
      );

  /// Infer the platform from the URL hostname.
  static String platformOf(String url) {
    final lower = url.toLowerCase();
    if (lower.contains('instagram.com')) return 'instagram';
    if (lower.contains('tiktok.com')) return 'tiktok';
    if (lower.contains('youtube.com') || lower.contains('youtu.be')) return 'youtube';
    return 'web';
  }

  /// Extract a display title from the URL hostname.
  static String titleOf(String url) {
    try {
      final uri = Uri.parse(url);
      final host = uri.host.replaceFirst('www.', '');
      if (host.contains('instagram.com')) return 'Instagram post';
      if (host.contains('tiktok.com')) return 'TikTok video';
      if (host.contains('youtube.com') || host.contains('youtu.be')) return 'YouTube video';
      return host.isNotEmpty ? host : url;
    } catch (_) {
      return url;
    }
  }
}

class Trip {
  final String name;
  final String code;
  final String startDate; // yyyy-MM-dd
  final List<String> collaborators;
  final List<TripDay> days;
  final List<InspirationLink> inspirations;
  const Trip({
    required this.name,
    required this.code,
    required this.startDate,
    required this.collaborators,
    required this.days,
    this.inspirations = const [],
  });

  Trip copyWith({
    String? name,
    String? startDate,
    List<String>? collaborators,
    List<TripDay>? days,
    List<InspirationLink>? inspirations,
  }) =>
      Trip(
        name: name ?? this.name,
        code: code,
        startDate: startDate ?? this.startDate,
        collaborators: collaborators ?? this.collaborators,
        days: days ?? this.days,
        inspirations: inspirations ?? this.inspirations,
      );

  Map<String, dynamic> toJson() => {
        'name': name,
        'code': code,
        'startDate': startDate,
        'collaborators': collaborators,
        'days': days.map((d) => d.toJson()).toList(),
        'inspirations': inspirations.map((l) => l.toJson()).toList(),
      };
  factory Trip.fromJson(Map<String, dynamic> json) => Trip(
        name: json['name'] as String,
        code: json['code'] as String,
        startDate: json['startDate'] as String,
        collaborators: (json['collaborators'] as List).cast<String>(),
        days: (json['days'] as List).map((d) => TripDay.fromJson(d as Map<String, dynamic>)).toList(),
        inspirations: json['inspirations'] == null
            ? const []
            : (json['inspirations'] as List)
                .map((l) => InspirationLink.fromJson(l as Map<String, dynamic>))
                .toList(),
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

/// Places the trip planner can add as stops. Starts empty and is filled
/// from real /places data by PlacesNotifier (via TripCatalogItem.fromPlace).
var tripCatalog = <TripCatalogItem>[];

/// The sample trip older app builds gave every new user (code WLG-7QF).
/// Its stops pointed at made-up places, so [cleanLegacySampleTrip] strips
/// them from any copy that's still saved on a phone.
const _legacySampleTripCode = 'WLG-7QF';
const _legacySampleStopIds = {'s1', 's2', 's3', 's4', 's5', 's6', 's7'};
const _legacySampleCollaborators = {'theo', 'mia'};

/// Removes the old sample trip's made-up content (its name, fake
/// collaborators and sample stops) and keeps anything the user added.
/// Other trips come back unchanged.
Trip cleanLegacySampleTrip(Trip t) {
  if (t.code != _legacySampleTripCode) return t;
  return t.copyWith(
    name: t.name == 'Weligama Weekend' ? 'Spotto Trip' : t.name,
    collaborators: t.collaborators.where((c) => !_legacySampleCollaborators.contains(c)).toList(),
    days: [
      for (final d in t.days)
        d.copyWith(
          stops: d.stops.where((s) => s.custom != null || !_legacySampleStopIds.contains(s.id)).toList(),
        ),
    ],
  );
}

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

/// `GRP-` + 6 random base36 chars for a new travel group. The live-location
/// socket is keyed by this code, so it must not repeat across groups (the
/// server also swaps out a code that's already taken).
String newGroupCode() {
  const chars = '0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ';
  final r = math.Random.secure();
  return 'GRP-${List.generate(6, (_) => chars[r.nextInt(chars.length)]).join()}';
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
