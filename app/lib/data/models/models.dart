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
  final bool active;
  final String? label;
  final String? tag;
  const Deal({required this.active, this.label, this.tag});
  static const none = Deal(active: false);

  factory Deal.fromJson(Map<String, dynamic> json) => Deal(
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
  });
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

class Group {
  final String name;
  final String code;
  final List<String> memberIds;
  const Group({required this.name, required this.code, required this.memberIds});

  Group copyWith({String? name, String? code, List<String>? memberIds}) => Group(
        name: name ?? this.name,
        code: code ?? this.code,
        memberIds: memberIds ?? this.memberIds,
      );

  Map<String, dynamic> toJson() => {'name': name, 'code': code, 'memberIds': memberIds};
  factory Group.fromJson(Map<String, dynamic> json) => Group(
        name: json['name'] as String,
        code: json['code'] as String,
        memberIds: (json['memberIds'] as List).cast<String>(),
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
