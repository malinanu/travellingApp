// Inspiration Board: reel/saved-place models, shared-link parsing, the
// board's polling while the server watches a reel, saved places keeping map
// bookmarks in step, and places from reels keeping their real location in
// the trip.

import 'package:dio/dio.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:spotto/core/api/api_client.dart';
import 'package:spotto/core/storage/local_store.dart';
import 'package:spotto/data/models/inspiration.dart';
import 'package:spotto/data/repositories/inspiration_repository.dart';
import 'package:spotto/data/repositories/repositories.dart';

const _geo = {
  'id': 'gp_1',
  'name': 'Big Buddha Phuket',
  'lat': 7.8276,
  'lng': 98.3129,
  'address': 'Karon, Mueang Phuket',
  'city': 'Phuket',
  'countryCode': 'TH',
  'kind': 'see',
  'rating': 4.7,
  'ratingCount': 51234,
  'photoUrl': '/geo-places/gp_1/photo',
};

Map<String, dynamic> _link(String status, {List<Map<String, dynamic>> places = const []}) => {
      'id': 'insp_1',
      'url': 'https://www.instagram.com/reels/DbQSwgwMM_n/',
      'createdAt': '2026-10-20T08:00:00Z',
      'platform': 'instagram',
      'status': status,
      'author': 'itsemandty',
      'places': places,
    };

/// Signed-in container whose API answers from [respond] (path, method, body).
Future<(ProviderContainer, List<String>)> makeContainer(
  Object? Function(String method, String path, Object? body) respond,
) async {
  SharedPreferences.setMockInitialValues({StoreKeys.auth: true, StoreKeys.boardMigrated: true});
  TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger.setMockMethodCallHandler(
    const MethodChannel('plugins.it_nomads.com/flutter_secure_storage'),
    (call) async => call.method == 'read' && call.arguments['key'] == StoreKeys.token ? 'tok' : null,
  );
  final store = await LocalStore.init();
  final calls = <String>[];
  final api = ApiClient(store);
  api.dio.interceptors.add(InterceptorsWrapper(onRequest: (options, handler) {
    calls.add('${options.method} ${options.path}');
    final data = respond(options.method, options.path, options.data);
    if (data == null) {
      handler.reject(DioException(requestOptions: options, message: 'not faked'));
    } else {
      handler.resolve(Response(requestOptions: options, statusCode: 200, data: data));
    }
  }));
  final container = ProviderContainer(
    retry: (_, _) => null,
    overrides: [localStoreProvider.overrideWithValue(store), apiProvider.overrideWithValue(api)],
  );
  addTearDown(container.dispose);
  return (container, calls);
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  test('a reel link parses its places and labels', () {
    final link = BoardLink.fromJson(_link('done', places: [
      {'name': 'Big Buddha', 'kind': 'see', 'city': 'Phuket', 'evidence': 'on_screen', 'atSec': 42, 'confidence': 1, 'geoPlace': _geo},
      {'name': 'Secret Beach Bar', 'kind': 'drink', 'evidence': 'speech', 'confidence': 0.4},
    ]));
    expect(link.places, hasLength(2));
    expect(link.confirmedPlaces.single.geoPlace!.name, 'Big Buddha Phuket');
    expect(link.places.first.evidenceLabel, 'At 0:42 · on screen');
    expect(link.places.last.evidenceLabel, 'mentioned');
    expect(link.region, 'Phuket');
    expect(link.working, isFalse);
    expect(link.canRetry, isFalse);
  });

  test('finds the reel link in shared text', () {
    expect(findSupportedLink('Look at this! https://www.instagram.com/reel/DbQSwgwMM_n/?igsh=x and more'),
        'https://www.instagram.com/reel/DbQSwgwMM_n/?igsh=x');
    expect(findSupportedLink('https://vm.tiktok.com/ZMabc123/'), 'https://vm.tiktok.com/ZMabc123/');
    expect(findSupportedLink('https://youtu.be/aBcDeFgHiJk'), 'https://youtu.be/aBcDeFgHiJk');
    expect(findSupportedLink('Watch this https://www.facebook.com/share/r/1AbCdEfGh/ 🌴'), 'https://www.facebook.com/share/r/1AbCdEfGh/');
    expect(findSupportedLink('https://fb.watch/qWeRtY12/'), 'https://fb.watch/qWeRtY12/');
    expect(findSupportedLink('https://m.facebook.com/reel/4720710134913664'), 'https://m.facebook.com/reel/4720710134913664');
    expect(findSupportedLink('https://example.com/blog'), isNull);
  });

  test('saved places round-trip through local storage JSON', () {
    final saved = SavedPlace.fromJson({'id': 'sav_1', 'geoPlaceId': 'gp_1', 'createdAt': '2026-10-20T08:00:00Z', 'geoPlace': _geo});
    final again = SavedPlace.fromJson(saved.toJson());
    expect(again.name, 'Big Buddha Phuket');
    expect(again.lat, 7.8276);
    expect(again.city, 'Phuket');
    expect(again.geoPlace!.photoUrl, '/geo-places/gp_1/photo');
  });

  test('the board polls while a reel is being watched, then stops', () async {
    var gets = 0;
    final (c, calls) = await makeContainer((method, path, _) {
      if (method == 'GET' && path == '/me/inspirations') {
        gets++;
        return {
          'links': [gets == 1 ? _link('processing') : _link('done', places: [{'name': 'Big Buddha', 'geoPlace': _geo}])]
        };
      }
      return null;
    });
    c.read(boardLinksProvider);
    await Future<void>.delayed(const Duration(milliseconds: 200));
    expect(c.read(boardLinksProvider).single.working, isTrue);

    await Future<void>.delayed(boardPollInterval + const Duration(milliseconds: 500));
    expect(c.read(boardLinksProvider).single.status, 'done');
    expect(gets, 2);

    await Future<void>.delayed(boardPollInterval + const Duration(milliseconds: 500));
    expect(gets, 2, reason: 'no more polling once nothing is being watched');
    expect(calls.where((c) => c == 'GET /me/inspirations'), hasLength(2));
  });

  test('saving a catalog place bookmarks it on the map too', () async {
    final (c, _) = await makeContainer((method, path, body) {
      if (method == 'GET' && path == '/me/saved-places') return {'places': []};
      if (method == 'POST' && path == '/me/saved-places') {
        return {
          'id': 'sav_9',
          'placeId': (body as Map)['placeId'],
          'createdAt': '2026-10-20T08:00:00Z',
          'place': {'id': body['placeId'], 'name': 'Weligama Surf Cafe', 'lat': 5.97, 'lng': 80.42},
        };
      }
      return null;
    });
    c.read(savedPlacesProvider);
    await c.read(favoritesProvider.notifier).toggle('pl_surf');
    expect(c.read(savedPlacesProvider).single.placeId, 'pl_surf');
    expect(c.read(favoritesProvider), contains('pl_surf'));
  });

  test('a place from a reel keeps its real location in the trip', () async {
    final (c, _) = await makeContainer((_, _, _) => null);
    final trip = c.read(tripProvider.notifier);
    if (c.read(tripProvider).days.isEmpty) trip.addDay();
    trip.addGeoStop(0, name: 'Big Buddha Phuket', lat: 7.8276, lng: 98.3129, type: 'Sight', icon: 'Camera');
    final stop = c.read(tripProvider).days.first.stops.last;
    expect(stop.custom!.name, 'Big Buddha Phuket');
    expect(stop.custom!.lat, 7.8276);
    expect(stop.custom!.lng, 98.3129);
    expect(stop.custom!.icon, 'Camera');
  });
}
