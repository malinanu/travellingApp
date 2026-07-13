// Core Spotto data models — ported from Prototype/design/data.jsx.
// Plain immutable classes; only persisted types get toJson/fromJson.

class WifiInfo {
  final double score;
  final String speed;
  final String label;
  const WifiInfo({required this.score, required this.speed, required this.label});
}

class Deal {
  final bool active;
  final String? label;
  final String? tag;
  const Deal({required this.active, this.label, this.tag});
  static const none = Deal(active: false);
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
}

class MenuSection {
  final String id;
  final String name;
  final List<MenuItem> items;
  const MenuSection({required this.id, required this.name, required this.items});
}

class Place {
  final String id;
  final String name;
  final String type;
  final String icon; // Lucide icon name from the prototype
  final String color; // pin tone key: teal/coral/sun/clay/plum
  final double x; // map position, percent 0-100
  final double y;
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
    required this.at,
    required this.sub,
    this.seededIn = false,
  });
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
