// Travel content ("Stories"): audio guides and audiobooks, blog posts,
// travel guides, itineraries and tips, written in the Command Center and
// read from the public GET /content API (server/internal/http/content.go).

import 'dart:math' as math;

/// A JSON list field, or empty when it's missing or not a list (an `as
/// List?` cast would throw on the wrong type instead).
List<Object?> _list(Object? v) => v is List ? v : const [];

/// The kinds of post, in the order the Stories tab offers them.
enum ContentKind {
  audio('audio', 'Audio', 'Audio guide'),
  blog('blog', 'Blogs', 'Blog'),
  guide('guide', 'Guides', 'Travel guide'),
  itinerary('itinerary', 'Itineraries', 'Itinerary'),
  tip('tip', 'Tips', 'Tip');

  const ContentKind(this.id, this.plural, this.label);

  /// The API's `type` value.
  final String id;

  /// Filter chip label.
  final String plural;

  /// Label on a single post.
  final String label;

  static ContentKind fromId(String? id) =>
      ContentKind.values.firstWhere((k) => k.id == id, orElse: () => ContentKind.blog);
}

class ContentChapter {
  final String title;
  final int startSec;
  const ContentChapter({required this.title, required this.startSec});

  factory ContentChapter.fromJson(Map<String, dynamic> json) => ContentChapter(
        title: json['title'] as String? ?? '',
        startSec: (json['startSec'] as num?)?.toInt() ?? 0,
      );
}

/// One stop of an itinerary day: a Spotto place ([placeId] set) or a named
/// point, optionally at a time of day ("09:30").
class ItineraryStop {
  final String placeId;
  final String name;
  final double? lat;
  final double? lng;
  final String time;
  final String note;
  const ItineraryStop({
    this.placeId = '',
    required this.name,
    this.lat,
    this.lng,
    this.time = '',
    this.note = '',
  });

  factory ItineraryStop.fromJson(Map<String, dynamic> json) => ItineraryStop(
        placeId: json['placeId'] as String? ?? '',
        name: json['name'] as String? ?? '',
        lat: (json['lat'] as num?)?.toDouble(),
        lng: (json['lng'] as num?)?.toDouble(),
        time: json['time'] as String? ?? '',
        note: json['note'] as String? ?? '',
      );

  /// Minutes since midnight, or null when there's no (valid) time.
  int? get minutes {
    final m = RegExp(r'^(\d{1,2}):(\d{2})$').firstMatch(time);
    if (m == null) return null;
    final h = int.parse(m.group(1)!), min = int.parse(m.group(2)!);
    if (h > 23 || min > 59) return null;
    return h * 60 + min;
  }
}

class ItineraryDay {
  final String title;
  final List<ItineraryStop> stops;
  const ItineraryDay({required this.title, required this.stops});

  factory ItineraryDay.fromJson(Map<String, dynamic> json) => ItineraryDay(
        title: json['title'] as String? ?? '',
        stops: [
          for (final s in _list(json['stops']))
            if (s is Map<String, dynamic>) ItineraryStop.fromJson(s),
        ],
      );
}

/// A Spotto place linked from a post (a guide's places, an itinerary's
/// stops), as the API summarises it.
class ContentPlace {
  final String id;
  final String name;
  final String category;
  final double lat;
  final double lng;
  final String cover;
  const ContentPlace({
    required this.id,
    required this.name,
    this.category = '',
    this.lat = 0,
    this.lng = 0,
    this.cover = '',
  });

  factory ContentPlace.fromJson(Map<String, dynamic> json) => ContentPlace(
        id: json['id'] as String? ?? '',
        name: json['name'] as String? ?? '',
        category: json['category'] as String? ?? '',
        lat: (json['lat'] as num?)?.toDouble() ?? 0,
        lng: (json['lng'] as num?)?.toDouble() ?? 0,
        cover: json['cover'] as String? ?? '',
      );
}

class ContentItem {
  final String id;
  final ContentKind kind;
  final String slug;
  final String title;
  final String summary;

  /// Markdown. Empty in list responses.
  final String body;

  /// Signed "/media/raw/…" paths (see resolveMediaUrl), or ''.
  final String coverUrl;
  final String audioUrl;
  final int audioSeconds;
  final List<ContentChapter> chapters;
  final List<String> placeIds;
  final List<ItineraryDay> days;
  final List<String> tags;
  final String regionId;
  final String regionName;
  final double? regionLat;
  final double? regionLng;
  final String authorName;
  final DateTime? publishedAt;

  /// Linked places — only on a single post (GET /content/{id}).
  final List<ContentPlace> places;

  const ContentItem({
    required this.id,
    required this.kind,
    this.slug = '',
    required this.title,
    this.summary = '',
    this.body = '',
    this.coverUrl = '',
    this.audioUrl = '',
    this.audioSeconds = 0,
    this.chapters = const [],
    this.placeIds = const [],
    this.days = const [],
    this.tags = const [],
    this.regionId = '',
    this.regionName = '',
    this.regionLat,
    this.regionLng,
    this.authorName = '',
    this.publishedAt,
    this.places = const [],
  });

  factory ContentItem.fromJson(Map<String, dynamic> json) {
    List<T> list<T>(String key, T Function(Map<String, dynamic>) parse) => [
          for (final v in _list(json[key]))
            if (v is Map<String, dynamic>) parse(v),
        ];
    List<String> strings(String key) => [
          for (final v in _list(json[key]))
            if (v is String) v,
        ];
    return ContentItem(
      id: json['id'] as String? ?? '',
      kind: ContentKind.fromId(json['type'] as String?),
      slug: json['slug'] as String? ?? '',
      title: json['title'] as String? ?? '',
      summary: json['summary'] as String? ?? '',
      body: json['body'] as String? ?? '',
      coverUrl: json['coverUrl'] as String? ?? '',
      audioUrl: json['audioUrl'] as String? ?? '',
      audioSeconds: (json['audioSeconds'] as num?)?.toInt() ?? 0,
      chapters: list('chapters', ContentChapter.fromJson),
      placeIds: strings('placeIds'),
      days: list('days', ItineraryDay.fromJson),
      tags: strings('tags'),
      regionId: json['regionId'] as String? ?? '',
      regionName: json['regionName'] as String? ?? '',
      regionLat: (json['regionLat'] as num?)?.toDouble(),
      regionLng: (json['regionLng'] as num?)?.toDouble(),
      authorName: json['authorName'] as String? ?? '',
      publishedAt: DateTime.tryParse(json['publishedAt'] as String? ?? ''),
      places: list('places', ContentPlace.fromJson),
    );
  }

  int get stopCount => days.fold(0, (n, d) => n + d.stops.length);

  /// The linked place [id], when the post carries it.
  ContentPlace? place(String id) {
    for (final p in places) {
      if (p.id == id) return p;
    }
    return null;
  }
}

/// Straight-line km between two points.
double contentDistanceKm(double lat1, double lng1, double lat2, double lng2) {
  const r = 6371.0;
  double rad(double d) => d * math.pi / 180;
  final dLat = rad(lat2 - lat1), dLng = rad(lng2 - lng1);
  final h = math.pow(math.sin(dLat / 2), 2) +
      math.cos(rad(lat1)) * math.cos(rad(lat2)) * math.pow(math.sin(dLng / 2), 2);
  return 2 * r * math.asin(math.sqrt(h));
}

/// Splits posts into those about a town within [radiusKm] of the traveller
/// (nearest town first) and the rest (in the order given, newest first).
/// Posts without a town are never "near".
({List<ContentItem> near, List<ContentItem> rest}) splitNearby(
  List<ContentItem> items, {
  required double lat,
  required double lng,
  double radiusKm = 40,
}) {
  final near = <(ContentItem, double)>[];
  final rest = <ContentItem>[];
  for (final item in items) {
    final rLat = item.regionLat, rLng = item.regionLng;
    final km = rLat == null || rLng == null ? null : contentDistanceKm(lat, lng, rLat, rLng);
    if (km != null && km <= radiusKm) {
      near.add((item, km));
    } else {
      rest.add(item);
    }
  }
  // Stable: posts about the same town keep their newest-first order.
  final indexed = [for (var i = 0; i < near.length; i++) (i, near[i])];
  indexed.sort((a, b) {
    final c = a.$2.$2.compareTo(b.$2.$2);
    return c != 0 ? c : a.$1.compareTo(b.$1);
  });
  return (near: [for (final e in indexed) e.$2.$1], rest: rest);
}

/// "12 min", "1 h 05 min" for an audio length; '' when unknown.
String formatListenLength(int seconds) {
  if (seconds <= 0) return '';
  final mins = (seconds / 60).round();
  if (mins < 60) return '${math.max(mins, 1)} min';
  return '${mins ~/ 60} h ${(mins % 60).toString().padLeft(2, '0')} min';
}

/// 754 → "12:34", 3723 → "1:02:03".
String formatClock(Duration d) {
  final s = d.inSeconds < 0 ? 0 : d.inSeconds;
  final h = s ~/ 3600, m = (s % 3600) ~/ 60, sec = (s % 60).toString().padLeft(2, '0');
  return h > 0 ? '$h:${m.toString().padLeft(2, '0')}:$sec' : '$m:$sec';
}
