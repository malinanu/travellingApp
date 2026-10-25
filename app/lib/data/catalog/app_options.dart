// Fixed app options: the travel styles, interests, time-of-day slots and
// vibe tags the app offers. Places, travellers, groups and trips all come
// from the API.

import '../models/models.dart';

const travelStyles = [
  TravelStyle(id: 'nomad', label: 'Digital Nomad', icon: 'Laptop', blurb: 'Fast Wi-Fi, good coffee, places to focus.'),
  TravelStyle(id: 'surf', label: 'Surf Traveler', icon: 'Waves', blurb: 'Breaks, board rentals, dawn patrol.'),
  TravelStyle(id: 'luxury', label: 'Luxury', icon: 'Gem', blurb: 'Refined stays, fine dining, sunset views.'),
  TravelStyle(id: 'backpacker', label: 'Backpacker', icon: 'Backpack', blurb: 'Cheap eats, hostels, good company.'),
];

const interests = [
  Interest(id: 'coffee', label: 'Coffee', icon: 'Coffee'),
  Interest(id: 'coworking', label: 'Coworking', icon: 'Laptop'),
  Interest(id: 'nightlife', label: 'Nightlife', icon: 'Martini'),
  Interest(id: 'surf', label: 'Surf', icon: 'Waves'),
  Interest(id: 'yoga', label: 'Yoga', icon: 'Flower2'),
  Interest(id: 'food', label: 'Local Food', icon: 'UtensilsCrossed'),
  Interest(id: 'beach', label: 'Beaches', icon: 'Palmtree'),
  Interest(id: 'music', label: 'Live Music', icon: 'Music'),
  Interest(id: 'hiking', label: 'Hiking', icon: 'Mountain'),
  Interest(id: 'vegan', label: 'Vegan / Plant-based', icon: 'Flower2'),
  Interest(id: 'halal', label: 'Halal Food', icon: 'UtensilsCrossed'),
  Interest(id: 'gluten-free', label: 'Gluten-Free', icon: 'UtensilsCrossed'),
  Interest(id: 'wifi', label: 'Fast Wi-Fi', icon: 'Wifi'),
  Interest(id: 'outlets', label: 'Power Outlets', icon: 'Zap'),
  Interest(id: 'quiet', label: 'Quiet Zone', icon: 'Volume2'),
];

const timeSlots = [
  TimeSlot(id: 'morning', label: 'Morning', icon: 'Sunrise', range: '6–11 AM'),
  TimeSlot(id: 'afternoon', label: 'Afternoon', icon: 'Sun', range: '11–4 PM'),
  TimeSlot(id: 'evening', label: 'Evening', icon: 'Sunset', range: '4–10 PM'),
];

const vibes = [
  VibeTag(id: 'productive', label: 'Productive', icon: 'Laptop', color: 0xFF0B6F86),
  VibeTag(id: 'loud', label: 'Loud', icon: 'Volume2', color: 0xFFF0682F),
  VibeTag(id: 'crowded', label: 'Crowded', icon: 'Users', color: 0xFF6C8582),
];
