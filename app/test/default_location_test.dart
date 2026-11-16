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

  test('resolveDemoLocation correctly resolves Ratnapura and custom coordinates', () {
    // 1. Ratnapura by name
    final ratnapura = resolveDemoLocation({
      'demoPrivilege': true,
      'demoLocation': 'Ratnapura',
    });
    expect(ratnapura, isNotNull);
    expect(ratnapura!.name, 'Ratnapura');
    expect(ratnapura.lat, closeTo(6.6828, 0.001));
    expect(ratnapura.lng, closeTo(80.3992, 0.001));

    // 2. Explicit demoLat and demoLng
    final custom = resolveDemoLocation({
      'demoPrivilege': true,
      'demoLocation': 'My Gem Mine',
      'demoLat': 6.7000,
      'demoLng': 80.4000,
    });
    expect(custom, isNotNull);
    expect(custom!.name, 'My Gem Mine');
    expect(custom.lat, 6.7000);
    expect(custom.lng, 80.4000);

    // 3. Comma-separated coordinates
    final coords = resolveDemoLocation({
      'demoPrivilege': true,
      'demoLocation': '6.9271, 79.8612',
    });
    expect(coords, isNotNull);
    expect(coords!.lat, closeTo(6.9271, 0.001));
    expect(coords.lng, closeTo(79.8612, 0.001));

    // 4. No demo privilege returns null
    final noPriv = resolveDemoLocation({
      'demoPrivilege': false,
      'demoLocation': 'Ratnapura',
    });
    expect(noPriv, isNull);
  });
}

