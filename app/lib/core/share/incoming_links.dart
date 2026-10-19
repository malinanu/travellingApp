// Routes things that arrive from outside the app: a reel link shared from
// Instagram/TikTok (Share → Spotto, Android), and a tapped push notification.

import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:receive_sharing_intent/receive_sharing_intent.dart';

import '../../app/router.dart';
import '../../data/models/inspiration.dart';
import '../../data/models/models.dart' show Place;
import '../../data/repositories/repositories.dart';
import '../../features/traveller/place_detail/business_drawer.dart';
import '../location/nearby_alerts.dart';

class IncomingLinks {
  IncomingLinks._();

  /// Starts listening. Call once, after runApp.
  static void start(ProviderContainer container) {
    if (defaultTargetPlatform != TargetPlatform.android) return; // iOS Share Extension not set up yet
    void handle(List<SharedMediaFile> files) {
      final text = files.map((f) => '${f.path} ${f.message ?? ''}').join(' ');
      final link = findSupportedLink(text);
      if (link == null) return;
      // After the first frame, so the router has settled its initial route.
      WidgetsBinding.instance.addPostFrameCallback((_) {
        container.read(routerProvider).push('/t/inspiration?add=${Uri.encodeQueryComponent(link)}');
      });
      ReceiveSharingIntent.instance.reset();
    }

    ReceiveSharingIntent.instance.getInitialMedia().then(handle).catchError((Object e) {
      debugPrint('shared link (launch) failed: $e');
    });
    ReceiveSharingIntent.instance.getMediaStream().listen(handle, onError: (Object e) {
      debugPrint('shared link failed: $e');
    });
  }

  /// Handles taps on "you're near a saved place" alerts, including one
  /// that launched the app.
  static Future<void> startNearbyAlerts(ProviderContainer container) async {
    try {
      await initNearbyNotifications(onTap: (payload) => _openNearby(container, payload));
      final launch = await nearbyLaunchPayload();
      if (launch != null) _openNearby(container, launch);
    } catch (e) {
      debugPrint('nearby alerts setup failed: $e');
    }
  }

  /// Opens the place an alert was about: the Spotto drawer for catalog
  /// places, otherwise the board's Saved tab.
  static void _openNearby(ProviderContainer container, String? payload) {
    final data = payload == null ? null : jsonDecode(payload);
    final placeId = data is Map ? data['placeId'] as String? : null;
    WidgetsBinding.instance.addPostFrameCallback((_) async {
      if (placeId != null) {
        try {
          final Place place = container.read(placesProvider).where((p) => p.id == placeId).firstOrNull ??
              await container.read(placeDetailsProvider(placeId).future);
          final ctx = rootNavigatorKey.currentContext;
          if (ctx != null && ctx.mounted) {
            await showBusinessDrawer(ctx, place);
            return;
          }
        } catch (e) {
          debugPrint('nearby alert: open place $placeId failed: $e');
        }
      }
      container.read(routerProvider).push('/t/inspiration?tab=saved');
    });
  }

  /// Opens the screen a push notification points at.
  static void openNotification(ProviderContainer container, Map<String, dynamic> data) {
    final route = switch (data['type']) {
      'inspiration_ready' when data['linkId'] is String => '/t/inspiration/${data['linkId']}',
      _ => null,
    };
    if (route == null) return;
    WidgetsBinding.instance.addPostFrameCallback((_) => container.read(routerProvider).push(route));
  }
}
