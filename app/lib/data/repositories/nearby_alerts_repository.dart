// Foreground side of nearby saved-place alerts (see
// core/location/nearby_alerts.dart): turning them on (permissions), and
// keeping the phone's watched regions in step with the saved places and
// trip stops — on start, when the app comes back, and whenever they change.

import 'dart:async';
import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:flutter/widgets.dart' show AppLifecycleListener;
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:geolocator/geolocator.dart';
import 'package:native_geofence/native_geofence.dart';

import '../../core/location/nearby_alerts.dart';
import '../catalog/trip_catalog.dart';
import '../models/inspiration.dart';
import 'inspiration_repository.dart';
import 'repositories.dart';

enum NearbyAlertsIssue {
  /// Location is only allowed while the app is open, so alerts can't fire
  /// with the app closed.
  needsAlwaysLocation,
  locationDenied,
  notificationsDenied,
}

class NearbyAlertsState {
  final bool enabled;
  final int watching;
  final NearbyAlertsIssue? issue;
  const NearbyAlertsState({this.enabled = false, this.watching = 0, this.issue});
}

/// Result of [NearbyAlertsNotifier.enable], for the opt-in UI to explain.
enum NearbyEnableResult { enabled, openedSettings, locationDenied, notificationsDenied }

class NearbyAlertsNotifier extends Notifier<NearbyAlertsState> {
  Timer? _debounce;
  bool _pluginReady = false;

  @override
  NearbyAlertsState build() {
    final enabled = ref.read(localStoreProvider).getBool(kNearbyEnabledKey);
    ref.listen(savedPlacesProvider, (_, _) => _scheduleSync());
    ref.listen(tripProvider, (_, _) => _scheduleSync());
    final lifecycle = AppLifecycleListener(onResume: _scheduleSync);
    ref.onDispose(() {
      _debounce?.cancel();
      lifecycle.dispose();
    });
    if (enabled) _scheduleSync();
    return NearbyAlertsState(enabled: enabled);
  }

  void _scheduleSync() {
    _debounce?.cancel();
    _debounce = Timer(const Duration(seconds: 2), () {
      if (ref.mounted) sync();
    });
  }

  /// Turns alerts on: location (then "all the time"), notifications, then
  /// starts watching.
  Future<NearbyEnableResult> enable() async {
    var perm = await Geolocator.checkPermission();
    if (perm == LocationPermission.denied) perm = await Geolocator.requestPermission();
    if (perm == LocationPermission.denied || perm == LocationPermission.deniedForever) {
      return NearbyEnableResult.locationDenied;
    }
    if (!await requestNearbyNotificationPermission()) return NearbyEnableResult.notificationsDenied;

    await ref.read(localStoreProvider).setBool(kNearbyEnabledKey, true);
    state = NearbyAlertsState(enabled: true, watching: state.watching);
    if (perm == LocationPermission.whileInUse) {
      // iOS asks to upgrade on a second request; Android 11+ only offers
      // "Allow all the time" in Settings.
      if (defaultTargetPlatform == TargetPlatform.iOS) perm = await Geolocator.requestPermission();
      if (perm != LocationPermission.always) {
        await Geolocator.openAppSettings();
        return NearbyEnableResult.openedSettings;
      }
    }
    await sync();
    return NearbyEnableResult.enabled;
  }

  Future<void> disable() async {
    await ref.read(localStoreProvider).setBool(kNearbyEnabledKey, false);
    state = const NearbyAlertsState();
    try {
      await _ensurePlugin();
      await NativeGeofenceManager.instance.removeAllGeofences();
    } catch (e) {
      debugPrint('nearby alerts: remove regions failed: $e');
    }
  }

  /// Re-registers the regions around the nearest saved places.
  Future<void> sync() async {
    if (!ref.read(localStoreProvider).getBool(kNearbyEnabledKey)) return;
    final spots = currentSpots();
    await ref.read(localStoreProvider).setString(kNearbySpotsKey, jsonEncode([for (final s in spots) s.toJson()]));

    final perm = await Geolocator.checkPermission();
    if (perm == LocationPermission.denied || perm == LocationPermission.deniedForever) {
      state = const NearbyAlertsState(enabled: true, issue: NearbyAlertsIssue.locationDenied);
      return;
    }
    Position? pos;
    try {
      pos = await Geolocator.getLastKnownPosition();
    } catch (_) {}
    final watch = spotsToWatch(spots, lat: pos?.latitude, lng: pos?.longitude, max: maxNearbyRegions(defaultTargetPlatform));

    var registered = 0;
    try {
      await _ensurePlugin();
      await NativeGeofenceManager.instance.removeAllGeofences();
      for (final s in watch) {
        try {
          await NativeGeofenceManager.instance.createGeofence(nearbyGeofence(s), onNearbyGeofence);
          registered++;
        } on NativeGeofenceException catch (e) {
          if (e.code == NativeGeofenceErrorCode.missingLocationPermission ||
              e.code == NativeGeofenceErrorCode.missingBackgroundLocationPermission) {
            break;
          }
          debugPrint('nearby alerts: region ${s.id} failed: ${e.code} ${e.message}');
        }
      }
    } catch (e) {
      debugPrint('nearby alerts: sync failed: $e');
    }
    if (!ref.mounted) return;
    state = NearbyAlertsState(
      enabled: true,
      watching: registered,
      issue: perm != LocationPermission.always && watch.isNotEmpty ? NearbyAlertsIssue.needsAlwaysLocation : null,
    );
  }

  Future<void> _ensurePlugin() async {
    if (_pluginReady) return;
    await NativeGeofenceManager.instance.initialize();
    _pluginReady = true;
  }

  /// Saved places plus trip stops that have a real location.
  List<NearbySpot> currentSpots() {
    final links = {for (final l in ref.read(boardLinksProvider)) l.id: l};
    final spots = <NearbySpot>[];
    for (final s in ref.read(savedPlacesProvider)) {
      final lat = s.lat, lng = s.lng;
      if (lat == null || lng == null || (lat == 0 && lng == 0)) continue;
      final link = s.sourceLinkId == null ? null : links[s.sourceLinkId];
      spots.add(NearbySpot(
        id: s.id,
        name: s.name,
        kind: s.kind,
        lat: lat,
        lng: lng,
        placeId: s.placeId,
        geoPlaceId: s.geoPlaceId,
        source: link == null ? '' : (link.region.isNotEmpty ? 'a ${link.region} reel' : 'a reel'),
      ));
    }
    final catalog = {for (final p in ref.read(placesProvider)) p.id: p};
    for (final day in ref.read(tripProvider).days) {
      for (final stop in day.stops) {
        final custom = stop.custom;
        if (custom != null && !_isDefaultStopLocation(custom)) {
          spots.add(NearbySpot(
            id: 'trip_${stop.id}',
            name: custom.name,
            kind: _kindFromIcon(custom.icon),
            lat: custom.lat,
            lng: custom.lng,
            source: 'your trip',
          ));
        } else if (catalog[stop.placeId] case final p?) {
          spots.add(NearbySpot(
            id: 'trip_${stop.id}',
            name: p.name,
            kind: 'see',
            lat: p.lat,
            lng: p.lng,
            placeId: p.id,
            source: 'your trip',
          ));
        }
      }
    }
    return spots;
  }
}

/// Custom stops without a real location sit on CustomStopInfo's default.
bool _isDefaultStopLocation(CustomStopInfo c) => c.lat == const CustomStopInfo(name: '').lat && c.lng == const CustomStopInfo(name: '').lng;

String _kindFromIcon(String icon) =>
    placeKinds.firstWhere((k) => kindIcon(k) == icon, orElse: () => 'see');

final nearbyAlertsProvider = NotifierProvider<NearbyAlertsNotifier, NearbyAlertsState>(NearbyAlertsNotifier.new);
