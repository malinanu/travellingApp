// Device push notification service for Spotto.
// Handles FCM device token registration, unregistration on logout, and
// local notification alerts for offer claims, vibe updates, and trip reminders.

import 'dart:developer' as developer;
import 'package:flutter/foundation.dart';
import 'package:firebase_core/firebase_core.dart';
import 'package:firebase_messaging/firebase_messaging.dart';

import '../../firebase_options.dart';
import 'api_client.dart';

class PushNotificationService {
  static final PushNotificationService instance = PushNotificationService._();
  PushNotificationService._();

  String? _deviceToken;
  bool _initialized = false;
  static const _stepTimeout = Duration(seconds: 10);

  /// Initializes device push notification hooks.
  ///
  /// Must never throw or block app startup: on iOS builds without the push
  /// entitlement (e.g. an unsigned sideloaded .ipa) no APNs token can be
  /// issued, so these calls can fail or hang. Push is best-effort — the app
  /// works without it — so any failure here is logged and swallowed.
  Future<void> initialize(ApiClient api) async {
    if (_initialized) return;
    _initialized = true;

    try {
      await Firebase.initializeApp(
        options: DefaultFirebaseOptions.currentPlatform,
      );

      final messaging = FirebaseMessaging.instance;
      await messaging.requestPermission().timeout(_stepTimeout);
      // register()/onRegistered (the replacement) yields a Firebase
      // Installation ID, not the FCM registration token the server sends
      // to, so keep getToken() until the server supports FID direct-send.
      // ignore: deprecated_member_use
      _deviceToken = await messaging.getToken().timeout(_stepTimeout);

      if (_deviceToken != null) {
        try {
          await api.dio.post('/me/push-token', data: {'deviceToken': _deviceToken});
        } catch (e) {
          developer.log('[PushNotifications] Failed to register token: $e');
        }
      }

      FirebaseMessaging.onMessage.listen((RemoteMessage message) {
        developer.log('[PushNotificationReceived] ${message.notification?.title} — ${message.notification?.body}');
      });

      if (kDebugMode) {
        developer.log('[PushNotifications] Device push token initialized: $_deviceToken');
      }
    } catch (e, st) {
      // Allow a later retry (e.g. after the user grants permission).
      _initialized = false;
      developer.log('[PushNotifications] Push unavailable, continuing without it: $e',
          stackTrace: st);
    }
  }

  /// Active device push token.
  String? get deviceToken => _deviceToken;

  /// Formats push payload for local presentation.
  void handleIncomingPayload(Map<String, dynamic> payload) {
    final title = payload['title'] as String? ?? 'Spotto Alert';
    final body = payload['body'] as String? ?? '';
    developer.log('[PushNotificationReceived] $title — $body');
  }
}
