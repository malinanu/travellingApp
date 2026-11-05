import 'package:flutter_test/flutter_test.dart';
import 'package:spotto/data/repositories/repositories.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  tearDown(() {
    setServerDefaultLocation(null);
  });

  test('bestEffortPosition returns Mirissa with granted result when default location is set', () async {
    setServerDefaultLocation((lat: 5.9483, lng: 80.4552, name: 'Mirissa', zoom: 14.5));

    final (pos, result) = await bestEffortPosition();

    expect(pos, isNotNull);
    expect(pos!.latitude, 5.9483);
    expect(pos.longitude, 80.4552);
    expect(result, DeviceLocationResult.granted);
    expect(await hasLocationPermission(), isTrue);
  });

  test('fallback coordinates point to Mirissa Beach', () {
    expect(kFallbackLat, 5.9483);
    expect(kFallbackLng, 80.4552);
  });
}
