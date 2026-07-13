// validateCode: seeded demo codes, on-device claims, and rejects.

import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:spotto/core/storage/local_store.dart';
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

  test('seeded demo code validates', () async {
    final c = await makeContainer();
    final res = c.read(Provider((ref) => validateCode(ref, 'spotto-x4k9')));
    expect(res.valid, isTrue);
    expect(res.name, 'Tom R.');
  });

  test('code claimed on this device validates via the claims store', () async {
    final c = await makeContainer();
    final code = c.read(claimsProvider.notifier).claim('driftwood');
    final res = c.read(Provider((ref) => validateCode(ref, code)));
    expect(res.valid, isTrue);
    expect(res.name, 'You (this device)');
  });

  test('unknown code is rejected', () async {
    final c = await makeContainer();
    final res = c.read(Provider((ref) => validateCode(ref, 'SPOTTO-NOPE')));
    expect(res.valid, isFalse);
  });
}
