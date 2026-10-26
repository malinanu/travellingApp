// Turns a published itinerary (a Stories post) into a trip the traveller
// can then edit in the Trip tab.

import '../models/content.dart';
import 'trip_catalog.dart';

/// A new trip with one day per itinerary day and the same stops.
///
/// Every stop carries its own name and coordinates ([CustomStopInfo]):
/// the trip screen only shows a plain place-id stop when that place is in
/// the traveller's loaded catalog, which a post's places often aren't.
/// Coordinates come from the stop itself, else its linked Spotto place,
/// else the post's town. Stops without a time follow the previous one by
/// 90 minutes, starting at 9:00.
Trip tripFromItinerary(ContentItem item, {required String name, required String startDate}) {
  final days = <TripDay>[];
  for (final day in item.days) {
    final stops = <TripStop>[];
    var next = 9 * 60;
    for (final s in day.stops) {
      final time = (s.minutes ?? next).clamp(0, 23 * 60 + 30);
      next = (time + 90).clamp(0, 23 * 60 + 30);
      final place = s.placeId.isEmpty ? null : item.place(s.placeId);
      final lat = s.lat ?? place?.lat ?? item.regionLat;
      final lng = s.lng ?? place?.lng ?? item.regionLng;
      final custom = CustomStopInfo(
        name: s.name.isNotEmpty ? s.name : (place?.name ?? 'Stop'),
        type: place?.category.isNotEmpty == true ? place!.category : 'Itinerary stop',
        icon: 'MapPin',
      );
      stops.add(TripStop(
        id: tripNewId('s'),
        placeId: s.placeId,
        time: time,
        by: 'me',
        custom: lat == null || lng == null
            ? custom
            : CustomStopInfo(name: custom.name, type: custom.type, icon: custom.icon, lat: lat, lng: lng),
      ));
    }
    days.add(TripDay(id: tripNewId('d'), stops: stops));
  }
  if (days.isEmpty) days.add(TripDay(id: tripNewId('d'), stops: const []));
  final trimmed = name.trim();
  return Trip(
    name: trimmed.isEmpty ? (item.title.isEmpty ? 'Spotto Trip' : item.title) : trimmed,
    code: newTripCode(),
    startDate: startDate,
    collaborators: const [],
    days: days,
  );
}
