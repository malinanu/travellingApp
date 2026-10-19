// Inspiration Board models — mirror server/internal/http/inspirations.go.
// A BoardLink is a reel/post the traveller saved; the server watches it and
// fills in [BoardLink.places]. GeoPlace is a real-world place found on
// Google (anywhere, unlike the Sri Lanka catalog).

double? _d(Object? v) => (v as num?)?.toDouble();

/// Spotto's simple place kinds, as the server sends them.
const placeKinds = ['eat', 'drink', 'stay', 'see', 'do', 'shop'];

String kindLabel(String kind) => switch (kind) {
      'eat' => 'Food',
      'drink' => 'Drinks',
      'stay' => 'Stay',
      'see' => 'Sight',
      'do' => 'Activity',
      'shop' => 'Shopping',
      _ => 'Place',
    };

/// Lucide icon name for a kind, for trip stops (see spottoIcon).
String kindIcon(String kind) => switch (kind) {
      'eat' => 'Utensils',
      'drink' => 'Martini',
      'stay' => 'BedDouble',
      'see' => 'Camera',
      'do' => 'Sparkles',
      'shop' => 'ShoppingBag',
      _ => 'MapPin',
    };

class GeoPlace {
  final String id;
  final String name;
  final double lat;
  final double lng;
  final String address;
  final String city;
  final String countryCode;
  final String kind;
  final double? rating;
  final int ratingCount;
  final String mapsUrl;

  /// Relative to the API (resolve with resolveMediaUrl); null without a photo.
  final String? photoUrl;

  const GeoPlace({
    required this.id,
    required this.name,
    required this.lat,
    required this.lng,
    this.address = '',
    this.city = '',
    this.countryCode = '',
    this.kind = 'see',
    this.rating,
    this.ratingCount = 0,
    this.mapsUrl = '',
    this.photoUrl,
  });

  factory GeoPlace.fromJson(Map<String, dynamic> j) => GeoPlace(
        id: j['id'] as String,
        name: j['name'] as String? ?? '',
        lat: _d(j['lat']) ?? 0,
        lng: _d(j['lng']) ?? 0,
        address: j['address'] as String? ?? '',
        city: j['city'] as String? ?? '',
        countryCode: j['countryCode'] as String? ?? '',
        kind: j['kind'] as String? ?? 'see',
        rating: _d(j['rating']),
        ratingCount: (j['ratingCount'] as num?)?.toInt() ?? 0,
        mapsUrl: j['mapsUrl'] as String? ?? '',
        photoUrl: (j['photoUrl'] as String?)?.isNotEmpty == true ? j['photoUrl'] as String : null,
      );

  Map<String, dynamic> toJson() => {
        'id': id,
        'name': name,
        'lat': lat,
        'lng': lng,
        'address': address,
        'city': city,
        'countryCode': countryCode,
        'kind': kind,
        if (rating != null) 'rating': rating,
        'ratingCount': ratingCount,
        'mapsUrl': mapsUrl,
        if (photoUrl != null) 'photoUrl': photoUrl,
      };
}

/// One place found in a reel.
class BoardPlace {
  final String name;
  final String kind;
  final String area;
  final String city;
  final String country;

  /// Where the reel showed it: on_screen | speech | caption.
  final String evidence;
  final int? atSec;
  final double confidence;

  /// Set when the place was found on Google.
  final GeoPlace? geoPlace;

  /// Set when it is also a Spotto catalog place.
  final String? placeId;

  const BoardPlace({
    required this.name,
    this.kind = 'see',
    this.area = '',
    this.city = '',
    this.country = '',
    this.evidence = 'caption',
    this.atSec,
    this.confidence = 0,
    this.geoPlace,
    this.placeId,
  });

  /// Located on a map — the reel's name was matched to a real place.
  bool get confirmed => geoPlace != null;

  String get where => [area, city].where((s) => s.isNotEmpty).join(', ');

  /// "Seen at 0:42 · on screen", for the place card.
  String get evidenceLabel {
    final how = switch (evidence) {
      'on_screen' => 'on screen',
      'speech' => 'mentioned',
      _ => 'in the caption',
    };
    if (atSec == null) return how;
    final m = atSec! ~/ 60, s = atSec! % 60;
    return 'At $m:${s.toString().padLeft(2, '0')} · $how';
  }

  factory BoardPlace.fromJson(Map<String, dynamic> j) => BoardPlace(
        name: j['name'] as String? ?? '',
        kind: j['kind'] as String? ?? 'see',
        area: j['area'] as String? ?? '',
        city: j['city'] as String? ?? '',
        country: j['country'] as String? ?? '',
        evidence: j['evidence'] as String? ?? 'caption',
        atSec: (j['atSec'] as num?)?.toInt(),
        confidence: _d(j['confidence']) ?? 0,
        geoPlace: j['geoPlace'] == null ? null : GeoPlace.fromJson((j['geoPlace'] as Map).cast<String, dynamic>()),
        placeId: (j['placeId'] as String?)?.isNotEmpty == true ? j['placeId'] as String : null,
      );
}

/// A saved reel/post and what the server found in it.
class BoardLink {
  final String id;
  final String url;
  final String note;
  final DateTime createdAt;

  /// instagram | tiktok | youtube | facebook
  final String platform;

  /// pending | processing | done | partial | failed
  final String status;

  /// What went wrong (failed), or why results are partial.
  final String error;
  final String caption;
  final String author;
  final String thumbnailUrl;
  final List<BoardPlace> places;

  const BoardLink({
    required this.id,
    required this.url,
    this.note = '',
    required this.createdAt,
    this.platform = '',
    this.status = 'pending',
    this.error = '',
    this.caption = '',
    this.author = '',
    this.thumbnailUrl = '',
    this.places = const [],
  });

  bool get working => status == 'pending' || status == 'processing';
  bool get failed => status == 'failed';
  bool get canRetry => status == 'failed' || status == 'partial';
  List<BoardPlace> get confirmedPlaces => places.where((p) => p.confirmed).toList();

  /// The reel's main city, for headings ("Phuket").
  String get region {
    final counts = <String, int>{};
    for (final p in places) {
      final c = p.geoPlace?.city.isNotEmpty == true ? p.geoPlace!.city : p.city;
      if (c.isNotEmpty) counts[c] = (counts[c] ?? 0) + 1;
    }
    if (counts.isEmpty) return '';
    return (counts.entries.toList()..sort((a, b) => b.value.compareTo(a.value))).first.key;
  }

  factory BoardLink.fromJson(Map<String, dynamic> j) => BoardLink(
        id: j['id'] as String,
        url: j['url'] as String? ?? '',
        note: j['note'] as String? ?? '',
        createdAt: DateTime.tryParse(j['createdAt'] as String? ?? '') ?? DateTime.now(),
        platform: j['platform'] as String? ?? '',
        status: j['status'] as String? ?? 'pending',
        error: j['error'] as String? ?? '',
        caption: j['caption'] as String? ?? '',
        author: j['author'] as String? ?? '',
        thumbnailUrl: j['thumbnailUrl'] as String? ?? '',
        places: [
          for (final p in (j['places'] as List? ?? const []))
            BoardPlace.fromJson((p as Map).cast<String, dynamic>()),
        ],
      );
}

/// Summary of a Spotto catalog place on a saved entry.
class SavedCatalogPlace {
  final String id;
  final String name;
  final double lat;
  final double lng;
  final String categoryId;
  const SavedCatalogPlace({required this.id, required this.name, required this.lat, required this.lng, this.categoryId = ''});

  factory SavedCatalogPlace.fromJson(Map<String, dynamic> j) => SavedCatalogPlace(
        id: j['id'] as String,
        name: j['name'] as String? ?? '',
        lat: _d(j['lat']) ?? 0,
        lng: _d(j['lng']) ?? 0,
        categoryId: j['categoryId'] as String? ?? '',
      );

  Map<String, dynamic> toJson() => {'id': id, 'name': name, 'lat': lat, 'lng': lng, 'categoryId': categoryId};
}

/// A "save for later" place: a Google place from a reel or a catalog place.
class SavedPlace {
  final String id;
  final String? placeId;
  final String? geoPlaceId;
  final String? sourceLinkId;
  final DateTime createdAt;
  final GeoPlace? geoPlace;
  final SavedCatalogPlace? place;

  const SavedPlace({
    required this.id,
    this.placeId,
    this.geoPlaceId,
    this.sourceLinkId,
    required this.createdAt,
    this.geoPlace,
    this.place,
  });

  String get name => geoPlace?.name ?? place?.name ?? '';
  double? get lat => geoPlace?.lat ?? place?.lat;
  double? get lng => geoPlace?.lng ?? place?.lng;
  String get kind => geoPlace?.kind ?? 'see';
  String get city => geoPlace?.city ?? '';

  static String? _s(Object? v) => (v as String?)?.isNotEmpty == true ? v as String : null;

  factory SavedPlace.fromJson(Map<String, dynamic> j) => SavedPlace(
        id: j['id'] as String,
        placeId: _s(j['placeId']),
        geoPlaceId: _s(j['geoPlaceId']),
        sourceLinkId: _s(j['sourceLinkId']),
        createdAt: DateTime.tryParse(j['createdAt'] as String? ?? '') ?? DateTime.now(),
        geoPlace: j['geoPlace'] == null ? null : GeoPlace.fromJson((j['geoPlace'] as Map).cast<String, dynamic>()),
        place: j['place'] == null ? null : SavedCatalogPlace.fromJson((j['place'] as Map).cast<String, dynamic>()),
      );

  Map<String, dynamic> toJson() => {
        'id': id,
        if (placeId != null) 'placeId': placeId,
        if (geoPlaceId != null) 'geoPlaceId': geoPlaceId,
        if (sourceLinkId != null) 'sourceLinkId': sourceLinkId,
        'createdAt': createdAt.toIso8601String(),
        if (geoPlace != null) 'geoPlace': geoPlace!.toJson(),
        if (place != null) 'place': place!.toJson(),
      };
}

final _shareUrlRe = RegExp(
  r'https?://(?:www\.|m\.|vm\.|vt\.|web\.)?(?:instagram\.com|tiktok\.com|youtube\.com|youtu\.be|facebook\.com|fb\.watch)/\S+',
  caseSensitive: false,
);

/// The first reel/post link in shared or pasted text ("Check this out
/// https://www.instagram.com/reel/..."), or null.
String? findSupportedLink(String text) => _shareUrlRe.firstMatch(text)?.group(0);
