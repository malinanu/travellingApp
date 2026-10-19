// "You're near a saved place" alerts. The phone's own region monitoring
// (Android GeofencingClient / iOS CLLocationManager, via native_geofence)
// watches circles around the traveller's saved places, so it works with the
// app closed and the location never leaves the phone. When one is entered,
// [onNearbyGeofence] runs in a background isolate, applies the cooldowns
// and shows a local notification.
//
// Everything here must work without Riverpod or the app's UI: the
// background isolate only has SharedPreferences and the plugins.

import 'dart:convert';
import 'dart:math' as math;

import 'package:flutter/foundation.dart';
import 'package:flutter_local_notifications/flutter_local_notifications.dart';
import 'package:native_geofence/native_geofence.dart';
import 'package:shared_preferences/shared_preferences.dart';

/// SharedPreferences keys shared with the foreground app.
const kNearbyEnabledKey = 'spotto_nearby_enabled';
const kNearbySpotsKey = 'spotto_nearby_spots';
const kNearbyHistoryKey = 'spotto_nearby_history';

/// Android allows 100 geofences per app; iOS 20 regions.
int maxNearbyRegions(TargetPlatform platform) => platform == TargetPlatform.iOS ? 20 : 90;

/// Food and drink alerts fire closer in: people decide on those on foot.
double nearbyRadiusMeters(String kind) => switch (kind) {
      'eat' || 'drink' => 250,
      'stay' => 300,
      _ => 400,
    };

/// A saved place the phone watches.
class NearbySpot {
  /// Geofence id: the saved-place id, or `trip_<stopId>` for a trip stop.
  final String id;
  final String name;
  final String kind;
  final double lat;
  final double lng;
  final String? placeId;
  final String? geoPlaceId;

  /// Where the traveller saved it from, e.g. "a Phuket reel".
  final String source;

  const NearbySpot({
    required this.id,
    required this.name,
    required this.kind,
    required this.lat,
    required this.lng,
    this.placeId,
    this.geoPlaceId,
    this.source = '',
  });

  Map<String, dynamic> toJson() => {
        'id': id,
        'name': name,
        'kind': kind,
        'lat': lat,
        'lng': lng,
        if (placeId != null) 'placeId': placeId,
        if (geoPlaceId != null) 'geoPlaceId': geoPlaceId,
        'source': source,
      };

  factory NearbySpot.fromJson(Map<String, dynamic> j) => NearbySpot(
        id: j['id'] as String,
        name: j['name'] as String? ?? '',
        kind: j['kind'] as String? ?? 'see',
        lat: (j['lat'] as num).toDouble(),
        lng: (j['lng'] as num).toDouble(),
        placeId: j['placeId'] as String?,
        geoPlaceId: j['geoPlaceId'] as String?,
        source: j['source'] as String? ?? '',
      );
}

double _metersBetween(double lat1, double lng1, double lat2, double lng2) {
  const r = 6371000.0;
  double rad(double d) => d * math.pi / 180;
  final dLat = rad(lat2 - lat1), dLng = rad(lng2 - lng1);
  final a = math.pow(math.sin(dLat / 2), 2) + math.cos(rad(lat1)) * math.cos(rad(lat2)) * math.pow(math.sin(dLng / 2), 2);
  return 2 * r * math.asin(math.sqrt(a));
}

/// The [max] spots to watch: the nearest to the traveller when their
/// position is known, otherwise the first ones (most recently saved).
List<NearbySpot> spotsToWatch(List<NearbySpot> spots, {double? lat, double? lng, required int max}) {
  final seen = <String>{};
  final unique = [for (final s in spots) if (seen.add(s.id)) s];
  if (lat != null && lng != null) {
    unique.sort((a, b) => _metersBetween(lat, lng, a.lat, a.lng).compareTo(_metersBetween(lat, lng, b.lat, b.lng)));
  }
  return unique.take(max).toList();
}

/// When alerts were sent, to keep them rare: one per place per day and at
/// most three a day overall.
class NearbyAlertHistory {
  static const perPlaceCooldown = Duration(hours: 24);
  static const dailyCap = 3;

  final Map<String, int> lastByPlace; // spot id -> ms since epoch
  final List<int> sent; // ms since epoch, last 24 h

  NearbyAlertHistory({Map<String, int>? lastByPlace, List<int>? sent})
      : lastByPlace = lastByPlace ?? {},
        sent = sent ?? [];

  bool shouldAlert(String spotId, DateTime now) {
    final ms = now.millisecondsSinceEpoch;
    final last = lastByPlace[spotId];
    if (last != null && ms - last < perPlaceCooldown.inMilliseconds) return false;
    final today = sent.where((t) => ms - t < const Duration(hours: 24).inMilliseconds).length;
    return today < dailyCap;
  }

  void record(String spotId, DateTime now) {
    final ms = now.millisecondsSinceEpoch;
    lastByPlace[spotId] = ms;
    sent
      ..add(ms)
      ..removeWhere((t) => ms - t >= const Duration(hours: 24).inMilliseconds);
  }

  Map<String, dynamic> toJson() => {'lastByPlace': lastByPlace, 'sent': sent};

  factory NearbyAlertHistory.fromJson(Map<String, dynamic> j) => NearbyAlertHistory(
        lastByPlace: (j['lastByPlace'] as Map? ?? {}).map((k, v) => MapEntry(k as String, (v as num).toInt())),
        sent: [for (final t in (j['sent'] as List? ?? [])) (t as num).toInt()],
      );
}

/// Notification title and text, phrased by what kind of place it is.
(String, String) nearbyAlertText(NearbySpot s) {
  final saved = s.source.isNotEmpty ? 'You saved it from ${s.source}.' : 'It\'s on your saved list.';
  return switch (s.kind) {
    'eat' => ('🍜 You\'re near ${s.name}', '$saved A good stop to eat while you\'re here.'),
    'drink' => ('🍹 You\'re near ${s.name}', '$saved Good for a drink nearby.'),
    'stay' => ('🛏 You\'re near ${s.name}', '$saved Take a look while you\'re passing.'),
    'shop' => ('🛍 You\'re near ${s.name}', '$saved Worth a browse while you\'re here.'),
    'do' => ('✨ You\'re near ${s.name}', '$saved You can do it while you\'re here.'),
    _ => ('📍 You\'re near ${s.name}', '$saved Worth a visit while you\'re here.'),
  };
}

// ---- notifications ----

final _notifications = FlutterLocalNotificationsPlugin();
bool _notificationsReady = false;

const _channel = AndroidNotificationDetails(
  'nearby_saved',
  'Nearby saved places',
  channelDescription: 'A heads-up when you\'re close to a place you saved.',
  importance: Importance.high,
  priority: Priority.high,
);

/// Sets up local notifications. [onTap] receives the tapped alert's payload
/// (foreground isolate only).
Future<void> initNearbyNotifications({void Function(String? payload)? onTap}) async {
  if (_notificationsReady) return;
  await _notifications.initialize(
    settings: const InitializationSettings(
      android: AndroidInitializationSettings('@mipmap/ic_launcher'),
      iOS: DarwinInitializationSettings(
        requestAlertPermission: false,
        requestBadgePermission: false,
        requestSoundPermission: false,
      ),
    ),
    onDidReceiveNotificationResponse: onTap == null ? null : (r) => onTap(r.payload),
  );
  _notificationsReady = true;
}

/// Payload of the alert that launched the app, if one did.
Future<String?> nearbyLaunchPayload() async {
  final details = await _notifications.getNotificationAppLaunchDetails();
  return details?.didNotificationLaunchApp == true ? details!.notificationResponse?.payload : null;
}

/// Asks to show notifications (Android 13+, iOS). Returns whether allowed.
Future<bool> requestNearbyNotificationPermission() async {
  final android = _notifications.resolvePlatformSpecificImplementation<AndroidFlutterLocalNotificationsPlugin>();
  if (android != null) return await android.requestNotificationsPermission() ?? false;
  final ios = _notifications.resolvePlatformSpecificImplementation<IOSFlutterLocalNotificationsPlugin>();
  if (ios != null) return await ios.requestPermissions(alert: true, sound: true) ?? false;
  return true;
}

Future<void> _showAlert(NearbySpot s) async {
  final (title, body) = nearbyAlertText(s);
  await _notifications.show(
    id: s.id.hashCode & 0x7fffffff,
    title: title,
    body: body,
    notificationDetails: const NotificationDetails(android: _channel, iOS: DarwinNotificationDetails()),
    payload: jsonEncode({'spotId': s.id, 'placeId': ?s.placeId, 'geoPlaceId': ?s.geoPlaceId}),
  );
}

// ---- geofence callback ----

/// Runs in a background isolate when the phone enters a watched region.
@pragma('vm:entry-point')
Future<void> onNearbyGeofence(GeofenceCallbackParams params) async {
  if (params.event != GeofenceEvent.enter) return;
  try {
    final prefs = await SharedPreferences.getInstance();
    await prefs.reload();
    if (!(prefs.getBool(kNearbyEnabledKey) ?? false)) return;

    final spots = {
      for (final s in (jsonDecode(prefs.getString(kNearbySpotsKey) ?? '[]') as List))
        (s as Map)['id'] as String: NearbySpot.fromJson(s.cast<String, dynamic>()),
    };
    final history = NearbyAlertHistory.fromJson(
        (jsonDecode(prefs.getString(kNearbyHistoryKey) ?? '{}') as Map).cast<String, dynamic>());

    await initNearbyNotifications();
    final now = DateTime.now();
    for (final g in params.geofences) {
      final spot = spots[g.id];
      if (spot == null || !history.shouldAlert(spot.id, now)) continue;
      await _showAlert(spot);
      history.record(spot.id, now);
    }
    await prefs.setString(kNearbyHistoryKey, jsonEncode(history.toJson()));
  } catch (e, st) {
    debugPrint('nearby alert failed: $e\n$st');
  }
}

/// Builds the geofence the phone watches for one spot.
Geofence nearbyGeofence(NearbySpot s) => Geofence(
      id: s.id,
      location: Location(latitude: s.lat, longitude: s.lng),
      radiusMeters: nearbyRadiusMeters(s.kind),
      triggers: const {GeofenceEvent.enter},
      // No alert for places the traveller is already standing at when the
      // list syncs; only for ones they walk up to.
      iosSettings: const IosGeofenceSettings(initialTrigger: false),
      androidSettings: const AndroidGeofenceSettings(
        initialTriggers: {},
        notificationResponsiveness: Duration(minutes: 2),
      ),
    );
