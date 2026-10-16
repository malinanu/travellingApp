// Thin dio wrapper for the Spotto Go API.
//
// Base URL comes from --dart-define=API_URL, defaulting to the Android
// emulator's host loopback. The JWT is kept in LocalStore's secure storage
// and attached as a Bearer header. Access tokens last an hour: a 401 first
// trades the session's refresh token for a new pair and retries once; only
// when the server rejects the refresh does onUnauthorized fire (wired to
// sign-out in repositories.dart).

import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart' show debugPrint, defaultTargetPlatform, kReleaseMode, visibleForTesting;

import '../storage/local_store.dart';

const apiBaseUrl = String.fromEnvironment('API_URL', defaultValue: 'https://spottoapi.threelogiq.com');

/// The R2 bucket behind media uploads is private — the server hands back
/// app-relative paths like `/media/raw/<key>?token=...` (see
/// handleMediaRaw/signMediaPath server-side) rather than an absolute R2/CDN
/// URL, so every render site needs to resolve against our own API host
/// before handing the URL to Image.network/VideoPlayerController. Anything
/// already absolute (http/https) passes through unchanged.
String resolveMediaUrl(String url) {
  if (url.startsWith('/')) {
    final base = apiBaseUrl.endsWith('/') ? apiBaseUrl.substring(0, apiBaseUrl.length - 1) : apiBaseUrl;
    return '$base$url';
  }
  return url;
}

// Guards against shipping a release build that still points at the plaintext
// emulator default (or any http:// URL) — the bearer token would go out
// over cleartext. Uses kReleaseMode + a thrown error rather than assert()
// because asserts are stripped from release binaries and wouldn't fire here.
String _validatedBaseUrl() {
  if (kReleaseMode && !apiBaseUrl.startsWith('https://')) {
    throw StateError(
      'Release builds must set --dart-define=API_URL=https://... '
      '(got "$apiBaseUrl") — refusing to send auth tokens over plaintext HTTP.',
    );
  }
  return apiBaseUrl;
}

/// Sent with every request so the server can tell sessions apart (the app's
/// idle limit is longer than the web apps') and label the device for admins.
final Map<String, String> _clientHeaders = {
  'X-Spotto-Client': 'app',
  'X-Spotto-Device': 'Spotto app (${defaultTargetPlatform.name})',
};

enum _Refresh { refreshed, rejected, failed }

class ApiClient {
  final Dio dio;
  final LocalStore store;
  void Function()? onUnauthorized;

  // No interceptors, so a failing refresh can't trigger another refresh.
  final Dio _refreshDio;
  // Single-flight: several requests hitting 401 together share one refresh.
  Future<_Refresh>? _refreshing;

  /// [adapter] replaces the HTTP transport for both clients (tests only).
  ApiClient(this.store, {@visibleForTesting HttpClientAdapter? adapter})
      : dio = Dio(BaseOptions(
          baseUrl: _validatedBaseUrl(),
          connectTimeout: const Duration(seconds: 5),
          receiveTimeout: const Duration(seconds: 10),
          headers: _clientHeaders,
        )),
        _refreshDio = Dio(BaseOptions(
          baseUrl: _validatedBaseUrl(),
          connectTimeout: const Duration(seconds: 5),
          receiveTimeout: const Duration(seconds: 10),
          headers: _clientHeaders,
        )) {
    if (adapter != null) {
      dio.httpClientAdapter = adapter;
      _refreshDio.httpClientAdapter = adapter;
    }
    dio.interceptors.add(InterceptorsWrapper(
      onRequest: (options, handler) {
        final token = store.getToken();
        if (token != null && token.isNotEmpty) {
          options.headers['Authorization'] = 'Bearer $token';
        }
        handler.next(options);
      },
      onError: (e, handler) async {
        final opts = e.requestOptions;
        final retryable =
            e.response?.statusCode == 401 && !opts.path.startsWith('/auth/') && opts.extra['retried'] != true;
        if (retryable) {
          final sent = (opts.headers['Authorization'] as String?)?.replaceFirst('Bearer ', '');
          final result = await _refreshSession(sent);
          if (result == _Refresh.refreshed) {
            opts.extra['retried'] = true;
            try {
              return handler.resolve(await dio.fetch(opts));
            } on DioException catch (retryError) {
              // A second 401 already went through onUnauthorized inside fetch.
              return handler.next(retryError);
            }
          }
          // Couldn't reach the server to refresh: keep the session and let
          // this request fail like any other network error.
          if (result == _Refresh.failed) return handler.next(e);
        }
        if (e.response?.statusCode == 401) onUnauthorized?.call();
        handler.next(e);
      },
    ));
  }

  Future<_Refresh> _refreshSession(String? sentToken) {
    // Another request refreshed while this one was in flight.
    final current = store.getToken();
    if (sentToken != null && current != null && current != sentToken) return Future.value(_Refresh.refreshed);
    return _refreshing ??= _refresh().whenComplete(() => _refreshing = null);
  }

  Future<_Refresh> _refresh() async {
    final refresh = store.getRefreshToken();
    if (refresh == null || refresh.isEmpty) return _Refresh.rejected;
    try {
      final res = await _refreshDio.post('/auth/refresh', data: {'refreshToken': refresh});
      final data = res.data as Map;
      await store.setToken(data['token'] as String, refreshToken: data['refreshToken'] as String?);
      return _Refresh.refreshed;
    } on DioException catch (e) {
      final status = e.response?.statusCode;
      if (status == 401 || status == 400) return _Refresh.rejected;
      debugPrint('ApiClient: session refresh failed (${status ?? e.type}): ${e.message}');
      return _Refresh.failed;
    } catch (e) {
      debugPrint('ApiClient: unexpected refresh response: $e');
      return _Refresh.failed;
    }
  }

  /// Gives a token from before server sessions existed a refresh token, so
  /// updating the app doesn't sign anyone out. No-op once it has one.
  Future<void> upgradeLegacySession() async {
    if (store.getToken() == null || store.getRefreshToken() != null) return;
    try {
      final res = await dio.post('/auth/session/upgrade');
      final data = res.data as Map;
      await store.setToken(data['token'] as String, refreshToken: data['refreshToken'] as String?);
    } catch (e) {
      debugPrint('ApiClient: legacy session upgrade failed: $e');
    }
  }

  /// Ends this device's server session (best effort: signing out locally
  /// must never wait on, or fail because of, the network).
  void logout() {
    final refresh = store.getRefreshToken();
    if (refresh == null) return;
    _refreshDio.post('/auth/logout', data: {'refreshToken': refresh}).ignore();
  }

  /// Extracts the server's {"error": "..."} message, or a generic fallback.
  static String errorMessage(Object e) {
    if (e is DioException) {
      final data = e.response?.data;
      if (data is Map && data['error'] is String) return data['error'] as String;
      if (e.type == DioExceptionType.connectionTimeout ||
          e.type == DioExceptionType.connectionError) {
        return 'Cannot connect to the server. Please check your internet connection and try again.';
      }
      // A response came back but wasn't the {"error": "..."} shape (e.g. a
      // plain-text 4xx/5xx body, or a proxy/500 page) — the user only ever
      // sees the generic fallback below, so at least log the real status/
      // body here rather than leaving this undiagnosable from the outside.
      debugPrint('ApiClient: unshaped error response ${e.response?.statusCode}: ${e.response?.data}');
    } else {
      // Not even a DioException — a bug in client-side code (bad cast,
      // null field, etc.) rather than a network/server problem. Same idea:
      // the user sees "Something went wrong," but this makes it traceable
      // in the Flutter logs instead of a dead end.
      debugPrint('ApiClient: non-network error: $e');
    }
    return 'Something went wrong. Please try again.';
  }
}
