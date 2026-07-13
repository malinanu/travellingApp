// TripNotifier semantics: move swaps times, custom stops survive JSON,
// and pre-custom-schema payloads still load.

import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:spotto/core/storage/local_store.dart';
import 'package:spotto/data/mock/trip_catalog.dart';
import 'package:spotto/data/repositories/repositories.dart';

Future<ProviderContainer> makeContainer() async {
  SharedPreferences.setMockInitialValues({});
  final store = await LocalStore.init();
  final container = ProviderContainer(overrides: [
    localStoreProvider.overrideWithValue(store),
  ]);
  addTearDown(container.dispose);
  return container;
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  test('moveStop swaps stops but keeps the time slots chronological', () async {
    final c = await makeContainer();
    final notifier = c.read(tripProvider.notifier);
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
    final c = await makeContainer();
    final notifier = c.read(tripProvider.notifier);
    final last = c.read(tripProvider).days[0].stops.last.time;
    final item = tripCatalog.firstWhere((p) => p.id == 'mirissa');

    notifier.addCatalogStop(0, item);

    final stops = c.read(tripProvider).days[0].stops;
    expect(stops.map((s) => s.placeId), contains('mirissa'));
    final added = stops.firstWhere((s) => s.placeId == 'mirissa');
    expect(added.time, (last + 90).clamp(0, 23 * 60 + 30));
  });

  test('pre-custom-schema stop JSON still loads', () {
    final stop = TripStop.fromJson({'id': 's1', 'placeId': 'driftwood', 'time': 480});
    expect(stop.custom, isNull);
    expect(tripPlace(stop)!.name, 'Driftwood Coffee & Co-Work');
  });
}
