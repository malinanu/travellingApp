// Core Spotto data models — ported from Prototype/design/data.jsx.
// Plain immutable classes; only persisted types get toJson/fromJson.

class WifiInfo {
  final double score;
  final String speed;
  final String label;
  const WifiInfo({required this.score, required this.speed, required this.label});

  factory WifiInfo.fromJson(Map<String, dynamic> json) => WifiInfo(
        score: (json['score'] as num?)?.toDouble() ?? 0,
        speed: json['speed'] as String? ?? '',
        label: json['label'] as String? ?? '',
      );
}

class Deal {
  final String? id;
  final bool active;
  final String? label;
  final String? tag;
  const Deal({this.id, required this.active, this.label, this.tag});
  static const none = Deal(active: false);

  factory Deal.fromJson(Map<String, dynamic> json) => Deal(
        id: json['id'] as String?,
        active: json['active'] as bool? ?? false,
        label: json['label'] as String?,
        tag: json['tag'] as String?,
      );
}

class MenuItem {
  final String id;
  final String name;
  final int price; // Rs
  final String desc;
  final String icon;
  const MenuItem({
    required this.id,
    required this.name,
    required this.price,
    required this.desc,
    required this.icon,
  });

  factory MenuItem.fromJson(Map<String, dynamic> json) => MenuItem(
        id: json['id'] as String? ?? '',
        name: json['name'] as String? ?? '',
        price: (json['price'] as num?)?.toInt() ?? 0,
        desc: json['desc'] as String? ?? '',
        icon: json['icon'] as String? ?? '',
      );
}

class MenuSection {
  final String id;
  final String name;
  final List<MenuItem> items;
  const MenuSection({required this.id, required this.name, required this.items});

  factory MenuSection.fromJson(Map<String, dynamic> json) => MenuSection(
        id: json['id'] as String? ?? '',
        name: json['name'] as String? ?? '',
        items: [
          for (final i in (json['items'] as List? ?? const []))
            MenuItem.fromJson(i as Map<String, dynamic>)
        ],
      );
}

class Place {
  final String id;
  final String name;
  final String type;
  final String icon; // Lucide icon name from the prototype
  final String color; // pin tone key: teal/coral/sun/clay/plum
  final double x; // map position, percent 0-100
  final double y;
  final double lat; // real-world coordinates, for the Google Map
  final double lng;
  final double rating;
  final int reviews;
  final String distance;
  final String priceLevel;
  final WifiInfo wifi;
  final String hours;
  final bool openNow;
  final List<String> bestTimes; // 'morning' | 'afternoon' | 'evening'
  final Deal deal;
  final List<String> tags;
  final String blurb;
  final bool seaView;
  final bool indoor;
  final bool artisan;
  final bool popularEvening;
  final String photo; // illustration kind: cowork/surf/sunset/clay/batik
  final List<MenuSection> menu;
  final Map<String, int> vibes; // vibeId -> vote count, from the server
  // Trip-planner fields — only meaningful for stops surfaced in the trip
  // catalog (see TripCatalogItem.fromPlace in data/mock/trip_catalog.dart).
  final int cost; // suggested visit cost, Rs
  final List<String> styles; // travel styles this stop suits
  final String hint; // suggested time "HH:mm"
  final bool curated; // true for the 5 original Spotto business places —
  // gates whether the trip planner links a stop to the full business drawer
  // (menu/wifi/deal/reviews); plain points of interest have none of that.

  const Place({
    required this.id,
    required this.name,
    required this.type,
    required this.icon,
    required this.color,
    required this.x,
    required this.y,
    this.lat = 0,
    this.lng = 0,
    required this.rating,
    required this.reviews,
    required this.distance,
    required this.priceLevel,
    required this.wifi,
    required this.hours,
    required this.openNow,
    required this.bestTimes,
    required this.deal,
    required this.tags,
    required this.blurb,
    required this.seaView,
    this.indoor = false,
    this.artisan = false,
    this.popularEvening = false,
    required this.photo,
    this.menu = const [],
    this.vibes = const {},
    this.cost = 0,
    this.styles = const [],
    this.hint = '',
    this.curated = false,
  });

  /// Parses the API shape: place row flattened with its JSONB data document.
  factory Place.fromJson(Map<String, dynamic> json) => Place(
        id: json['id'] as String? ?? '',
        name: json['name'] as String? ?? '',
        type: json['type'] as String? ?? '',
        icon: json['icon'] as String? ?? 'MapPin',
        color: json['color'] as String? ?? 'teal',
        x: (json['x'] as num?)?.toDouble() ?? 50,
        y: (json['y'] as num?)?.toDouble() ?? 50,
        lat: (json['lat'] as num?)?.toDouble() ?? 0,
        lng: (json['lng'] as num?)?.toDouble() ?? 0,
        rating: (json['rating'] as num?)?.toDouble() ?? 0,
        reviews: (json['reviews'] as num?)?.toInt() ?? 0,
        distance: json['distance'] as String? ?? '',
        priceLevel: json['priceLevel'] as String? ?? '',
        wifi: json['wifi'] is Map
            ? WifiInfo.fromJson((json['wifi'] as Map).cast<String, dynamic>())
            : const WifiInfo(score: 0, speed: '', label: ''),
        hours: json['hours'] as String? ?? '',
        openNow: json['openNow'] as bool? ?? false,
        bestTimes: [for (final t in (json['bestTimes'] as List? ?? const [])) t as String],
        deal: json['deal'] is Map
            ? Deal.fromJson((json['deal'] as Map).cast<String, dynamic>())
            : Deal.none,
        tags: [for (final t in (json['tags'] as List? ?? const [])) t as String],
        blurb: json['blurb'] as String? ?? '',
        seaView: json['seaView'] as bool? ?? false,
        indoor: json['indoor'] as bool? ?? false,
        artisan: json['artisan'] as bool? ?? false,
        popularEvening: json['popularEvening'] as bool? ?? false,
        photo: json['photo'] as String? ?? 'cowork',
        menu: [
          for (final s in (json['menu'] as List? ?? const []))
            MenuSection.fromJson(s as Map<String, dynamic>)
        ],
        vibes: (json['vibes'] as Map?)?.map(
              (k, v) => MapEntry(k as String, (v as num).toInt()),
            ) ??
            const {},
        cost: (json['cost'] as num?)?.toInt() ?? 0,
        styles: [for (final s in (json['styles'] as List? ?? const [])) s as String],
        hint: json['hint'] as String? ?? '',
        curated: json['curated'] as bool? ?? false,
      );
}

class TravelStyle {
  final String id;
  final String label;
  final String icon;
  final String blurb;
  const TravelStyle({required this.id, required this.label, required this.icon, required this.blurb});
}

class Interest {
  final String id;
  final String label;
  final String icon;
  const Interest({required this.id, required this.label, required this.icon});
}

class Traveler {
  final String id;
  final String name;
  final String initial;
  final int color; // ARGB
  final double x;
  final double y;
  final double lat; // real-world coordinates, for the Google Map
  final double lng;
  final String at;
  final String sub;
  final bool seededIn; // `in` in the prototype: seeded group member
  // hasPosition is false when a real group member hasn't opted into (or
  // recently reported) their location — the map should skip their pin
  // rather than show one at a stale/default (0,0) position.
  final bool hasPosition;
  const Traveler({
    required this.id,
    required this.name,
    required this.initial,
    required this.color,
    this.x = 0,
    this.y = 0,
    this.lat = 0,
    this.lng = 0,
    required this.at,
    required this.sub,
    this.seededIn = false,
    this.hasPosition = true,
  });

  static const _colors = [0xFFF0682F, 0xFF0369A1, 0xFF7C5CFF, 0xFF0B6F86, 0xFF16A34A];

  /// Maps a real, resolved group member onto the Traveler shape the map/UI
  /// already knows how to render — real position only if they've shared it
  /// (see GroupMember.lat/lng), never a fabricated one.
  factory Traveler.fromGroupMember(GroupMember m) => Traveler(
        id: m.id,
        name: m.name,
        initial: m.name.isNotEmpty ? m.name[0].toUpperCase() : '?',
        color: _colors[m.id.hashCode.abs() % _colors.length],
        lat: m.lat ?? 0,
        lng: m.lng ?? 0,
        at: m.lat != null ? 'their live location' : 'not sharing location right now',
        sub: m.email,
        seededIn: true,
        hasPosition: m.lat != null && m.lng != null,
      );

  /// Maps a nearby-traveller lookup result onto the same display shape used
  /// for the "add nearby travellers" suggestion list — distance shown
  /// instead of an exact position, since NearbyTraveller never carries one.
  factory Traveler.fromNearby(NearbyTraveller t) => Traveler(
        id: t.id,
        name: t.name,
        initial: t.name.isNotEmpty ? t.name[0].toUpperCase() : '?',
        color: _colors[t.id.hashCode.abs() % _colors.length],
        at: t.distanceKm < 1
            ? '${(t.distanceKm * 1000).round()} m away'
            : '${t.distanceKm.toStringAsFixed(1)} km away',
        sub: t.travelStyle,
        hasPosition: false,
      );

  /// The current user's own row (group host), from /me's raw map
  /// (see meProvider) — falls back to generic text before /me has loaded.
  factory Traveler.fromMe(Map<String, dynamic>? me) {
    final name = me?['name'] as String? ?? 'You';
    return Traveler(
      id: 'me',
      name: name,
      initial: name.isNotEmpty ? name[0].toUpperCase() : '?',
      color: 0xFF0B6F86,
      at: 'here',
      sub: 'Trip host',
    );
  }
}

/// A nearby opted-in traveller (GET /travellers/nearby) — coarse distance
/// only, never their exact position (unlike group members, who are a
/// mutual-trust circle the user explicitly joined).
class NearbyTraveller {
  final String id;
  final String name;
  final String travelStyle;
  final double distanceKm;
  const NearbyTraveller({
    required this.id,
    required this.name,
    required this.travelStyle,
    required this.distanceKm,
  });

  factory NearbyTraveller.fromJson(Map<String, dynamic> json) => NearbyTraveller(
        id: json['id'] as String,
        name: json['name'] as String,
        travelStyle: json['travelStyle'] as String? ?? '',
        distanceKm: (json['distanceKm'] as num).toDouble(),
      );
}

/// A traveller's rating (1-5) + text for a place. 1-2 star reviews count
/// against the vendor for the admin map's red-flag threshold.
class Review {
  final String id;
  final String placeId;
  final int rating;
  final String text;
  final String createdAt;
  final String? travellerName;

  const Review({
    required this.id,
    required this.placeId,
    required this.rating,
    required this.text,
    required this.createdAt,
    this.travellerName,
  });

  factory Review.fromJson(Map<String, dynamic> json) => Review(
        id: json['id'] as String? ?? '',
        placeId: json['placeId'] as String? ?? '',
        rating: (json['rating'] as num?)?.toInt() ?? 0,
        text: json['text'] as String? ?? '',
        createdAt: json['createdAt'] as String? ?? '',
        travellerName: json['travellerName'] as String?,
      );
}

/// A group member's real profile, resolved server-side from memberIds
/// (GET /me/group) — replaces the old fake local phone-contacts list.
/// lat/lng are only present when that member has opted into live location
/// sharing and reported a recent position — otherwise null, meaning "don't
/// show a map pin for them right now" rather than a stale/fake one.
class GroupMember {
  final String id;
  final String name;
  final String email;
  final double? lat;
  final double? lng;
  const GroupMember({required this.id, required this.name, required this.email, this.lat, this.lng});

  factory GroupMember.fromJson(Map<String, dynamic> json) => GroupMember(
        id: json['id'] as String,
        name: json['name'] as String,
        email: json['email'] as String,
        lat: (json['lat'] as num?)?.toDouble(),
        lng: (json['lng'] as num?)?.toDouble(),
      );
}

class Group {
  final String name;
  final String code;
  final List<String> memberIds;
  final List<GroupMember> members;
  const Group({
    required this.name,
    required this.code,
    required this.memberIds,
    this.members = const [],
  });

  Group copyWith({String? name, String? code, List<String>? memberIds}) => Group(
        name: name ?? this.name,
        code: code ?? this.code,
        memberIds: memberIds ?? this.memberIds,
        // Resolved members are server-owned — cleared on any local edit
        // until the next successful /me/group pull refreshes them.
        members: const [],
      );

  Map<String, dynamic> toJson() => {'name': name, 'code': code, 'memberIds': memberIds};
  factory Group.fromJson(Map<String, dynamic> json) => Group(
        name: json['name'] as String,
        code: json['code'] as String,
        memberIds: (json['memberIds'] as List).cast<String>(),
        members: [
          for (final m in (json['members'] as List? ?? const []))
            GroupMember.fromJson((m as Map).cast<String, dynamic>())
        ],
      );
}

class VibeTag {
  final String id;
  final String label;
  final String icon;
  final int color; // ARGB
  const VibeTag({required this.id, required this.label, required this.icon, required this.color});
}

class WeatherInfo {
  final String id; // sunny/cloudy/rainy
  final String label;
  final String icon;
  final String temp;
  final String tip;
  const WeatherInfo({
    required this.id,
    required this.label,
    required this.icon,
    required this.temp,
    required this.tip,
  });
}

class TimeSlot {
  final String id; // morning/afternoon/evening
  final String label;
  final String icon;
  final String range;
  const TimeSlot({required this.id, required this.label, required this.icon, required this.range});
}

/// Onboarding result: travel style + interests. Persisted as spotto_prefs.
class UserPrefs {
  final String style;
  final List<String> interests;
  const UserPrefs({required this.style, required this.interests});

  Map<String, dynamic> toJson() => {'style': style, 'interests': interests};
  factory UserPrefs.fromJson(Map<String, dynamic> json) => UserPrefs(
        style: json['style'] as String,
        interests: (json['interests'] as List).cast<String>(),
      );
}
