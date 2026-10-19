// Routes things that arrive from outside the app: a reel link shared from
// Instagram/TikTok (Share → Spotto, Android), and a tapped push notification.

import 'package:flutter/foundation.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:receive_sharing_intent/receive_sharing_intent.dart';

import '../../app/router.dart';
import '../../data/models/inspiration.dart';

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
