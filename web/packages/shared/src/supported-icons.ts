// Lucide icon names the Flutter app actually knows how to render, mirrored
// from app/lib/core/widgets/spotto_icons.dart's `_iconMap`. A vendor
// category's `icon` must be one of these — anything else silently falls
// back to a generic dot on mobile. Keep in sync with that file: add here
// whenever a name is added there, and vice versa.
export const SUPPORTED_ICON_NAMES = [
  'Laptop', 'Waves', 'Gem', 'Backpack', 'Coffee', 'Martini', 'Flower2',
  'UtensilsCrossed', 'Utensils', 'Palmtree', 'Music', 'EggFried', 'Sandwich',
  'Soup', 'Fish', 'GlassWater', 'Wine', 'Flame', 'Cookie', 'Palette', 'Brush',
  'Sunrise', 'Sun', 'Sunset', 'CloudSun', 'CloudRain', 'Volume2', 'Wind',
  'BedDouble', 'Mountain', 'ShoppingBasket', 'Footprints', 'CarTaxiFront',
  'Store', 'Tag', 'Ticket', 'Eye', 'Percent', 'BadgeDollarSign', 'Gift',
  'Package', 'MapPin', 'Star', 'Megaphone', 'Sparkles', 'Wifi', 'Clock',
  'Users', 'Search', 'Navigation', 'Bookmark', 'QrCode', 'MessageCircle',
  'Compass', 'Map', 'Route', 'Bell', 'Settings', 'LogOut', 'Camera', 'Check',
  'Plus', 'X', 'ShoppingBag', 'Truck', 'TreePine', 'Dumbbell', 'Mic', 'Film',
  'Landmark', 'Library', 'ShoppingCart', 'Car', 'Bike', 'Banknote',
  'Scissors', 'Key', 'Droplets', 'Fuel', 'Stethoscope', 'Mail',
  'SquareParking', 'Pill', 'Building2', 'BedSingle', 'Anchor', 'ChefHat',
  'Hammer', 'Shirt', 'Leaf', 'PawPrint', 'Ship', 'PartyPopper',
  'HeartHandshake', 'Plane', 'Zap', 'Trophy', 'Tv', 'Home', 'WashingMachine',
  'Image', 'Umbrella', 'Medal', 'Ruler', 'Sailboat',
] as const;

export type SupportedIconName = (typeof SUPPORTED_ICON_NAMES)[number];
