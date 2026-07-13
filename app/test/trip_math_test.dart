import 'package:flutter_test/flutter_test.dart';
import 'package:spotto/data/mock/trip_catalog.dart';

TripCatalogItem _byId(String id) => tripCatalog.firstWhere((p) => p.id == id);

void main() {
  group('tripLeg', () {
    test('short hops are walking legs', () {
      final leg = tripLeg(_byId('bayvilla'), _byId('driftwood'));
      expect(leg.mode, 'Walk');
      expect(leg.km, lessThanOrEqualTo(1.2));
      expect(leg.mins, greaterThanOrEqualTo(1));
      expect(leg.gmaps, contains('travelmode=walking'));
    });

    test('long hops are tuk-tuk legs', () {
      final leg = tripLeg(_byId('driftwood'), _byId('coconuthill'));
      expect(leg.mode, 'Tuk-tuk');
      expect(leg.km, greaterThan(1.2));
      expect(leg.gmaps, contains('travelmode=driving'));
    });

    test('haversine matches known distance order of magnitude', () {
      // Weligama to Mirissa is roughly 5-6 km as the crow flies.
      final km = tripHaversineKm(_byId('welibeach'), _byId('mirissa'));
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
      expect(tripDayGmaps([tripSeed.days[0].stops.first]), isNull);
      final url = tripDayGmaps(tripSeed.days[0].stops);
      expect(url, contains('waypoints='));
      expect(url, contains('travelmode=driving'));
    });
  });
}
