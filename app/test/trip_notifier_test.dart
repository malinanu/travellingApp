// TripNotifier semantics: move swaps times, custom stops survive JSON,
// and pre-custom-schema payloads still load. Also what a first launch
// starts with (a blank trip and group) and how saved copies of the old
// sample trip/group are cleaned up.

import 'dart:convert';

import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:spotto/core/storage/local_store.dart';
import 'package:spotto/data/catalog/trip_catalog.dart';
import 'package:spotto/data/models/models.dart';
import 'package:spotto/data/repositories/repositories.dart';

/// flutter_secure_storage has no real platform implementation under
/// `flutter test` — mock its channel so LocalStore.init() (which reads the
/// session token from secure storage) doesn't throw MissingPluginException.
void _mockSecureStorage() {
  const channel = MethodChannel('plugins.it_nomads.com/flutter_secure_storage');
  TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
      .setMockMethodCallHandler(channel, (call) async {
    switch (call.method) {
      case 'read':
        return null;
      case 'readAll':
        return <String, String>{};
      case 'containsKey':
        return false;
      default:
        return null;
    }
  });
}

Future<ProviderContainer> makeContainer([Map<String, Object> saved = const {}]) async {
  SharedPreferences.setMockInitialValues(saved);
  _mockSecureStorage();
  final store = await LocalStore.init();
  final container = ProviderContainer(overrides: [
    localStoreProvider.overrideWithValue(store),
  ]);
  addTearDown(container.dispose);
  return container;
}

const _beach = TripCatalogItem(id: 'beach', name: 'Mirissa Beach', type: 'Beach', icon: 'Palmtree', lat: 5.9445, lng: 80.4595, cost: 0, styles: ['surf'], hint: '15:00');

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  tearDown(() => tripCatalog = []);

  test('a first launch starts on a blank Spotto Trip', () async {
    final c = await makeContainer();
    final trip = c.read(tripProvider);
    expect(trip.name, 'Spotto Trip');
    expect(trip.code, startsWith('TRP-'));
    expect(trip.collaborators, isEmpty);
    expect(trip.days.length, 3);
    expect(trip.days.every((d) => d.stops.isEmpty), isTrue);
  });

  test('a saved copy of the old sample trip is cleaned on load', () async {
    final sample = const Trip(
      name: 'Weligama Weekend',
      code: 'WLG-7QF',
      startDate: '2026-06-20',
      collaborators: ['theo', 'mia'],
      days: [
        TripDay(id: 'd1', stops: [
          TripStop(id: 's1', placeId: 'bayvilla', time: 14 * 60),
          TripStop(id: 's102_7', placeId: 'beach', time: 16 * 60, by: 'me'),
        ]),
      ],
    ).toJson();
    final c = await makeContainer({StoreKeys.trip: jsonEncode(sample)});
    final trip = c.read(tripProvider);
    expect(trip.name, 'Spotto Trip');
    expect(trip.collaborators, isEmpty);
    expect(trip.days.single.stops.map((s) => s.id), ['s102_7']);
  });

  test('a first launch starts with an empty Spotto Crew group', () async {
    final c = await makeContainer();
    final group = c.read(groupProvider);
    expect(group.name, 'Spotto Crew');
    expect(group.code, matches(RegExp(r'^GRP-[0-9A-Z]{6}$')));
    expect(group.memberIds, isEmpty);
  });

  test('a saved copy of the old sample group is dropped, a real one kept', () async {
    final sample = const Group(name: 'Weligama Crew', code: 'WLG-4K2', memberIds: ['mia', 'theo', 'ines']).toJson();
    var c = await makeContainer({StoreKeys.group: jsonEncode(sample)});
    var group = c.read(groupProvider);
    expect(group.name, 'Spotto Crew');
    expect(group.code, startsWith('GRP-'));
    expect(group.memberIds, isEmpty);

    final real = const Group(name: 'Beach friends', code: 'GRP-ABC123', memberIds: ['usr_1']).toJson();
    c = await makeContainer({StoreKeys.group: jsonEncode(real)});
    group = c.read(groupProvider);
    expect(group.name, 'Beach friends');
    expect(group.code, 'GRP-ABC123');
    expect(group.memberIds, ['usr_1']);
  });

  test('moveStop swaps stops but keeps the time slots chronological', () async {
    final c = await makeContainer();
    final notifier = c.read(tripProvider.notifier);
    notifier.addCustomStop(0, 'First');
    notifier.addCustomStop(0, 'Second');
    final before = c.read(tripProvider).days[0].stops;
    final t0 = before[0].time, t1 = before[1].time;
    final id0 = before[0].id, id1 = before[1].id;

    notifier.moveStop(0, 0, 1);

    final after = c.read(tripProvider).days[0].stops;
    expect(after[0].id, id1);
    expect(after[1].id, id0);
    expect(after[0].time, t0);
    expect(after[1].time, t1);
  });

  test('createTrip builds a fresh trip with a new code and empty days', () async {
    final c = await makeContainer();
    final notifier = c.read(tripProvider.notifier);
    final oldCode = c.read(tripProvider).code;

    notifier.createTrip('Test Trip', 3, '2026-08-01');

    final trip = c.read(tripProvider);
    expect(trip.name, 'Test Trip');
    expect(trip.days.length, 3);
    expect(trip.days.every((d) => d.stops.isEmpty), isTrue);
    expect(trip.code, isNot(oldCode));
    expect(trip.code, startsWith('TRP-'));
  });

  test('addCustomStop round-trips through JSON', () async {
    final c = await makeContainer();
    final notifier = c.read(tripProvider.notifier);
    notifier.addCustomStop(0, 'Secret Beach');

    final trip = c.read(tripProvider);
    final restored = Trip.fromJson(trip.toJson());
    final custom = restored.days[0].stops.where((s) => s.custom != null).toList();
    expect(custom, hasLength(1));
    expect(custom.first.custom!.name, 'Secret Beach');
    expect(tripPlace(custom.first)!.name, 'Secret Beach');
  });

  test('addCatalogStop lands 90 min after the last stop', () async {
    tripCatalog = [_beach];
    final c = await makeContainer();
    final notifier = c.read(tripProvider.notifier);
    notifier.addCustomStop(0, 'Breakfast');
    final last = c.read(tripProvider).days[0].stops.last.time;

    notifier.addCatalogStop(0, tripCatalog.single);

    final stops = c.read(tripProvider).days[0].stops;
    expect(stops.map((s) => s.placeId), contains('beach'));
    final added = stops.firstWhere((s) => s.placeId == 'beach');
    expect(added.time, (last + 90).clamp(0, 23 * 60 + 30));
  });

  test('pre-custom-schema stop JSON still loads', () {
    tripCatalog = [_beach];
    final stop = TripStop.fromJson({'id': 's1', 'placeId': 'beach', 'time': 480});
    expect(stop.custom, isNull);
    expect(tripPlace(stop)!.name, 'Mirissa Beach');
  });
}
