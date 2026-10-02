// Push setup is best-effort: when Firebase/APNs is unavailable (e.g. an
// unsigned sideloaded iOS build) initialize() must complete quietly instead
// of throwing, otherwise it can take app startup down with it.

import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:spotto/core/api/api_client.dart';
import 'package:spotto/core/api/push_notification_service.dart';
import 'package:spotto/core/storage/local_store.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  test('initialize swallows Firebase failures instead of throwing', () async {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(
      const MethodChannel('plugins.it_nomads.com/flutter_secure_storage'),
      (call) async => null,
    );
    SharedPreferences.setMockInitialValues({});
    final api = ApiClient(await LocalStore.init());

    // No Firebase platform implementation exists under `flutter test`, so
    // Firebase.initializeApp throws — it must not escape.
    await PushNotificationService.instance.initialize(api);
    expect(PushNotificationService.instance.deviceToken, isNull);
  });
}
