// First launch after an install starts signed out, even when the platform
// kept the old session (iOS keeps Keychain items when an app is deleted).

import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:spotto/core/storage/local_store.dart';

Map<String, String> _mockSecure(Map<String, String> initial) {
  final secure = {...initial};
  TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger.setMockMethodCallHandler(
    const MethodChannel('plugins.it_nomads.com/flutter_secure_storage'),
    (call) async {
      final key = call.arguments['key'] as String?;
      switch (call.method) {
        case 'read':
          return secure[key];
        case 'write':
          secure[key!] = call.arguments['value'] as String;
        case 'delete':
          secure.remove(key);
      }
      return null;
    },
  );
  return secure;
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  test('a fresh install drops a session left in secure storage', () async {
    final secure = _mockSecure({StoreKeys.token: 'stale', StoreKeys.refreshToken: 'stale-refresh'});
    SharedPreferences.setMockInitialValues({});
    final store = await LocalStore.init();
    expect(store.getToken(), isNull);
    expect(store.getRefreshToken(), isNull);
    expect(secure, isEmpty);
    expect(store.getBool(StoreKeys.installed), isTrue);
  });

  test('an existing install keeps its session', () async {
    _mockSecure({StoreKeys.token: 'live', StoreKeys.refreshToken: 'live-refresh'});
    SharedPreferences.setMockInitialValues({StoreKeys.installed: true});
    final store = await LocalStore.init();
    expect(store.getToken(), 'live');
    expect(store.getRefreshToken(), 'live-refresh');
  });

  test('an install from before this flag (other saved keys) keeps its session', () async {
    _mockSecure({StoreKeys.token: 'live'});
    SharedPreferences.setMockInitialValues({StoreKeys.hasSelectedLanguage: true});
    final store = await LocalStore.init();
    expect(store.getToken(), 'live');
  });
}
