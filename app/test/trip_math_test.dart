import 'package:flutter_test/flutter_test.dart';
import 'package:spotto/data/catalog/trip_catalog.dart';

// Test-only stops at real south-coast coordinates, so distances are
// realistic.
const _hotel = TripCatalogItem(id: 'hotel', name: 'Bay Hotel', type: 'Hotel', icon: 'BedDouble', lat: 5.9731, lng: 80.4254, cost: 9500, styles: ['luxury'], hint: '14:00');
const _cafe = TripCatalogItem(id: 'cafe', name: 'Beach Café', type: 'Café', icon: 'Coffee', lat: 5.9762, lng: 80.4221, cost: 1200, styles: ['nomad'], hint: '8:00');
const _viewpoint = TripCatalogItem(id: 'viewpoint', name: 'Coconut Tree Hill', type: 'Viewpoint', icon: 'Mountain', lat: 5.9417, lng: 80.4632, cost: 0, styles: ['backpacker'], hint: '17:00');
const _weligamaBeach = TripCatalogItem(id: 'weligama-beach', name: 'Weligama Beach', type: 'Beach', icon: 'Palmtree', lat: 5.9722, lng: 80.4286, cost: 0, styles: ['surf'], hint: '7:00');
const _mirissaBeach = TripCatalogItem(id: 'mirissa-beach', name: 'Mirissa Beach', type: 'Beach', icon: 'Palmtree', lat: 5.9445, lng: 80.4595, cost: 0, styles: ['surf'], hint: '15:00');

void main() {
  setUp(() => tripCatalog = [_hotel, _cafe, _viewpoint]);
  tearDown(() => tripCatalog = []);

  group('tripLeg', () {
    test('short hops are walking legs', () {
      final leg = tripLeg(_hotel, _cafe);
      expect(leg.mode, 'Walk');
      expect(leg.km, lessThanOrEqualTo(1.2));
      expect(leg.mins, greaterThanOrEqualTo(1));
      expect(leg.gmaps, contains('travelmode=walking'));
    });

    test('long hops are tuk-tuk legs', () {
      final leg = tripLeg(_cafe, _viewpoint);
      expect(leg.mode, 'Tuk-tuk');
      expect(leg.km, greaterThan(1.2));
      expect(leg.gmaps, contains('travelmode=driving'));
    });

    test('haversine matches known distance order of magnitude', () {
      // Weligama to Mirissa is roughly 5-6 km as the crow flies.
      final km = tripHaversineKm(_weligamaBeach, _mirissaBeach);
      expect(km, inInclusiveRange(3.0, 8.0));
    });
  });

  group('formatting', () {
    test('tripFmtTime formats minutes since midnight', () {
      expect(tripFmtTime(0), '12:00 AM');
      expect(tripFmtTime(14 * 60), '2:00 PM');
      expect(tripFmtTime(17 * 60 + 30), '5:30 PM');
      expect(tripFmtTime(12 * 60), '12:00 PM');
    });

    test('tripFmtRs formats rupees with Free for zero', () {
      expect(tripFmtRs(0), 'Free');
      expect(tripFmtRs(9500), 'Rs 9,500');
    });
  });

  group('day gmaps url', () {
    test('needs at least two resolvable stops', () {
      const stops = [
        TripStop(id: 'a', placeId: 'hotel', time: 14 * 60),
        TripStop(id: 'b', placeId: 'cafe', time: 16 * 60),
        TripStop(id: 'c', placeId: 'viewpoint', time: 17 * 60),
      ];
      expect(tripDayGmaps([stops.first]), isNull);
      final url = tripDayGmaps(stops);
      expect(url, contains('waypoints='));
      expect(url, contains('travelmode=driving'));
    });
  });

  group('old sample trip', () {
    test('loses its made-up stops, name and collaborators, keeps the rest', () {
      const sample = Trip(
        name: 'Weligama Weekend',
        code: 'WLG-7QF',
        startDate: '2026-06-20',
        collaborators: ['theo', 'mia', 'usr_real'],
        days: [
          TripDay(id: 'd1', stops: [
            TripStop(id: 's1', placeId: 'bayvilla', time: 14 * 60),
            TripStop(id: 's101_42', placeId: 'hotel', time: 15 * 60, by: 'me'),
          ]),
          TripDay(id: 'd2', stops: [
            TripStop(id: 's5', placeId: 'welibeach', time: 7 * 60),
            TripStop(id: 's6', placeId: '', time: 9 * 60, custom: CustomStopInfo(name: 'My own stop')),
          ]),
        ],
      );

      final cleaned = cleanLegacySampleTrip(sample);
      expect(cleaned.name, 'Spotto Trip');
      expect(cleaned.code, 'WLG-7QF');
      expect(cleaned.collaborators, ['usr_real']);
      expect(cleaned.days[0].stops.map((s) => s.id), ['s101_42']);
      expect(cleaned.days[1].stops.map((s) => s.id), ['s6']);
    });

    test('a renamed sample keeps its name, other trips are untouched', () {
      const renamed = Trip(name: 'Our south coast trip', code: 'WLG-7QF', startDate: '2026-06-20', collaborators: [], days: []);
      expect(cleanLegacySampleTrip(renamed).name, 'Our south coast trip');

      const other = Trip(
        name: 'Weligama Weekend',
        code: 'TRP-ABC',
        startDate: '2026-06-20',
        collaborators: ['theo'],
        days: [TripDay(id: 'd1', stops: [TripStop(id: 's1', placeId: 'hotel', time: 600)])],
      );
      expect(identical(cleanLegacySampleTrip(other), other), isTrue);
    });
  });
}
