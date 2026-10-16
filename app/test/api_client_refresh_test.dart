// Session refresh: an expired access token is swapped for a new one once
// (even when several requests hit 401 together) and the requests retried;
// only a refresh the server rejects signs the user out — being offline
// doesn't.

import 'dart:convert';

import 'package:dio/dio.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:spotto/core/api/api_client.dart';
import 'package:spotto/core/storage/local_store.dart';

/// Fake server: /me needs "Bearer fresh"; /auth/refresh answers per [refresh].
class _FakeServer implements HttpClientAdapter {
  _FakeServer(this.refresh);

  /// 'ok' issues a new pair, 'reject' answers 401, 'offline' fails to connect.
  final String refresh;
  int refreshCalls = 0;

  @override
  Future<ResponseBody> fetch(RequestOptions options, Stream<Uint8List>? requestStream, Future<void>? cancelFuture) async {
    if (options.path == '/auth/refresh') {
      refreshCalls++;
      await Future<void>.delayed(const Duration(milliseconds: 20));
      if (refresh == 'offline') throw DioException.connectionError(requestOptions: options, reason: 'offline (test)');
      if (refresh == 'reject') return _json(401, {'error': 'session ended'});
      return _json(200, {'token': 'fresh', 'refreshToken': 'r2'});
    }
    final ok = options.headers['Authorization'] == 'Bearer fresh';
    return ok ? _json(200, {'id': 'u1'}) : _json(401, {'error': 'invalid token'});
  }

  ResponseBody _json(int status, Object body) => ResponseBody.fromString(
    jsonEncode(body),
    status,
    headers: {
      Headers.contentTypeHeader: [Headers.jsonContentType],
    },
  );

  @override
  void close({bool force = false}) {}
}

Future<(ApiClient, _FakeServer, LocalStore, List<int>)> _client(String refresh) async {
  final secure = <String, String>{'spotto_token': 'expired', 'spotto_refresh_token': 'r1'};
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
  SharedPreferences.setMockInitialValues({});
  final store = await LocalStore.init();
  final server = _FakeServer(refresh);
  final api = ApiClient(store, adapter: server);
  final signOuts = <int>[];
  api.onUnauthorized = () => signOuts.add(1);
  return (api, server, store, signOuts);
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  test('concurrent 401s share one refresh and are retried', () async {
    final (api, server, store, signOuts) = await _client('ok');
    final results = await Future.wait([for (var i = 0; i < 3; i++) api.dio.get('/me')]);
    expect(results.map((r) => r.statusCode), everyElement(200));
    expect(server.refreshCalls, 1);
    expect(store.getToken(), 'fresh');
    expect(store.getRefreshToken(), 'r2');
    expect(signOuts, isEmpty);
  });

  test('a rejected refresh signs out', () async {
    final (api, _, _, signOuts) = await _client('reject');
    await expectLater(api.dio.get('/me'), throwsA(isA<DioException>()));
    expect(signOuts, isNotEmpty);
  });

  test('being offline during refresh keeps the session', () async {
    final (api, _, store, signOuts) = await _client('offline');
    await expectLater(api.dio.get('/me'), throwsA(isA<DioException>()));
    expect(signOuts, isEmpty);
    expect(store.getRefreshToken(), 'r1');
  });
}
