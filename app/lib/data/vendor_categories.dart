// Canonical vendor-category taxonomy, mirroring
// server/internal/domain/vendor_category.go and
// web/packages/shared/src/vendor-categories.ts exactly (id/label/segment/
// icon name/relevant vibe dimensions) — keep all three in sync if any
// change. Replaces what used to be a totally decoupled, hand-rolled 4-bucket
// filter taxonomy in the Discover screen.

enum VendorSegment { hospitality, activity, independent, events, brand }

class VendorSegmentInfo {
  final VendorSegment id;
  final String label;
  const VendorSegmentInfo(this.id, this.label);
}

// "future" (accommodation/flight/insurance/transfer/booking platforms) is
// deliberately excluded — affiliate-only per the business plan, not a real
// vendor category places can carry today.
const vendorSegments = <VendorSegmentInfo>[
  VendorSegmentInfo(VendorSegment.hospitality, 'Hospitality'),
  VendorSegmentInfo(VendorSegment.activity, 'Activities & Experiences'),
  VendorSegmentInfo(VendorSegment.independent, 'Independent & Local'),
  VendorSegmentInfo(VendorSegment.events, 'Events'),
  VendorSegmentInfo(VendorSegment.brand, 'Brand Partner'),
];

class VendorCategory {
  final String id;
  final String label;
  final VendorSegment segment;
  final String icon; // Lucide/spottoIcon name, see core/widgets/spotto_icons.dart
  final List<String> relevantVibeDimensions; // subset of productive/loud/crowded
  const VendorCategory({
    required this.id,
    required this.label,
    required this.segment,
    required this.icon,
    this.relevantVibeDimensions = const [],
  });
}

const vendorCategories = <VendorCategory>[
  // Hospitality vendors — cafes, restaurants, bars, hotels, beach clubs.
  VendorCategory(id: 'cafe', label: 'Café', segment: VendorSegment.hospitality, icon: 'Coffee', relevantVibeDimensions: ['productive', 'loud', 'crowded']),
  VendorCategory(id: 'coworking-cafe', label: 'Coworking Café', segment: VendorSegment.hospitality, icon: 'Laptop', relevantVibeDimensions: ['productive', 'loud', 'crowded']),
  VendorCategory(id: 'restaurant', label: 'Restaurant', segment: VendorSegment.hospitality, icon: 'UtensilsCrossed', relevantVibeDimensions: ['loud', 'crowded']),
  VendorCategory(id: 'street-food', label: 'Street Food', segment: VendorSegment.hospitality, icon: 'Soup', relevantVibeDimensions: ['crowded']),
  VendorCategory(id: 'bakery', label: 'Bakery', segment: VendorSegment.hospitality, icon: 'Cookie', relevantVibeDimensions: ['crowded']),
  VendorCategory(id: 'bar-pub', label: 'Bar & Pub', segment: VendorSegment.hospitality, icon: 'Martini', relevantVibeDimensions: ['loud', 'crowded']),
  VendorCategory(id: 'club', label: 'Club / Nightlife', segment: VendorSegment.hospitality, icon: 'Music', relevantVibeDimensions: ['loud', 'crowded']),
  VendorCategory(id: 'hotel', label: 'Hotel', segment: VendorSegment.hospitality, icon: 'Building2', relevantVibeDimensions: ['crowded']),
  VendorCategory(id: 'hostel', label: 'Hostel / Guesthouse', segment: VendorSegment.hospitality, icon: 'BedSingle', relevantVibeDimensions: ['loud', 'crowded']),
  VendorCategory(id: 'resort', label: 'Resort', segment: VendorSegment.hospitality, icon: 'Palmtree', relevantVibeDimensions: ['crowded']),
  VendorCategory(id: 'beach-club', label: 'Beach Club', segment: VendorSegment.hospitality, icon: 'Waves', relevantVibeDimensions: ['loud', 'crowded']),

  // Activity and experience providers — surf/dive schools, yoga, tours.
  VendorCategory(id: 'surf-school', label: 'Surf School', segment: VendorSegment.activity, icon: 'Waves', relevantVibeDimensions: ['crowded']),
  VendorCategory(id: 'dive-school', label: 'Diving / Dive Shop', segment: VendorSegment.activity, icon: 'Anchor', relevantVibeDimensions: ['crowded']),
  VendorCategory(id: 'yoga-studio', label: 'Yoga Studio', segment: VendorSegment.activity, icon: 'Flower2', relevantVibeDimensions: ['loud', 'crowded']),
  VendorCategory(id: 'spa-wellness', label: 'Spa & Wellness', segment: VendorSegment.activity, icon: 'Sparkles', relevantVibeDimensions: ['loud', 'crowded']),
  VendorCategory(id: 'cooking-class', label: 'Cooking Class', segment: VendorSegment.activity, icon: 'ChefHat', relevantVibeDimensions: ['crowded']),
  VendorCategory(id: 'pottery-workshop', label: 'Pottery Workshop', segment: VendorSegment.activity, icon: 'Hammer', relevantVibeDimensions: ['crowded']),
  VendorCategory(id: 'batik-atelier', label: 'Batik / Textile Atelier', segment: VendorSegment.activity, icon: 'Shirt', relevantVibeDimensions: ['crowded']),
  VendorCategory(id: 'spice-trail', label: 'Spice Trail', segment: VendorSegment.activity, icon: 'Leaf', relevantVibeDimensions: ['crowded']),
  VendorCategory(id: 'safari-wildlife', label: 'Safari & Wildlife', segment: VendorSegment.activity, icon: 'PawPrint', relevantVibeDimensions: ['crowded']),
  VendorCategory(id: 'boat-trip', label: 'Boat Trip', segment: VendorSegment.activity, icon: 'Ship', relevantVibeDimensions: ['crowded']),
  VendorCategory(id: 'city-tour', label: 'City / Cultural Tour', segment: VendorSegment.activity, icon: 'Compass', relevantVibeDimensions: ['crowded']),

  // Independent local providers — guides, artisans, tailors, solo hosts.
  VendorCategory(id: 'tour-guide', label: 'Tour / Walking Guide', segment: VendorSegment.independent, icon: 'Compass', relevantVibeDimensions: ['crowded']),
  VendorCategory(id: 'artisan-shop', label: 'Local Artisan / Craft Shop', segment: VendorSegment.independent, icon: 'Palette', relevantVibeDimensions: ['crowded']),
  VendorCategory(id: 'tailor-clothing', label: 'Tailor / Local Clothing', segment: VendorSegment.independent, icon: 'Shirt', relevantVibeDimensions: ['crowded']),
  VendorCategory(id: 'experience-host', label: 'Independent Experience Host', segment: VendorSegment.independent, icon: 'Backpack', relevantVibeDimensions: ['crowded']),

  // Event promoters — festivals, parties, competitions, community events.
  VendorCategory(id: 'festival-event', label: 'Festival', segment: VendorSegment.events, icon: 'PartyPopper', relevantVibeDimensions: ['loud', 'crowded']),
  VendorCategory(id: 'party-promoter', label: 'Party Promotion', segment: VendorSegment.events, icon: 'PartyPopper', relevantVibeDimensions: ['loud', 'crowded']),
  VendorCategory(id: 'surf-competition', label: 'Surf Competition', segment: VendorSegment.events, icon: 'Waves', relevantVibeDimensions: ['loud', 'crowded']),
  VendorCategory(id: 'community-charity-event', label: 'Community / Charity Event', segment: VendorSegment.events, icon: 'HeartHandshake', relevantVibeDimensions: ['crowded']),
  VendorCategory(id: 'music-night', label: 'Music Night', segment: VendorSegment.events, icon: 'Music', relevantVibeDimensions: ['loud', 'crowded']),

  // Brand partners — sponsors, not physical venues, so vibe is irrelevant.
  VendorCategory(id: 'beverage-brand', label: 'Beverage Brand', segment: VendorSegment.brand, icon: 'GlassWater'),
  VendorCategory(id: 'alcohol-brand', label: 'Beer & Alcohol Brand', segment: VendorSegment.brand, icon: 'Wine'),
  VendorCategory(id: 'food-brand', label: 'Food Brand', segment: VendorSegment.brand, icon: 'UtensilsCrossed'),
  VendorCategory(id: 'lifestyle-brand', label: 'Lifestyle Brand', segment: VendorSegment.brand, icon: 'Sparkles'),
  VendorCategory(id: 'tourism-company', label: 'Tourism Company', segment: VendorSegment.brand, icon: 'Plane'),

  // "future" segment deliberately has no categories — see vendorSegments.
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
