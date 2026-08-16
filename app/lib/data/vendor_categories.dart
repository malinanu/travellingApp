// Canonical vendor-category taxonomy, mirroring
// server/internal/domain/vendor_category.go and
// web/packages/shared/src/vendor-categories.ts exactly (id/label/segment/
// icon name/relevant vibe dimensions) — keep all three in sync if any
// change. Replaces what used to be a totally decoupled, hand-rolled 4-bucket
// filter taxonomy in the Discover screen.

enum VendorSegment { foodAndDrink, thingsToDo, shopping, services, brand }

class VendorSegmentInfo {
  final VendorSegment id;
  final String label;
  const VendorSegmentInfo(this.id, this.label);
}

// "future" (accommodation/flight/insurance/transfer/booking platforms) is
// deliberately excluded — affiliate-only per the business plan, not a real
// vendor category places can carry today.
const vendorSegments = <VendorSegmentInfo>[
  VendorSegmentInfo(VendorSegment.foodAndDrink, 'Food & Drink'),
  VendorSegmentInfo(VendorSegment.thingsToDo, 'Things to do'),
  VendorSegmentInfo(VendorSegment.shopping, 'Shopping'),
  VendorSegmentInfo(VendorSegment.services, 'Services'),
  VendorSegmentInfo(VendorSegment.brand, 'Brand Partner'),
];

/// Which real-booking flow a category's vendors use, if any — null means
/// not bookable (walk-in businesses, brand partners, event categories —
/// Events isn't a shipped feature yet). Mirrors
/// server/internal/domain/vendor_category.go's BookingType exactly.
enum BookingType { session, table, stay, rental }

class VendorCategory {
  final String id;
  final String label;
  final VendorSegment segment;
  final String icon; // Lucide/spottoIcon name, see core/widgets/spotto_icons.dart
  final List<String> relevantVibeDimensions; // subset of productive/loud/crowded
  final BookingType? bookingType;
  
  const VendorCategory({
    required this.id,
    required this.label,
    required this.segment,
    required this.icon,
    this.relevantVibeDimensions = const [],
    this.bookingType,
  });

  factory VendorCategory.fromJson(Map<String, dynamic> json) {
    return VendorCategory(
      id: json['id'] as String? ?? '',
      label: json['label'] as String? ?? '',
      segment: _parseSegment(json['segment'] as String?),
      icon: json['icon'] as String? ?? 'MapPin',
      relevantVibeDimensions: (json['relevantVibeDimensions'] as List?)?.cast<String>() ?? [],
      bookingType: _parseBookingType(json['bookingType'] as String?),
    );
  }

  static VendorSegment _parseSegment(String? val) {
    switch (val) {
      case 'food_and_drink': return VendorSegment.foodAndDrink;
      case 'things_to_do': return VendorSegment.thingsToDo;
      case 'shopping': return VendorSegment.shopping;
      case 'services': return VendorSegment.services;
      case 'brand': return VendorSegment.brand;
      default: return VendorSegment.brand;
    }
  }

  static BookingType? _parseBookingType(String? val) {
    if (val == null || val.isEmpty) return null;
    return BookingType.values.firstWhere((e) => e.name == val, orElse: () => BookingType.session);
  }
}

// Default fallback categories. Replaced dynamically by CategoriesNotifier on startup.
var vendorCategories = <VendorCategory>[
  VendorCategory(id: 'restaurants', label: 'Restaurants', segment: VendorSegment.foodAndDrink, icon: 'UtensilsCrossed', relevantVibeDimensions: ['loud', 'crowded'], bookingType: BookingType.table),
  VendorCategory(id: 'coffee', label: 'Coffee', segment: VendorSegment.foodAndDrink, icon: 'Coffee', relevantVibeDimensions: ['productive', 'loud', 'crowded'], bookingType: BookingType.table),
  VendorCategory(id: 'hotels', label: 'Hotels', segment: VendorSegment.services, icon: 'Building2', relevantVibeDimensions: ['crowded'], bookingType: BookingType.stay),
];

VendorCategory? categoryById(String id) {
  final norm = id.toLowerCase().trim();
  for (final c in vendorCategories) {
    if (c.id.toLowerCase() == norm) return c;
  }
  return null;
}

List<VendorCategory> categoriesBySegment(VendorSegment segment) =>
    [for (final c in vendorCategories) if (c.segment == segment) c];

/// Which of "productive"/"loud"/"crowded" are meaningful to show/score for
/// a place's category (e.g. "productive" doesn't mean anything for a beach
/// bar). Falls back to all 3 for an unknown/blank category id.
List<String> relevantVibeDimensions(String categoryId) {
  final c = categoryById(categoryId);
  if (c != null && c.relevantVibeDimensions.isNotEmpty) return c.relevantVibeDimensions;
  return const ['productive', 'loud', 'crowded'];
}

/// Whether this category's vendors can take real bookings through the
/// platform, and which flow — null means not bookable.
BookingType? bookingTypeFor(String categoryId) => categoryById(categoryId)?.bookingType;
