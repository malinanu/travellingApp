// Cold-start snapshot rules: the Discover screen opens from what's saved on
// the device, but only while that data is still trustworthy (today's quest,
// weather under an hour old, /me only while signed in, a recent fix with
// location permission still granted), and categories aren't refetched on
// every launch.

import 'dart:convert';

import 'package:dio/dio.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:spotto/core/api/api_client.dart';
import 'package:spotto/core/storage/local_store.dart';
import 'package:spotto/core/widgets/progressive_image.dart';
import 'package:spotto/data/repositories/repositories.dart';

void _mockSecureStorage({String? token}) {
  TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger.setMockMethodCallHandler(
    const MethodChannel('plugins.it_nomads.com/flutter_secure_storage'),
    (call) async => call.method == 'read' && call.arguments['key'] == StoreKeys.token ? token : null,
  );
}

/// LocationPermission index: 0 denied, 1 deniedForever, 2 whileInUse, 3 always.
void _mockLocationPermission(int index) {
  TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger.setMockMethodCallHandler(
    const MethodChannel('flutter.baseflow.com/geolocator'),
    (call) async => call.method == 'checkPermission' ? index : null,
  );
}

/// Every API request is recorded and then failed, so tests can assert on
/// which calls were made without a server.
Future<(ProviderContainer, List<String>)> makeContainer(Map<String, Object> prefs, {String? token}) async {
  SharedPreferences.setMockInitialValues(prefs);
  _mockSecureStorage(token: token);
  final store = await LocalStore.init();
  final requests = <String>[];
  final api = ApiClient(store);
  api.dio.interceptors.add(InterceptorsWrapper(onRequest: (options, handler) {
    requests.add(options.path);
    handler.reject(DioException(requestOptions: options, message: 'offline (test)'));
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

String _today() => questDay(DateTime.now());

String _json(Object v) => const JsonEncoder().convert(v);

const _questJson = {'goal': 3, 'progress': 1, 'rewardDescription': 'Free coconut', 'isComplete': false};

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  group('cachedQuestProvider', () {
    test("uses today's saved quest when signed in", () async {
      final (c, _) = await makeContainer({
        StoreKeys.quest: _json({'day': _today(), 'data': _questJson}),
      }, token: 'tok');
      expect(c.read(cachedQuestProvider)?.goal, 3);
    });

    test('ignores a quest saved on another day', () async {
      final (c, _) = await makeContainer({
        StoreKeys.quest: _json({'day': '2001-1-1', 'data': _questJson}),
      }, token: 'tok');
      expect(c.read(cachedQuestProvider), isNull);
    });

    test('a failed refresh keeps the saved quest instead of the default', () async {
      final (c, requests) = await makeContainer({
        StoreKeys.quest: _json({'day': _today(), 'data': _questJson}),
      }, token: 'tok');
      final quest = await c.read(dailyQuestProvider.future);
      expect(requests, contains('/me/quest'));
      expect(quest.goal, 3);
    });
  });

  group('questDay', () {
    test('follows the UTC date, like the server', () {
      // 01:00 on Oct 16 in Sri Lanka (UTC+5:30) is still Oct 15 in UTC.
      final t = DateTime.utc(2026, 10, 15, 19, 30).toLocal();
      expect(questDay(t), '2026-10-15');
    });
  });

  group('cachedWeatherProvider', () {
    test('ignores a snapshot older than an hour', () async {
      final old = DateTime.now().subtract(const Duration(hours: 2)).millisecondsSinceEpoch;
      final (c, _) = await makeContainer({
        StoreKeys.weather: _json({'at': old, 'data': <String, dynamic>{}}),
      });
      expect(c.read(cachedWeatherProvider), isNull);
    });
  });

  group('freshFor', () {
    test('returns the time left for a recent timestamp', () {
      final at = DateTime.now().subtract(const Duration(minutes: 10)).millisecondsSinceEpoch;
      final left = freshFor(at, const Duration(hours: 1));
      expect(left, isNotNull);
      expect(left!.inMinutes, inInclusiveRange(49, 50));
    });

    test('is null when stale, missing, or in the future', () {
      final old = DateTime.now().subtract(const Duration(hours: 2)).millisecondsSinceEpoch;
      final future = DateTime.now().add(const Duration(hours: 1)).millisecondsSinceEpoch;
      expect(freshFor(old, const Duration(hours: 1)), isNull);
      expect(freshFor(null, const Duration(hours: 1)), isNull);
      expect(freshFor(future, const Duration(hours: 1)), isNull);
    });
  });

  group('mediaCacheKey', () {
    test('drops only the rotating token', () {
      expect(mediaCacheKey('https://api.x/media/raw/a.jpg?token=abc'), 'https://api.x/media/raw/a.jpg');
      expect(mediaCacheKey('https://api.x/media/raw/a.jpg?w=320&token=abc'), 'https://api.x/media/raw/a.jpg?w=320');
      expect(mediaCacheKey('https://cdn.x/p.jpg?w=320'), 'https://cdn.x/p.jpg?w=320');
    });
  });

  group('meProvider', () {
    test('starts from the saved /me when signed in', () async {
      final (c, _) = await makeContainer({
        StoreKeys.me: _json({'id': 'u1', 'name': 'Asha', 'email': 'a@example.com'}),
      }, token: 'tok');
      expect(c.read(meProvider)?['id'], 'u1');
    });

    test('ignores the saved /me when signed out', () async {
      final (c, _) = await makeContainer({
        StoreKeys.me: _json({'id': 'u1', 'name': 'Asha'}),
      });
      expect(c.read(meProvider), isNull);
    });
  });

  group('categoriesProvider', () {
    test('skips the launch fetch when categories were fetched recently', () async {
      final (c, requests) = await makeContainer({
        StoreKeys.categoriesAt: '${DateTime.now().millisecondsSinceEpoch}',
      });
      c.read(categoriesProvider);
      await Future<void>.delayed(const Duration(milliseconds: 100));
      expect(requests, isNot(contains('/categories')));
    });

    test('fetches at launch when the saved list is stale', () async {
      final stale = DateTime.now().subtract(const Duration(hours: 2)).millisecondsSinceEpoch;
      final (c, requests) = await makeContainer({StoreKeys.categoriesAt: '$stale'});
      c.read(categoriesProvider);
      await Future<void>.delayed(const Duration(milliseconds: 100));
      expect(requests, contains('/categories'));
    });
  });

  group('recentSavedPosition', () {
    Map<String, Object> savedPos(Duration age) => {
          StoreKeys.lastPosition: _json({
            'lat': 6.0,
            'lng': 80.2,
            'at': DateTime.now().subtract(age).millisecondsSinceEpoch,
          }),
        };

    test('returns a recent fix while permission is granted', () async {
      _mockLocationPermission(2);
      final (c, _) = await makeContainer(savedPos(const Duration(minutes: 10)));
      final pos = await recentSavedPosition(c.read(localStoreProvider));
      expect(pos?.lat, 6.0);
    });

    test('returns null once permission has been revoked', () async {
      _mockLocationPermission(1);
      final (c, _) = await makeContainer(savedPos(const Duration(minutes: 10)));
      expect(await recentSavedPosition(c.read(localStoreProvider)), isNull);
    });

    test('returns null for a fix older than the max age', () async {
      _mockLocationPermission(2);
      final (c, _) = await makeContainer(savedPos(const Duration(hours: 7)));
      expect(await recentSavedPosition(c.read(localStoreProvider)), isNull);
    });
  });
}
