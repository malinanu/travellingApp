// Core Spotto data models — ported from Prototype/design/data.jsx.
// Plain immutable classes; only persisted types get toJson/fromJson.

/// Parses a string-list field that may still be in an older single-string
/// shape — `v as List?` is a hard cast in Dart that throws on a non-null,
/// wrong-typed value rather than returning null, so a server/app running
/// mismatched versions of a field's shape could crash parsing instead of
/// just showing slightly-stale data. Never throws.
List<String> _parseStringList(dynamic v) {
  if (v is List) return v.cast<String>();
  if (v is String && v.isNotEmpty) return [v];
  return const [];
}

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

class PaymentCard {
  final String id;
  final String brand;
  final String last4;
  final int expMonth;
  final int expYear;
  final String holderName;
  final bool isDefault;

  const PaymentCard({
    required this.id,
    required this.brand,
    required this.last4,
    required this.expMonth,
    required this.expYear,
    required this.holderName,
    required this.isDefault,
  });

  factory PaymentCard.fromJson(Map<String, dynamic> json) => PaymentCard(
        id: json['id'] as String? ?? '',
        brand: json['brand'] as String? ?? 'Card',
        last4: json['last4'] as String? ?? '0000',
        expMonth: (json['expMonth'] as num?)?.toInt() ?? 12,
        expYear: (json['expYear'] as num?)?.toInt() ?? 2026,
        holderName: json['holderName'] as String? ?? '',
        isDefault: json['isDefault'] as bool? ?? false,
      );

  Map<String, dynamic> toJson() => {
        'id': id,
        'brand': brand,
        'last4': last4,
        'expMonth': expMonth,
        'expYear': expYear,
        'holderName': holderName,
        'isDefault': isDefault,
      };
}

class ContactInfo {
  final String? phone;
  final String? website;
  final String? instagram;
  final String? facebook;
  const ContactInfo({this.phone, this.website, this.instagram, this.facebook});
  static const none = ContactInfo();

  bool get isEmpty =>
      (phone == null || phone!.isEmpty) &&
      (website == null || website!.isEmpty) &&
      (instagram == null || instagram!.isEmpty) &&
      (facebook == null || facebook!.isEmpty);

  factory ContactInfo.fromJson(Map<String, dynamic> json) => ContactInfo(
        phone: json['phone'] as String?,
        website: json['website'] as String?,
        instagram: json['instagram'] as String?,
        facebook: json['facebook'] as String?,
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
  final String img; // public R2 photo URL, or '' to show the icon
  const MenuItem({
    required this.id,
    required this.name,
    required this.price,
    required this.desc,
    required this.icon,
    this.img = '',
  });

  factory MenuItem.fromJson(Map<String, dynamic> json) => MenuItem(
        id: json['id'] as String? ?? '',
        name: json['name'] as String? ?? '',
        price: (json['price'] as num?)?.toInt() ?? 0,
        desc: json['desc'] as String? ?? '',
        icon: json['icon'] as String? ?? '',
        img: json['img'] as String? ?? '',
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

/// A place's current vibe reading — the average of each traveller's most
/// recent check-in (see server VibeAggregate), not an all-time blend.
class VibeAggregate {
  final double productive;
  final double loud;
  final double crowded;
  final int sampleSize;
  const VibeAggregate({
    this.productive = 0,
    this.loud = 0,
    this.crowded = 0,
    this.sampleSize = 0,
  });

  factory VibeAggregate.fromJson(Map<String, dynamic> json) => VibeAggregate(
        productive: (json['productive'] as num?)?.toDouble() ?? 0,
        loud: (json['loud'] as num?)?.toDouble() ?? 0,
        crowded: (json['crowded'] as num?)?.toDouble() ?? 0,
        sampleSize: (json['sampleSize'] as num?)?.toInt() ?? 0,
      );
}

/// One traveller's own timestamped vibe rating of a place — only ever the
/// latest one (see server MyLatestVibeCheckin); null means they haven't
/// rated this place (yet, or ever).
class VibeCheckin {
  final int productive;
  final int loud;
  final int crowded;
  final DateTime createdAt;
  const VibeCheckin({
    required this.productive,
    required this.loud,
    required this.crowded,
    required this.createdAt,
  });

  factory VibeCheckin.fromJson(Map<String, dynamic> json) => VibeCheckin(
        productive: (json['productive'] as num?)?.toInt() ?? 0,
        loud: (json['loud'] as num?)?.toInt() ?? 0,
        crowded: (json['crowded'] as num?)?.toInt() ?? 0,
        createdAt: DateTime.tryParse(json['createdAt'] as String? ?? '') ?? DateTime.now(),
      );
}

class Place {
  final String id;
  final String name;
  final String type;
  final String categoryId; // canonical vendor-category id, see data/vendor_categories.dart
  final String region; // display name, e.g. "Galle" — '' for older/seed data
  final String regionId; // e.g. "galle" — matches server/internal/geo.Regions
  final String address; // full street address, from Google Places imports
  final String icon; // Lucide icon name from the prototype
  final String color; // pin tone key: teal/coral/sun/clay/plum
  final double x; // map position, percent 0-100
  final double y;
  final double lat; // real-world coordinates, for the Google Map
  final double lng;
  final double rating;
  final int reviews;
  final String distance;
  final double? distanceKm; // real straight-line km, null unless the fetch was location-sorted
  final String priceLevel;
  final WifiInfo wifi;
  final String hours;
  final bool openNow;
  final List<String> bestTimes; // 'morning' | 'afternoon' | 'evening'
  final Deal deal;
  final List<String> tags;
  final String blurb;
  final ContactInfo contact;
  final bool? wheelchairAccessible; // null = unknown, from live Google Places details
  final bool seaView;
  final bool indoor;
  final bool artisan;
  final bool popularEvening;
  final String photo; // illustration kind: cowork/surf/sunset/clay/batik
  final String cover; // uploaded cover photo URL (R2), '' = use illustration
  // ~320px thumbnail derivative of [cover] — '' if not generated yet (the
  // background thumbnail worker fills it in within ~30s of upload) or if
  // [cover] itself is empty. Card/header UI should prefer this over
  // [cover], which is only worth requesting for something more prominent
  // than a card thumbnail — there currently isn't one for covers.
  final String coverThumb;
  // Inline base64 blur placeholder for [cover] ('data:image/jpeg;base64,...'),
  // shown instantly with no extra network fetch. '' under the same
  // conditions as [coverThumb].
  final String coverMicro;
  final List<String> gallery; // uploaded gallery photo URLs (R2)
  // Same-order thumbnail derivatives of [gallery] — each entry may be ''
  // under the same conditions as [coverThumb].
  final List<String> galleryThumbs;
  final String promoVideo; // uploaded promo video URL (R2), '' = none
  final List<MenuSection> menu;
  final VibeAggregate vibe; // place's current vibe reading, from the server
  final VibeCheckin? myVibe; // caller's own latest check-in — handlePlace only
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
    this.categoryId = '',
    this.region = '',
    this.regionId = '',
    this.address = '',
    required this.icon,
    required this.color,
    required this.x,
    required this.y,
    this.lat = 0,
    this.lng = 0,
    required this.rating,
    required this.reviews,
    required this.distance,
    this.distanceKm,
    required this.priceLevel,
    required this.wifi,
    required this.hours,
    required this.openNow,
    required this.bestTimes,
    required this.deal,
    required this.tags,
    required this.blurb,
    this.contact = ContactInfo.none,
    this.wheelchairAccessible,
    required this.seaView,
    this.indoor = false,
    this.artisan = false,
    this.popularEvening = false,
    required this.photo,
    this.cover = '',
    this.coverThumb = '',
    this.coverMicro = '',
    this.gallery = const [],
    this.galleryThumbs = const [],
    this.promoVideo = '',
    this.menu = const [],
    this.vibe = const VibeAggregate(),
    this.myVibe,
    this.cost = 0,
    this.styles = const [],
    this.hint = '',
    this.curated = false,
  });

  /// Parses the API shape: place row flattened with its JSONB data document.
  ///
  /// `cost`/`priceLevel` need defensive handling: `cost` is normally the
  /// trip-planner's numeric Rs estimate, but some imported records instead
  /// carry a Google-style `$`/`$$`/`$$$` string under that same key (a data
  /// pipeline mix-up) — a plain `as num?` cast throws on that shape, and
  /// since this factory runs inside a list comprehension (see
  /// PlacesNotifier.refresh), one bad record would otherwise blank out the
  /// entire live places list. Treat a non-numeric `cost` as the price-level
  /// tier when `priceLevel` itself wasn't set.
  factory Place.fromJson(Map<String, dynamic> json) {
    final rawCost = json['cost'];
    final numericCost = rawCost is num ? rawCost.toInt() : 0;
    var priceLevel = json['priceLevel'] as String? ?? '';
    if (priceLevel.isEmpty && rawCost is String) priceLevel = rawCost;

    return Place(
      id: json['id'] as String? ?? '',
      name: json['name'] as String? ?? '',
      type: json['type'] as String? ?? '',
      categoryId: json['categoryId'] as String? ?? '',
      region: json['region'] as String? ?? '',
      regionId: json['regionId'] as String? ?? '',
      address: json['address'] as String? ?? '',
      icon: json['icon'] as String? ?? 'MapPin',
      color: json['color'] as String? ?? 'teal',
      x: (json['x'] as num?)?.toDouble() ?? 50,
      y: (json['y'] as num?)?.toDouble() ?? 50,
      lat: (json['lat'] as num?)?.toDouble() ?? 0,
      lng: (json['lng'] as num?)?.toDouble() ?? 0,
      rating: (json['rating'] as num?)?.toDouble() ?? 0,
      reviews: (json['reviews'] as num?)?.toInt() ?? 0,
      distance: json['distance'] as String? ?? '',
      distanceKm: (json['distanceKm'] as num?)?.toDouble(),
      priceLevel: priceLevel,
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
      contact: json['contact'] is Map
          ? ContactInfo.fromJson((json['contact'] as Map).cast<String, dynamic>())
          : ContactInfo.none,
      wheelchairAccessible: json['wheelchairAccessible'] as bool?,
      seaView: json['seaView'] as bool? ?? false,
      indoor: json['indoor'] as bool? ?? false,
      artisan: json['artisan'] as bool? ?? false,
      popularEvening: json['popularEvening'] as bool? ?? false,
      photo: json['photo'] as String? ?? 'cowork',
      cover: json['cover'] as String? ?? '',
      coverThumb: json['coverThumb'] as String? ?? '',
      coverMicro: json['coverMicro'] as String? ?? '',
      gallery: [for (final g in (json['gallery'] as List? ?? const [])) g as String],
      galleryThumbs: [for (final g in (json['galleryThumbs'] as List? ?? const [])) g as String],
      promoVideo: json['promoVideo'] as String? ?? '',
      menu: [
        for (final s in (json['menu'] as List? ?? const []))
          MenuSection.fromJson(s as Map<String, dynamic>)
      ],
      vibe: json['vibe'] is Map
          ? VibeAggregate.fromJson((json['vibe'] as Map).cast<String, dynamic>())
          : const VibeAggregate(),
      myVibe: json['myVibe'] is Map
          ? VibeCheckin.fromJson((json['myVibe'] as Map).cast<String, dynamic>())
          : null,
      cost: numericCost,
      styles: [for (final s in (json['styles'] as List? ?? const [])) s as String],
      hint: json['hint'] as String? ?? '',
      curated: json['curated'] as bool? ?? false,
    );
  }
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
        sub: t.travelStyle.join(', '),
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
  final List<String> travelStyle;
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
        travelStyle: _parseStringList(json['travelStyle']),
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
  final List<ReviewPhoto> photos;

  const Review({
    required this.id,
    required this.placeId,
    required this.rating,
    required this.text,
    required this.createdAt,
    this.travellerName,
    this.photos = const [],
  });

  factory Review.fromJson(Map<String, dynamic> json) => Review(
        id: json['id'] as String? ?? '',
        placeId: json['placeId'] as String? ?? '',
        rating: (json['rating'] as num?)?.toInt() ?? 0,
        text: json['text'] as String? ?? '',
        createdAt: json['createdAt'] as String? ?? '',
        travellerName: json['travellerName'] as String?,
        photos: [
          for (final p in (json['photos'] as List? ?? const []))
            ReviewPhoto.fromJson(p as Map<String, dynamic>),
        ],
      );
}

/// One attached review photo. Mirrors [PlacePhoto]'s url/thumbUrl split: a
/// small (~320px) JPEG derivative for the grid, empty for photos uploaded
/// before thumbnails existed or if generation failed — fall back to [url]
/// whenever [thumbUrl] is empty. The full-screen viewer always uses [url].
class ReviewPhoto {
  final String url;
  final String thumbUrl;
  // Inline base64 blur placeholder ('data:image/jpeg;base64,...'), shown
  // instantly with no extra network fetch — '' under the same conditions
  // as [thumbUrl].
  final String microThumb;
  const ReviewPhoto({required this.url, this.thumbUrl = '', this.microThumb = ''});

  factory ReviewPhoto.fromJson(Map<String, dynamic> json) => ReviewPhoto(
        url: json['url'] as String? ?? '',
        thumbUrl: json['thumbUrl'] as String? ?? '',
        microThumb: json['microThumb'] as String? ?? '',
      );
}

/// One bookable thing a vendor offers (e.g. "Beginner surf lesson", "Deluxe
/// Room") — see GET /places/{id}/services. No payment is collected
/// anywhere in this platform; [priceLabel] is display-only, the vendor
/// collects payment in person.
class BookableService {
  final String id;
  final String vendorId;
  final String type; // session | table | stay — see data/vendor_categories.dart's BookingType
  final String name;
  final String description;
  final int durationMins; // session/table only
  final int capacity;
  final int partySizeMin; // session/table only
  final int partySizeMax; // session/table only
  final int guestsPerRoom; // stay only
  final String priceLabel;
  final bool active;

  const BookableService({
    required this.id,
    required this.vendorId,
    required this.type,
    required this.name,
    this.description = '',
    this.durationMins = 0,
    this.capacity = 1,
    this.partySizeMin = 1,
    this.partySizeMax = 1,
    this.guestsPerRoom = 1,
    this.priceLabel = '',
    this.active = true,
  });

  factory BookableService.fromJson(Map<String, dynamic> json) => BookableService(
        id: json['id'] as String? ?? '',
        vendorId: json['vendorId'] as String? ?? '',
        type: json['type'] as String? ?? '',
        name: json['name'] as String? ?? '',
        description: json['description'] as String? ?? '',
        durationMins: (json['durationMins'] as num?)?.toInt() ?? 0,
        capacity: (json['capacity'] as num?)?.toInt() ?? 1,
        partySizeMin: (json['partySizeMin'] as num?)?.toInt() ?? 1,
        partySizeMax: (json['partySizeMax'] as num?)?.toInt() ?? 1,
        guestsPerRoom: (json['guestsPerRoom'] as num?)?.toInt() ?? 1,
        priceLabel: json['priceLabel'] as String? ?? '',
        active: json['active'] as bool? ?? true,
      );
}

/// One open time window a service is bookable within on a given weekday
/// (see BookableService.availability — server-computed, this app only
/// consumes the resulting slot list from GET .../availability, never the
/// raw weekly rule).
class BookingSlot {
  final DateTime start;
  final DateTime end;
  final int remaining;
  const BookingSlot({required this.start, required this.end, required this.remaining});

  factory BookingSlot.fromJson(Map<String, dynamic> json) => BookingSlot(
        start: DateTime.tryParse(json['start'] as String? ?? '') ?? DateTime.now(),
        end: DateTime.tryParse(json['end'] as String? ?? '') ?? DateTime.now(),
        remaining: (json['remaining'] as num?)?.toInt() ?? 0,
      );
}

/// A traveller's real, auto-confirmed commitment against a [BookableService].
/// No payment collected — the vendor checks the traveller in via a
/// BOOK-XXXX QR/code, the same mechanism offer redemption uses (see
/// ClaimsNotifier) — [code] is that check-in code.
class Booking {
  final String id;
  final String code;
  final String vendorId;
  final String serviceId;
  final String type;
  final String placeId;
  final String startAt;
  final String endAt;
  final int partySize;
  final String status; // confirmed | cancelled | completed | no_show
  final String note;
  final String createdAt;
  final String serviceName;
  final String vendorName;

  const Booking({
    required this.id,
    required this.code,
    required this.vendorId,
    required this.serviceId,
    required this.type,
    required this.placeId,
    required this.startAt,
    required this.endAt,
    this.partySize = 1,
    this.status = 'confirmed',
    this.note = '',
    this.createdAt = '',
    this.serviceName = '',
    this.vendorName = '',
  });

  factory Booking.fromJson(Map<String, dynamic> json) => Booking(
        id: json['id'] as String? ?? '',
        code: json['code'] as String? ?? '',
        vendorId: json['vendorId'] as String? ?? '',
        serviceId: json['serviceId'] as String? ?? '',
        type: json['type'] as String? ?? '',
        placeId: json['placeId'] as String? ?? '',
        startAt: json['startAt'] as String? ?? '',
        endAt: json['endAt'] as String? ?? '',
        partySize: (json['partySize'] as num?)?.toInt() ?? 1,
        status: json['status'] as String? ?? 'confirmed',
        note: json['note'] as String? ?? '',
        createdAt: json['createdAt'] as String? ?? '',
        serviceName: json['serviceName'] as String? ?? '',
        vendorName: json['vendorName'] as String? ?? '',
      );

  /// Round-trips for LocalStore caching (see BookingsNotifier) — never sent
  /// back to the server, which only ever receives a fresh POST /bookings.
  Map<String, dynamic> toJson() => {
        'id': id, 'code': code, 'vendorId': vendorId, 'serviceId': serviceId, 'type': type,
        'placeId': placeId, 'startAt': startAt, 'endAt': endAt, 'partySize': partySize,
        'status': status, 'note': note, 'createdAt': createdAt,
        'serviceName': serviceName, 'vendorName': vendorName,
      };

  Booking copyWith({String? status}) => Booking(
        id: id, code: code, vendorId: vendorId, serviceId: serviceId, type: type, placeId: placeId,
        startAt: startAt, endAt: endAt, partySize: partySize, status: status ?? this.status,
        note: note, createdAt: createdAt, serviceName: serviceName, vendorName: vendorName,
      );
}

/// A traveller-contributed photo for a place (media role "contrib" — see
/// GET /places/{id}/photos, which returns full Media rows server-side).
/// Keeps just what the UI needs: `id` to delete it, `uploadedBy` to gate
/// the delete affordance to whoever posted it, `createdAt` for a relative
/// "2d ago"-style label.
class PlacePhoto {
  final String id;
  final String url;
  // A small (~320px) JPEG derivative for grid tiles — empty for photos
  // uploaded before this existed or if generation failed; fall back to
  // [url] whenever this is empty. The full-screen viewer always uses
  // [url], never this.
  final String thumbUrl;
  // Inline base64 blur placeholder ('data:image/jpeg;base64,...'), shown
  // instantly with no extra network fetch — '' under the same conditions
  // as [thumbUrl].
  final String microThumb;
  final String uploadedBy;
  final String createdAt;

  const PlacePhoto({
    required this.id,
    required this.url,
    this.thumbUrl = '',
    this.microThumb = '',
    required this.uploadedBy,
    required this.createdAt,
  });

  factory PlacePhoto.fromJson(Map<String, dynamic> json) => PlacePhoto(
        id: json['id'] as String? ?? '',
        url: json['url'] as String? ?? '',
        thumbUrl: json['thumbUrl'] as String? ?? '',
        microThumb: json['microThumb'] as String? ?? '',
        uploadedBy: json['uploadedBy'] as String? ?? '',
        createdAt: json['createdAt'] as String? ?? '',
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

/// One hour of forecast from GET /weather — see server/internal/weather
/// for the Tomorrow.io fetch and unit conversion (icon is already reduced
/// to the app's existing Sun/CloudSun/CloudRain vocabulary server-side).
class HourlyPoint {
  final DateTime time;
  final double temp;
  final String condition; // sunny | cloudy | rainy
  final String conditionLabel;
  final String icon;
  final int precipProbability;
  final double precipMm;
  final double windSpeed; // km/h
  final double uvIndex;
  final int cloudCover;
  const HourlyPoint({
    required this.time,
    required this.temp,
    required this.condition,
    required this.conditionLabel,
    required this.icon,
    required this.precipProbability,
    required this.precipMm,
    required this.windSpeed,
    required this.uvIndex,
    required this.cloudCover,
  });

  factory HourlyPoint.fromJson(Map<String, dynamic> json) => HourlyPoint(
        time: DateTime.tryParse(json['time'] as String? ?? '')?.toLocal() ?? DateTime.now(),
        temp: (json['temp'] as num?)?.toDouble() ?? 0,
        condition: json['condition'] as String? ?? 'sunny',
        conditionLabel: json['conditionLabel'] as String? ?? '',
        icon: json['icon'] as String? ?? 'Sun',
        precipProbability: (json['precipProbability'] as num?)?.toInt() ?? 0,
        precipMm: (json['precipMm'] as num?)?.toDouble() ?? 0,
        windSpeed: (json['windSpeed'] as num?)?.toDouble() ?? 0,
        uvIndex: (json['uvIndex'] as num?)?.toDouble() ?? 0,
        cloudCover: (json['cloudCover'] as num?)?.toInt() ?? 0,
      );
}

/// Current conditions — the "now" slice of GET /weather.
class WeatherNow {
  final double temp;
  final double feelsLike;
  final int humidity;
  final double windSpeed; // km/h
  final int windDirection; // degrees
  final double uvIndex;
  final double visibility; // km
  final double pressure; // hPa
  final int precipProbability;
  final int cloudCover;
  final String condition;
  final String conditionLabel;
  final String icon;
  const WeatherNow({
    required this.temp,
    required this.feelsLike,
    required this.humidity,
    required this.windSpeed,
    required this.windDirection,
    required this.uvIndex,
    required this.visibility,
    required this.pressure,
    required this.precipProbability,
    required this.cloudCover,
    required this.condition,
    required this.conditionLabel,
    required this.icon,
  });

  factory WeatherNow.fromJson(Map<String, dynamic> json) => WeatherNow(
        temp: (json['temp'] as num?)?.toDouble() ?? 0,
        feelsLike: (json['feelsLike'] as num?)?.toDouble() ?? 0,
        humidity: (json['humidity'] as num?)?.toInt() ?? 0,
        windSpeed: (json['windSpeed'] as num?)?.toDouble() ?? 0,
        windDirection: (json['windDirection'] as num?)?.toInt() ?? 0,
        uvIndex: (json['uvIndex'] as num?)?.toDouble() ?? 0,
        visibility: (json['visibility'] as num?)?.toDouble() ?? 0,
        pressure: (json['pressure'] as num?)?.toDouble() ?? 0,
        precipProbability: (json['precipProbability'] as num?)?.toInt() ?? 0,
        cloudCover: (json['cloudCover'] as num?)?.toInt() ?? 0,
        condition: json['condition'] as String? ?? 'sunny',
        conditionLabel: json['conditionLabel'] as String? ?? '',
        icon: json['icon'] as String? ?? 'Sun',
      );
}

/// Full GET /weather response — real conditions for the Weather screen and
/// the Discover-tab bar, replacing the old hardcoded weatherKinds lookup.
class WeatherSnapshot {
  final WeatherNow now;
  final List<HourlyPoint> hourly;
  final double high;
  final double low;
  final DateTime sunrise;
  final DateTime sunset;
  final DateTime fetchedAt;
  const WeatherSnapshot({
    required this.now,
    required this.hourly,
    required this.high,
    required this.low,
    required this.sunrise,
    required this.sunset,
    required this.fetchedAt,
  });

  factory WeatherSnapshot.fromJson(Map<String, dynamic> json) => WeatherSnapshot(
        now: json['now'] is Map
            ? WeatherNow.fromJson((json['now'] as Map).cast<String, dynamic>())
            : const WeatherNow(
                temp: 0,
                feelsLike: 0,
                humidity: 0,
                windSpeed: 0,
                windDirection: 0,
                uvIndex: 0,
                visibility: 0,
                pressure: 0,
                precipProbability: 0,
                cloudCover: 0,
                condition: 'sunny',
                conditionLabel: '',
                icon: 'Sun',
              ),
        hourly: [
          for (final h in (json['hourly'] as List? ?? const []))
            HourlyPoint.fromJson((h as Map).cast<String, dynamic>())
        ],
        high: (json['high'] as num?)?.toDouble() ?? 0,
        low: (json['low'] as num?)?.toDouble() ?? 0,
        sunrise: DateTime.tryParse(json['sunrise'] as String? ?? '')?.toLocal() ?? DateTime.now(),
        sunset: DateTime.tryParse(json['sunset'] as String? ?? '')?.toLocal() ?? DateTime.now(),
        fetchedAt: DateTime.tryParse(json['fetchedAt'] as String? ?? '')?.toLocal() ?? DateTime.now(),
      );
}

/// A location the traveller has pinned on the Weather screen to swipe to —
/// e.g. checking Arugam Bay's surf weather while physically in Colombo.
/// Persisted locally as spotto_weather_pins; lat/lng come from a real
/// place's own coordinates (see weather_screen.dart's region search), not
/// a separate geocoding step.
class PinnedLocation {
  final String id;
  final String name;
  final double lat;
  final double lng;
  const PinnedLocation({required this.id, required this.name, required this.lat, required this.lng});

  Map<String, dynamic> toJson() => {'id': id, 'name': name, 'lat': lat, 'lng': lng};
  factory PinnedLocation.fromJson(Map<String, dynamic> json) => PinnedLocation(
        id: json['id'] as String? ?? '',
        name: json['name'] as String? ?? '',
        lat: (json['lat'] as num?)?.toDouble() ?? 0,
        lng: (json['lng'] as num?)?.toDouble() ?? 0,
      );
}

class TimeSlot {
  final String id; // morning/afternoon/evening
  final String label;
  final String icon;
  final String range;
  const TimeSlot({required this.id, required this.label, required this.icon, required this.range});
}

/// Onboarding result: travel styles (multi-select) + interests. Persisted
/// as spotto_prefs.
class UserPrefs {
  final List<String> styles;
  final List<String> interests;
  const UserPrefs({required this.styles, required this.interests});

  Map<String, dynamic> toJson() => {'styles': styles, 'interests': interests};

  /// Reads the current {'styles': [...]} shape, but falls back to the old
  /// single {'style': '...'} shape so a cached spotto_prefs from before
  /// multi-select still loads instead of throwing.
  factory UserPrefs.fromJson(Map<String, dynamic> json) => UserPrefs(
        styles: json['styles'] is List
            ? (json['styles'] as List).cast<String>()
            : [
                if (json['style'] is String && (json['style'] as String).isNotEmpty)
                  json['style'] as String
              ],
        interests: (json['interests'] as List? ?? const []).cast<String>(),
      );
}
