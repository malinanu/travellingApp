// Thin dio wrapper for the Spotto Go API.
//
// Base URL comes from --dart-define=API_URL, defaulting to the Android
// emulator's host loopback. The JWT is kept in LocalStore under
// StoreKeys.token and attached as a Bearer header; a 401 triggers the
// onUnauthorized callback (wired to sign-out in main.dart).

import 'package:dio/dio.dart';

import '../storage/local_store.dart';

const apiBaseUrl = String.fromEnvironment('API_URL', defaultValue: 'http://10.0.2.2:8080');

class ApiClient {
  final Dio dio;
  final LocalStore store;
  void Function()? onUnauthorized;

  ApiClient(this.store)
      : dio = Dio(BaseOptions(
          baseUrl: apiBaseUrl,
          connectTimeout: const Duration(seconds: 5),
          receiveTimeout: const Duration(seconds: 10),
        )) {
    dio.interceptors.add(InterceptorsWrapper(
      onRequest: (options, handler) {
        final token = store.getString(StoreKeys.token);
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
    }
    return 'Something went wrong. Please try again.';
  }
}
