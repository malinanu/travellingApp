// Thin dio wrapper for the Spotto Go API.
//
// Base URL comes from --dart-define=API_URL, defaulting to the Android
// emulator's host loopback. The JWT is kept in LocalStore's secure storage
// and attached as a Bearer header; a 401 triggers the onUnauthorized
// callback (wired to sign-out in main.dart).

import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart' show debugPrint, kReleaseMode;

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

class ApiClient {
  final Dio dio;
  final LocalStore store;
  void Function()? onUnauthorized;

  ApiClient(this.store)
      : dio = Dio(BaseOptions(
          baseUrl: _validatedBaseUrl(),
          connectTimeout: const Duration(seconds: 5),
          receiveTimeout: const Duration(seconds: 10),
        )) {
    dio.interceptors.add(InterceptorsWrapper(
      onRequest: (options, handler) {
        final token = store.getToken();
        if (token != null && token.isNotEmpty) {
          options.headers['Authorization'] = 'Bearer $token';
        }
        handler.next(options);
      },
      onError: (e, handler) {
        if (e.response?.statusCode == 401) onUnauthorized?.call();
        handler.next(e);
      },
    ));
  }

  /// Extracts the server's {"error": "..."} message, or a generic fallback.
  static String errorMessage(Object e) {
    if (e is DioException) {
      final data = e.response?.data;
      if (data is Map && data['error'] is String) return data['error'] as String;
      if (e.type == DioExceptionType.connectionTimeout ||
          e.type == DioExceptionType.connectionError) {
        return 'Cannot reach the server — is the API running?';
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
