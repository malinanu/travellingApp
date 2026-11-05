import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:spotto/data/repositories/repositories.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  tearDown(() {
    setServerDefaultLocation(null);
  });

  test('bestEffortPosition returns Mirissa with granted result when default location is set', () async {
    setServerDefaultLocation((lat: 5.9483, lng: 80.4552, name: 'Mirissa', zoom: 14.5));

    expect(serverDefaultLocation, isNotNull);
    expect(serverDefaultLocation!.name, 'Mirissa');
    expect(serverDefaultLocation!.lat, 5.9483);
    expect(serverDefaultLocation!.lng, 80.4552);

    final (pos, result) = await bestEffortPosition();

    expect(pos, isNotNull);
    expect(pos!.latitude, 5.9483);
    expect(pos.longitude, 80.4552);
    expect(result, DeviceLocationResult.granted);
    expect(await hasLocationPermission(), isTrue);

    setServerDefaultLocation(null);
    expect(serverDefaultLocation, isNull);
  });

  test('weatherLocationProvider and devicePositionProvider resolve Mirissa under demo privilege', () async {
    setServerDefaultLocation((lat: 5.9483, lng: 80.4552, name: 'Mirissa', zoom: 14.5));

    final container = ProviderContainer();
    addTearDown(container.dispose);

    final devicePos = await container.read(devicePositionProvider.future);
    expect(devicePos.$1.lat, 5.9483);
    expect(devicePos.$1.lng, 80.4552);
    expect(devicePos.$2, DeviceLocationResult.granted);

    final weatherLoc = await container.read(weatherLocationProvider.future);
    expect(weatherLoc.lat, 5.95);
    expect(weatherLoc.lng, 80.46);

    setServerDefaultLocation(null);
    expect(serverDefaultLocation, isNull);
  });
}

