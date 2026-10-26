// Signing in and out while other providers that follow the session are
// alive. Riverpod 3 rejects (in debug builds) a provider invalidating one
// that depends on it, so AuthNotifier must not invalidate providers that
// watch authProvider: they refresh on their own when it changes.

import 'package:dio/dio.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:spotto/core/api/api_client.dart';
import 'package:spotto/core/storage/local_store.dart';
import 'package:spotto/data/repositories/repositories.dart';

const _quest = {'goal': 3, 'progress': 1, 'rewardDescription': 'Free coconut', 'isComplete': false};

/// A container whose API answers login and today's quest, records every
/// request, and fails everything else as if offline.
Future<(ProviderContainer, List<String>)> _container({String? token}) async {
  SharedPreferences.setMockInitialValues({});
  TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger.setMockMethodCallHandler(
    const MethodChannel('plugins.it_nomads.com/flutter_secure_storage'),
    (call) async => call.method == 'read' && call.arguments['key'] == StoreKeys.token ? token : null,
  );
  final store = await LocalStore.init();
  final requests = <String>[];
  final api = ApiClient(store);
  api.dio.interceptors.add(InterceptorsWrapper(onRequest: (options, handler) {
    requests.add(options.path);
    switch (options.path) {
      case '/auth/login':
        handler.resolve(Response(
          requestOptions: options,
          statusCode: 200,
          data: {'token': 'new-token', 'refreshToken': 'new-refresh', 'user': <String, dynamic>{}},
        ));
      case '/me/quest':
        handler.resolve(Response(requestOptions: options, statusCode: 200, data: _quest));
      default:
        handler.reject(DioException(requestOptions: options, message: 'offline (test)'));
    }
  }));
  final container = ProviderContainer(
    retry: (_, _) => null,
    overrides: [
      localStoreProvider.overrideWithValue(store),
      apiProvider.overrideWithValue(api),
    ],
  );
  addTearDown(container.dispose);
  return (container, requests);
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  test("login succeeds after today's quest was shown, and the quest refetches", () async {
    final (c, requests) = await _container();
    // Discover was open earlier in this run (e.g. before a sign-out).
    c.listen(dailyQuestProvider, (_, _) {});
    c.listen(cachedQuestProvider, (_, _) {});
    expect((await c.read(dailyQuestProvider.future)).goal, kDefaultQuest.goal);

    final error = await c.read(authProvider.notifier).login('traveller@example.com', 'password-123');

    expect(error, isNull);
    expect(c.read(authProvider), isTrue);
    expect((await c.read(dailyQuestProvider.future)).goal, 3);
    expect(requests, contains('/me/quest'));
  });

  test('sign-out with the quest providers alive', () async {
    final (c, _) = await _container(token: 'old-token');
    c.listen(dailyQuestProvider, (_, _) {});
    c.listen(cachedQuestProvider, (_, _) {});
    await c.read(dailyQuestProvider.future);

    c.read(authProvider.notifier).signOut();

    expect(c.read(authProvider), isFalse);
    expect((await c.read(dailyQuestProvider.future)).goal, kDefaultQuest.goal);
  });
}
