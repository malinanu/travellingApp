// The prototype's data uses Lucide icon names as strings throughout.
// This single mapping resolves them to Material icons.

import 'package:flutter/material.dart';

const _iconMap = <String, IconData>{
  'Laptop': Icons.laptop_mac,
  'Waves': Icons.waves,
  'Gem': Icons.diamond_outlined,
  'Backpack': Icons.backpack_outlined,
  'Coffee': Icons.local_cafe,
  'Martini': Icons.local_bar,
  'Flower2': Icons.local_florist,
  'UtensilsCrossed': Icons.restaurant,
  'Utensils': Icons.restaurant_menu,
  'Palmtree': Icons.beach_access,
  'Music': Icons.music_note,
  'EggFried': Icons.egg_alt,
  'Sandwich': Icons.lunch_dining,
  'Soup': Icons.ramen_dining,
  'Fish': Icons.set_meal,
  'GlassWater': Icons.local_drink,
  'Wine': Icons.wine_bar,
  'Flame': Icons.local_fire_department,
  'Cookie': Icons.cookie_outlined,
  'Palette': Icons.palette_outlined,
  'Brush': Icons.brush,
  'Sunrise': Icons.wb_sunny_outlined,
  'Sun': Icons.wb_sunny,
  'Sunset': Icons.wb_twilight,
  'CloudSun': Icons.wb_cloudy,
  'CloudRain': Icons.water_drop,
  'Volume2': Icons.volume_up,
  'Wind': Icons.air,
  'BedDouble': Icons.king_bed,
  'Mountain': Icons.landscape,
  'ShoppingBasket': Icons.shopping_basket,
  'Footprints': Icons.directions_walk,
  'CarTaxiFront': Icons.local_taxi,
  'Store': Icons.storefront,
  'Tag': Icons.local_offer,
  'Ticket': Icons.confirmation_number,
  'Eye': Icons.visibility,
  'Percent': Icons.percent,
  'BadgeDollarSign': Icons.attach_money,
  'Gift': Icons.card_giftcard,
  'Package': Icons.inventory_2,
  'MapPin': Icons.place,
  'Star': Icons.star,
  'Megaphone': Icons.campaign,
  'Sparkles': Icons.auto_awesome,
  'Wifi': Icons.wifi,
  'Clock': Icons.schedule,
  'Users': Icons.group,
  'Search': Icons.search,
  'Navigation': Icons.near_me,
  'Bookmark': Icons.bookmark_outline,
  'QrCode': Icons.qr_code_2,
  'MessageCircle': Icons.chat_bubble_outline,
  'Compass': Icons.explore,
  'Map': Icons.map,
  'Route': Icons.route,
  'Bell': Icons.notifications_none,
  'Settings': Icons.settings_outlined,
  'LogOut': Icons.logout,
  'Camera': Icons.photo_camera_outlined,
  'Check': Icons.check,
  'Plus': Icons.add,
  'X': Icons.close,
  // Additional common Lucide icons from server seeds
  'ShoppingBag': Icons.shopping_bag,
  'Truck': Icons.local_shipping,
  'TreePine': Icons.park,
  'Dumbbell': Icons.fitness_center,
  'Mic': Icons.mic,
  'Film': Icons.movie,
  'Landmark': Icons.account_balance,
  'Library': Icons.local_library,
  'ShoppingCart': Icons.shopping_cart,
  'Car': Icons.directions_car,
  'Bike': Icons.pedal_bike,
  'Banknote': Icons.payments,
  'Scissors': Icons.content_cut,
  'Key': Icons.vpn_key,
  'Droplets': Icons.water_drop,
  'Fuel': Icons.local_gas_station,
  'Stethoscope': Icons.medical_services,
  'Mail': Icons.mail,
  'SquareParking': Icons.local_parking,
  'Pill': Icons.medication,
  // Vendor-category taxonomy icons (see data/vendor_categories.dart) not
  // already covered above.
  'Building2': Icons.apartment,
  'BedSingle': Icons.bed,
  'Anchor': Icons.anchor,
  'ChefHat': Icons.local_dining,
  'Hammer': Icons.handyman,
  'Shirt': Icons.checkroom,
  'Leaf': Icons.eco,
  'PawPrint': Icons.pets,
  'Ship': Icons.directions_boat,
  'PartyPopper': Icons.celebration,
  'HeartHandshake': Icons.handshake,
  'Plane': Icons.flight,
  // Added to differentiate categories that previously shared an icon with
  // an unrelated category (see spotto-icons-taxonomy-cleanup memory).
  'Zap': Icons.ev_station,
  'Trophy': Icons.emoji_events,
  'Tv': Icons.tv,
  'Home': Icons.home,
  'WashingMachine': Icons.local_laundry_service,
  'Image': Icons.image_outlined,
  'Umbrella': Icons.umbrella,
  'Medal': Icons.military_tech,
  'Ruler': Icons.straighten,
  'Sailboat': Icons.sailing,
};

// Lowercased once at first use (not per spottoIcon call) so exact-name
// lookup stays O(1) instead of scanning+lowercasing all of _iconMap on
// every call — this runs per place on every map/list render.
final Map<String, IconData> _iconMapLower =
    _iconMap.map((key, value) => MapEntry(key.toLowerCase(), value));

IconData spottoIcon(String name) {
  final norm = name.toLowerCase().trim();
  final exact = _iconMapLower[norm];
  if (exact != null) return exact;

  // Robust fallbacks if the user manually entered a generic name in the admin panel
  if (norm.contains('cafe') || norm.contains('coffee')) return Icons.local_cafe;
  if (norm.contains('restaurant') || norm.contains('food') || norm.contains('dining') || norm.contains('meal') || norm.contains('eat')) return Icons.restaurant;
  if (norm.contains('hotel') || norm.contains('stay') || norm.contains('bed') || norm.contains('hostel') || norm.contains('resort')) return Icons.hotel;
  if (norm.contains('bar') || norm.contains('pub') || norm.contains('drink') || norm.contains('wine') || norm.contains('beer') || norm.contains('martini') || norm.contains('club')) return Icons.local_bar;
  if (norm.contains('surf') || norm.contains('wave') || norm.contains('beach') || norm.contains('sea') || norm.contains('boat') || norm.contains('water') || norm.contains('dive')) return Icons.waves;
  if (norm.contains('gym') || norm.contains('fitness') || norm.contains('workout') || norm.contains('sport')) return Icons.fitness_center;
  if (norm.contains('art') || norm.contains('paint') || norm.contains('palette') || norm.contains('craft') || norm.contains('pottery') || norm.contains('batik')) return Icons.palette;
  if (norm.contains('shop') || norm.contains('store') || norm.contains('market') || norm.contains('retail') || norm.contains('grocery') || norm.contains('clothing')) return Icons.storefront;
  if (norm.contains('music') || norm.contains('dj')) return Icons.music_note;
  if (norm.contains('park') || norm.contains('tree') || norm.contains('garden') || norm.contains('nature') || norm.contains('safari')) return Icons.park;
  if (norm.contains('tour') || norm.contains('guide') || norm.contains('trip') || norm.contains('explore')) return Icons.explore;
  if (norm.contains('yoga') || norm.contains('wellness') || norm.contains('spa')) return Icons.spa;
  if (norm.contains('charg') || norm.contains('ev station')) return Icons.ev_station;
  if (norm.contains('laundry') || norm.contains('dry clean')) return Icons.local_laundry_service;
  if (norm.contains('electronic') || norm.contains('tv') || norm.contains('television')) return Icons.tv;
  if (norm.contains('trophy') || norm.contains('medal') || norm.contains('award')) return Icons.emoji_events;

  return Icons.circle_outlined;
}
