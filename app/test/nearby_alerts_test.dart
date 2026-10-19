// Nearby saved-place alerts: which places the phone watches (nearest first,
// within the platform's region limit), how often alerts may fire, and what
// they say.

import 'package:flutter/foundation.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:spotto/core/location/nearby_alerts.dart';

NearbySpot spot(String id, double lat, double lng, {String kind = 'see', String source = ''}) =>
    NearbySpot(id: id, name: 'Place $id', kind: kind, lat: lat, lng: lng, source: source);

void main() {
  group('spotsToWatch', () {
    // Around Phuket: Kata (7.82, 98.30), Patong (7.89, 98.29), Old Town (7.88, 98.39).
    final kata = spot('kata', 7.8206, 98.2985);
    final patong = spot('patong', 7.8961, 98.2966);
    final oldTown = spot('old', 7.8847, 98.3923);
    final bangkok = spot('bkk', 13.7563, 100.5018);

    test('picks the nearest places to the traveller', () {
      final picked = spotsToWatch([bangkok, oldTown, kata, patong], lat: 7.8210, lng: 98.2990, max: 2);
      expect(picked.map((s) => s.id), ['kata', 'patong']);
    });

    test('without a position keeps the saved order', () {
      expect(spotsToWatch([bangkok, kata, patong], max: 2).map((s) => s.id), ['bkk', 'kata']);
    });

    test('drops duplicates and respects the platform limit', () {
      final many = [for (var i = 0; i < 150; i++) spot('s$i', 7.8, 98.3)];
      expect(spotsToWatch([...many, many.first], max: maxNearbyRegions(TargetPlatform.android)), hasLength(90));
      expect(spotsToWatch(many, max: maxNearbyRegions(TargetPlatform.iOS)), hasLength(20));
    });
  });

  group('NearbyAlertHistory', () {
    final t0 = DateTime(2026, 10, 20, 9);

    test('one alert per place per day', () {
      final h = NearbyAlertHistory();
      expect(h.shouldAlert('kata', t0), isTrue);
      h.record('kata', t0);
      expect(h.shouldAlert('kata', t0.add(const Duration(hours: 23))), isFalse);
      expect(h.shouldAlert('kata', t0.add(const Duration(hours: 24))), isTrue);
    });

    test('at most three alerts a day overall', () {
      final h = NearbyAlertHistory();
      for (final (i, id) in ['a', 'b', 'c'].indexed) {
        h.record(id, t0.add(Duration(hours: i)));
      }
      expect(h.shouldAlert('d', t0.add(const Duration(hours: 5))), isFalse);
      expect(h.shouldAlert('d', t0.add(const Duration(hours: 24, minutes: 1))), isTrue);
    });

    test('survives a JSON round trip', () {
      final h = NearbyAlertHistory()..record('kata', t0);
      final again = NearbyAlertHistory.fromJson(h.toJson());
      expect(again.shouldAlert('kata', t0.add(const Duration(hours: 1))), isFalse);
    });
  });

  test('alert text fits the kind of place and says where it was saved from', () {
    final (title, body) = nearbyAlertText(spot('n', 7.8, 98.3, kind: 'eat', source: 'a Phuket reel'));
    expect(title, "🍜 You're near Place n");
    expect(body, contains('You saved it from a Phuket reel.'));
    expect(body, contains('eat'));
    final (seeTitle, seeBody) = nearbyAlertText(spot('b', 7.8, 98.3));
    expect(seeTitle, startsWith('📍'));
    expect(seeBody, contains('saved list'));
  });

  test('food alerts fire closer in than sights', () {
    expect(nearbyRadiusMeters('eat'), lessThan(nearbyRadiusMeters('see')));
    final g = nearbyGeofence(spot('k', 7.82, 98.29, kind: 'drink'));
    expect(g.radiusMeters, 250);
    expect(g.iosSettings.initialTrigger, isFalse, reason: 'no alert for a place you are already at when syncing');
  });

  test('spots survive a JSON round trip for the background isolate', () {
    final s = NearbySpot(id: 'sav_1', name: 'Big Buddha', kind: 'see', lat: 7.8276, lng: 98.3129, geoPlaceId: 'gp_1', source: 'a Phuket reel');
    final again = NearbySpot.fromJson(s.toJson());
    expect((again.id, again.lat, again.geoPlaceId, again.source), ('sav_1', 7.8276, 'gp_1', 'a Phuket reel'));
  });
}
