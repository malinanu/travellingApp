// State layer: Riverpod notifiers backed by the Spotto Go API, with the last
// good server response cached in LocalStore as an offline snapshot. Mutation
// method signatures are unchanged from the mock era so screens barely change.

import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
// StateProvider moved here in Riverpod 3 (still fully supported, just
// signalled as the "legacy" API) — see placesLocationStatusProvider/
// timeFilterProvider/groupModeProvider below, the only StateProviders in
// this app.
import 'package:flutter_riverpod/legacy.dart';
import 'package:geolocator/geolocator.dart';

import '../../core/api/api_client.dart';
import '../../core/storage/local_store.dart';
import '../mock/places.dart' as seed;
import '../mock/trip_catalog.dart';
import '../models/models.dart';
import '../vendor_categories.dart';

/// Parses a string-list field from server JSON that may still be in an
/// older single-string shape (e.g. travelStyle before it became
/// multi-select) — `v as List?` is a hard cast in Dart that throws on a
/// non-null, wrong-typed value rather than returning null, so a server
/// that hasn't redeployed a schema change yet (or has redeployed and the
/// app hasn't, in either direction) could crash login/profile instead of
/// just showing slightly-stale data. Never throws.
List<String> parseStringList(dynamic v) {
  if (v is List) return v.cast<String>();
  if (v is String && v.isNotEmpty) return [v];
  return const [];
}

// ── device location (shared best-effort helper) ────────────────────

/// Weligama Bay — the app's home base, used as a fallback location when
/// device position is denied/unavailable (never a forced default — just
/// what the map/weather show when we truly can't get the user's own
/// position).
const kFallbackLat = 5.9724;
const kFallbackLng = 80.4292;

typedef LatLngAt = ({double lat, double lng});

enum DeviceLocationResult { granted, deniedForever, unavailable }

// bestEffortPosition has 4 independent callers (devicePositionProvider,
// weatherLocationProvider, PlacesNotifier._currentPosition,
// NearbyTravellersNotifier._tick) that can all fire within the same frame
// at cold start. Geolocator.requestPermission() has no Dart-side timeout,
// and on Android the plugin stores the pending native callback in a single
// field — if two requests race, the second silently overwrites the first,
// and the first caller's Future never resolves and never throws (looks
// exactly like an infinite spinner with nothing in the logs). Memoizing
// the in-flight call here means concurrent callers share one real
// permission request instead of racing separate ones.
Future<(Position?, DeviceLocationResult)>? _inFlightPosition;

Future<(Position?, DeviceLocationResult)> bestEffortPosition() {
  return _inFlightPosition ??= _resolvePosition().whenComplete(() => _inFlightPosition = null);
}

Future<(Position?, DeviceLocationResult)> _resolvePosition() async {
  try {
    var permission = await Geolocator.checkPermission()
        .timeout(const Duration(seconds: 5), onTimeout: () => LocationPermission.denied);
    if (permission == LocationPermission.denied) {
      // Belt-and-suspenders alongside the single-flight de-dup above: even
      // a single, non-racing request could in principle hang on some
      // device, so this still can't be allowed to wait forever.
      permission = await Geolocator.requestPermission()
          .timeout(const Duration(seconds: 20), onTimeout: () => LocationPermission.denied);
    }
    if (permission == LocationPermission.deniedForever) {
      return (null, DeviceLocationResult.deniedForever);
    }
    if (permission == LocationPermission.denied) {
      return (null, DeviceLocationResult.unavailable);
    }
    if (!await Geolocator.isLocationServiceEnabled()) {
      return (null, DeviceLocationResult.unavailable);
    }
    // Check cached last known position first for instant startup resolution (<5ms)
    final lastPos = await Geolocator.getLastKnownPosition();
    if (lastPos != null) {
      return (lastPos, DeviceLocationResult.granted);
    }

    final pos = await Geolocator.getCurrentPosition(
      locationSettings: const LocationSettings(accuracy: LocationAccuracy.medium, timeLimit: Duration(seconds: 6)),
    );
    return (pos, DeviceLocationResult.granted);
  } catch (_) {
    return (null, DeviceLocationResult.unavailable);
  }
}

/// Guaranteed lat/lng — the user's real position, or Weligama Bay when
/// denied/unavailable — paired with the real [DeviceLocationResult] so a
/// caller that actually needs to tell "still resolving" apart from
/// "resolved to a fallback because permission was denied" apart from "a
/// genuine granted position" doesn't have to re-derive it (see
/// _VibeCheck in business_drawer.dart, which gates check-ins on `granted`
/// specifically rather than treating the Weligama Bay fallback as a real
/// position for geofence math). Callers that just want map-centering/
/// weather lat-lng and don't care why can ignore the second element.
final devicePositionProvider = FutureProvider<(LatLngAt, DeviceLocationResult)>((ref) async {
  final (pos, result) = await bestEffortPosition();
  if (pos == null) return ((lat: kFallbackLat, lng: kFallbackLng), result);
  return ((lat: pos.latitude, lng: pos.longitude), DeviceLocationResult.granted);
});

/// Whether location permission is already granted right now — a plain
/// Geolocator.checkPermission() read, never requestPermission(), so
/// watching this can't itself pop an OS prompt. GoogleMap's
/// myLocationEnabled must not be set to true before permission is
/// actually confirmed granted: on Android the map's tile layer can fail
/// to initialize at all (not just skip the blue dot) if it's told to show
/// a location layer it doesn't have permission for yet.
final locationPermissionProvider = FutureProvider<bool>((ref) async {
  try {
    final permission = await Geolocator.checkPermission()
        .timeout(const Duration(seconds: 5), onTimeout: () => LocationPermission.denied);
    return permission == LocationPermission.always || permission == LocationPermission.whileInUse;
  } catch (_) {
    return false;
  }
});

/// Overridden in main() with the initialized instance.
final localStoreProvider = Provider<LocalStore>((ref) => throw UnimplementedError());

final apiProvider = Provider<ApiClient>((ref) {
  final client = ApiClient(ref.watch(localStoreProvider));
  client.onUnauthorized = () => ref.read(authProvider.notifier).signOut();
  return client;
});

// ── auth ──────────────────────────────────────────────────────────

class AuthNotifier extends Notifier<bool> {
  @override
  bool build() {
    final token = ref.read(localStoreProvider).getToken();
    return token != null && token.isNotEmpty;
  }

  /// Returns null on success, or a user-facing error message.
  Future<String?> login(String email, String password) async {
    try {
      final res = await ref.read(apiProvider).dio.post('/auth/login',
          data: {'email': email, 'password': password});
      await _adoptSession(res.data as Map<String, dynamic>);
      return null;
    } catch (e) {
      return ApiClient.errorMessage(e);
    }
  }

  Future<String?> register(String name, String email, String password) async {
    try {
      final res = await ref.read(apiProvider).dio.post('/auth/register',
          data: {'name': name, 'email': email, 'password': password});
      await _adoptSession(res.data as Map<String, dynamic>);
      return null;
    } catch (e) {
      return ApiClient.errorMessage(e);
    }
  }

  Future<String?> appleAuth(String appleId, {String? email, String? name}) async {
    try {
      final res = await ref.read(apiProvider).dio.post('/auth/apple', data: {
        'appleId': appleId,
        if (email != null) 'email': email,
        if (name != null) 'name': name,
      });
      await _adoptSession(res.data as Map<String, dynamic>);
      return null;
    } catch (e) {
      return ApiClient.errorMessage(e);
    }
  }

  Future<String?> forgotPassword(String email) async {
    try {
      await ref.read(apiProvider).dio.post('/auth/forgot-password', data: {'email': email});
      return null;
    } catch (e) {
      return ApiClient.errorMessage(e);
    }
  }

  Future<String?> resetPassword(String token, String newPassword) async {
    try {
      await ref.read(apiProvider).dio.post('/auth/reset-password', data: {
        'token': token,
        'newPassword': newPassword,
      });
      return null;
    } catch (e) {
      return ApiClient.errorMessage(e);
    }
  }

  Future<void> _adoptSession(Map<String, dynamic> data) async {
    final store = ref.read(localStoreProvider);
    await store.setToken(data['token'] as String);
    await store.setBool(StoreKeys.auth, true);

    // Adopt server-side prefs so a returning user skips onboarding.
    final user = data['user'] as Map<String, dynamic>? ?? const {};
    final styles = parseStringList(user['travelStyle']);
    if (styles.isNotEmpty) {
      ref.read(prefsProvider.notifier).saveLocal(UserPrefs(
            styles: styles,
            interests: [for (final i in (user['interests'] as List? ?? const [])) i as String],
          ));
    }
    state = true;
    ref.invalidate(claimsProvider);
    ref.invalidate(tripProvider);
    ref.invalidate(groupProvider);
    ref.invalidate(placesProvider);
  }

  void signOut() {
    state = false;
    final store = ref.read(localStoreProvider);
    store.clearToken();
    store.setBool(StoreKeys.auth, false);
    store.remove(StoreKeys.prefs);
    store.remove(StoreKeys.claims);
    store.remove(StoreKeys.trip);
    store.remove(StoreKeys.group);
    ref.invalidate(prefsProvider);
    ref.invalidate(claimsProvider);
    ref.invalidate(tripProvider);
    ref.invalidate(groupProvider);
  }
}

final authProvider = NotifierProvider<AuthNotifier, bool>(AuthNotifier.new);

// ── current user (email, totpEnabled, shareLocation, …) ────────────

/// Raw `/me` response — read directly by the security/privacy profile pages
/// rather than modeled, since those are the only consumers so far.
class MeNotifier extends Notifier<Map<String, dynamic>?> {
  @override
  Map<String, dynamic>? build() {
    Future.microtask(refresh);
    return null;
  }

  Future<void> refresh() async {
    if (!ref.read(authProvider)) return;
    try {
      final res = await ref.read(apiProvider).dio.get('/me');
      final data = (res.data as Map).cast<String, dynamic>();
      state = data;
      seed.me = Traveler.fromMe(data);
      // prefsProvider is otherwise local-storage-only until the user
      // explicitly saves — hydrate it from the server's real values here so
      // a fresh install/login shows the traveller's actual saved
      // style/interests instead of blank/default onboarding state.
      final styles = parseStringList(data['travelStyle']);
      if (styles.isNotEmpty) {
        final interests = (data['interests'] as List?)?.cast<String>() ?? const [];
        ref.read(prefsProvider.notifier).saveLocal(UserPrefs(styles: styles, interests: interests));
      }
    } catch (_) {
      // Offline — leave the previous state (or null) as-is.
    }
  }
}

final meProvider = NotifierProvider<MeNotifier, Map<String, dynamic>?>(MeNotifier.new);

// ── prefs (travel style + interests) ──────────────────────────────

class PrefsNotifier extends Notifier<UserPrefs?> {
  @override
  UserPrefs? build() {
    final json = ref.read(localStoreProvider).getJson(StoreKeys.prefs);
    return json == null ? null : UserPrefs.fromJson(json);
  }

  void saveLocal(UserPrefs prefs) {
    state = prefs;
    ref.read(localStoreProvider).setJson(StoreKeys.prefs, prefs.toJson());
  }

  /// Persists locally and pushes to the server (fire-and-forget).
  void save(UserPrefs prefs) {
    saveLocal(prefs);
    ref
        .read(apiProvider)
        .dio
        .put('/me', data: {'travelStyle': prefs.styles, 'interests': prefs.interests})
        .ignore();
  }
}

final prefsProvider = NotifierProvider<PrefsNotifier, UserPrefs?>(PrefsNotifier.new);

// ── places (server-owned, seed as offline fallback) ───────────────

/// Whether the last places fetch was sorted by device location, and why not
/// when it wasn't — lets the UI tell the user instead of silently showing
/// an unsorted list that looks identical to "the feature is broken".
enum PlacesLocationStatus { unknown, sorted, deniedForever, unavailable }

final placesLocationStatusProvider =
    StateProvider<PlacesLocationStatus>((ref) => PlacesLocationStatus.unknown);

/// Parses each raw place row independently, skipping (and logging) any
/// single record that fails instead of throwing and losing the whole batch
/// — one malformed record from the server or a stale local cache must never
/// blank out the entire live places list.
List<Place> _parsePlaces(List raw) {
  final places = <Place>[];
  for (final p in raw) {
    try {
      places.add(Place.fromJson((p as Map).cast<String, dynamic>()));
    } catch (e) {
      debugPrint('placesProvider: skipping unparseable place: $e');
    }
  }
  return places;
}

class PlacesNotifier extends Notifier<List<Place>> {
  @override
  List<Place> build() {
    Future.microtask(refresh);
    final cached = ref.read(localStoreProvider).getJsonList(StoreKeys.places);
    if (cached != null && cached.isNotEmpty) {
      final places = _parsePlaces(cached);
      if (places.isNotEmpty) {
        _syncTripCatalog(places);
        return places;
      }
    }
    return seed.places;
  }

  /// Replaces the trip planner's catalog with real places, once available —
  /// the hardcoded tripCatalog list otherwise stays as the pre-fetch
  /// fallback (see trip_catalog.dart's top-of-file comment).
  void _syncTripCatalog(List<Place> places) {
    tripCatalog = [for (final p in places) TripCatalogItem.fromPlace(p)];
  }

  Future<void> refresh() async {
    try {
      final pos = await _currentPosition();
      final res = await ref.read(apiProvider).dio.get('/places',
          queryParameters: pos == null ? null : {'lat': pos.latitude, 'lng': pos.longitude});
      final list = res.data as List;
      if (list.isEmpty) return;
      final places = _parsePlaces(list);
      if (places.isEmpty) return;
      state = places;
      _syncTripCatalog(places);
      await ref.read(localStoreProvider).setJsonList(StoreKeys.places, list);
    } catch (e) {
      // Offline or unauthenticated — keep the cached/seed state, but log so
      // a real failure (bad API_URL, server error) isn't indistinguishable
      // from "not logged in yet". _currentPosition() may have already set
      // status to sorted before this request failed, which would leave the
      // UI treating stale cached places as freshly location-sorted.
      ref.read(placesLocationStatusProvider.notifier).state = PlacesLocationStatus.unavailable;
      debugPrint('placesProvider refresh failed: $e');
    }
  }

  /// Best-effort device position for nearest-first sorting — delegates to
  /// the shared bestEffortPosition() helper, then records why into
  /// placesLocationStatusProvider so the UI can surface it instead of
  /// silently looking unsorted.
  Future<Position?> _currentPosition() async {
    final (pos, result) = await bestEffortPosition();
    ref.read(placesLocationStatusProvider.notifier).state = switch (result) {
      DeviceLocationResult.granted => PlacesLocationStatus.sorted,
      DeviceLocationResult.deniedForever => PlacesLocationStatus.deniedForever,
      DeviceLocationResult.unavailable => PlacesLocationStatus.unavailable,
    };
    return pos;
  }
}

final placesProvider = NotifierProvider<PlacesNotifier, List<Place>>(PlacesNotifier.new);

// ── categories (dynamic from server) ──────────────────────────────

class CategoriesNotifier extends Notifier<List<VendorCategory>> {
  @override
  List<VendorCategory> build() {
    Future.microtask(refresh);
    final cached = ref.read(localStoreProvider).getJsonList(StoreKeys.categories);
    if (cached != null && cached.isNotEmpty) {
      final categories = <VendorCategory>[];
      for (final raw in cached) {
        try {
          final cat = VendorCategory.fromJson((raw as Map).cast<String, dynamic>());
          final id = cat.id.toLowerCase();
          if (id != 'general' && id != 'genaral' && id != 'uncategorized') {
            categories.add(cat);
          }
        } catch (e) {
          debugPrint('CategoriesNotifier: skipping unparseable category: $e');
        }
      }
      if (categories.isNotEmpty) {
        vendorCategories = categories;
        return categories;
      }
    }
    return vendorCategories;
  }

  Future<void> refresh() async {
    try {
      final res = await ref.read(apiProvider).dio.get('/categories');
      final list = res.data as List;
      if (list.isEmpty) return;
      final categories = <VendorCategory>[];
      for (final raw in list) {
        try {
          final cat = VendorCategory.fromJson((raw as Map).cast<String, dynamic>());
          final id = cat.id.toLowerCase();
          if (id != 'general' && id != 'genaral' && id != 'uncategorized') {
            categories.add(cat);
          }
        } catch (e) {
          debugPrint('CategoriesNotifier: skipping unparseable category: $e');
        }
      }
      if (categories.isEmpty) return;
      state = categories;
      vendorCategories = categories; // update global fallback synchronously
      await ref.read(localStoreProvider).setJsonList(StoreKeys.categories, list);
    } catch (e) {
      debugPrint('categoriesProvider refresh failed: $e');
    }
  }
}

final categoriesProvider = NotifierProvider<CategoriesNotifier, List<VendorCategory>>(CategoriesNotifier.new);

// ── group ─────────────────────────────────────────────────────────

class GroupNotifier extends Notifier<Group> {
  @override
  Group build() {
    Future.microtask(_pull);
    final json = ref.read(localStoreProvider).getJson(StoreKeys.group);
    return json == null ? seed.groupSeed : Group.fromJson(json);
  }

  Future<void> _pull() async {
    if (!ref.read(authProvider)) return;
    try {
      final res = await ref.read(apiProvider).dio.get('/me/group');
      final data = res.data;
      if (data is Map<String, dynamic> && data['code'] != null) {
        state = Group.fromJson(data);
        await ref.read(localStoreProvider).setJson(StoreKeys.group, state.toJson());
        _syncKnownTravelers();
      }
    } catch (_) {}
  }

  /// Keeps tripWho()'s lookup pool (trip_common.dart) current — trip
  /// collaborators are drawn from the same real group members shown
  /// elsewhere, not the old hardcoded mock list.
  void _syncKnownTravelers() {
    seed.travelers = state.members.map(Traveler.fromGroupMember).toList();
  }

  void _persist(Group g) {
    state = g;
    ref.read(localStoreProvider).setJson(StoreKeys.group, g.toJson());
    if (ref.read(authProvider)) {
      ref.read(apiProvider).dio.put('/me/group', data: g.toJson()).ignore();
    }
  }

  void addMember(String id) {
    if (state.memberIds.contains(id)) return;
    _persist(state.copyWith(memberIds: [...state.memberIds, id]));
  }

  void removeMember(String id) {
    _persist(state.copyWith(memberIds: state.memberIds.where((m) => m != id).toList()));
  }

  void leave() => _persist(state.copyWith(memberIds: []));
}

// ── nearby travellers (live location, opt-in) ──────────────────────

/// Reports the device's position (only while the user has opted into
/// location sharing) and fetches other opted-in travellers nearby. Only
/// runs while something is actually watching nearbyTravellersProvider
/// (Riverpod tears the notifier down otherwise) — e.g. the Discover map.
class NearbyTravellersNotifier extends Notifier<List<NearbyTraveller>> {
  Timer? _timer;

  @override
  List<NearbyTraveller> build() {
    ref.onDispose(() => _timer?.cancel());
    final shareLocation = ref.watch(meProvider)?['shareLocation'] as bool? ?? false;
    _timer?.cancel();
    if (shareLocation && ref.watch(authProvider)) {
      Future.microtask(_tick);
      _timer = Timer.periodic(const Duration(minutes: 2), (_) => _tick());
    }
    return shareLocation ? state : [];
  }

  Future<void> _tick() async {
    try {
      final (pos, _) = await bestEffortPosition();
      if (pos == null) return;
      await ref
          .read(apiProvider)
          .dio
          .put('/me/location', data: {'lat': pos.latitude, 'lng': pos.longitude});
      final res = await ref.read(apiProvider).dio.get('/travellers/nearby');
      state = [
        for (final t in (res.data as List)) NearbyTraveller.fromJson((t as Map).cast<String, dynamic>())
      ];
      // Extend tripWho()'s lookup pool (trip_common.dart) with real nearby
      // travellers too, alongside group members — trip_share_sheet.dart's
      // "invite travellers" suggestions draw from the same pool.
      final knownIds = seed.travelers.map((t) => t.id).toSet();
      seed.travelers = [
        ...seed.travelers,
        for (final t in state)
          if (!knownIds.contains(t.id)) Traveler.fromNearby(t),
      ];
    } catch (_) {
      // Offline, permission denied, or not opted in server-side — leave
      // the last-known state rather than clearing it on a transient error.
    }
  }
}

final nearbyTravellersProvider =
    NotifierProvider<NearbyTravellersNotifier, List<NearbyTraveller>>(NearbyTravellersNotifier.new);

final groupProvider = NotifierProvider<GroupNotifier, Group>(GroupNotifier.new);

// ── trip (whole-document sync, debounced PUT) ─────────────────────

class TripNotifier extends Notifier<Trip> {
  Timer? _pushTimer;

  @override
  Trip build() {
    ref.onDispose(() => _pushTimer?.cancel());
    final json = ref.read(localStoreProvider).getJson(StoreKeys.trip);
    final hasLocal = json != null;
    Future.microtask(() => _pull(adopt: !hasLocal));
    return hasLocal ? Trip.fromJson(json) : tripSeed;
  }

  Future<void> _pull({required bool adopt}) async {
    if (!ref.read(authProvider)) return;
    try {
      final res = await ref.read(apiProvider).dio.get('/me/trip');
      final data = res.data;
      if (data is Map<String, dynamic> && data['days'] != null) {
        if (adopt) {
          state = Trip.fromJson(data);
          await ref.read(localStoreProvider).setJson(StoreKeys.trip, state.toJson());
        }
      } else if (!adopt) {
        _schedulePush(); // server empty, local exists — push ours up
      }
    } catch (_) {}
  }

  void _schedulePush() {
    if (!ref.read(authProvider)) return;
    _pushTimer?.cancel();
    _pushTimer = Timer(const Duration(milliseconds: 800), () {
      ref.read(apiProvider).dio.put('/me/trip', data: state.toJson()).ignore();
    });
  }

  void update(Trip trip) {
    state = trip;
    ref.read(localStoreProvider).setJson(StoreKeys.trip, trip.toJson());
    ref.read(savedTripsProvider.notifier).saveTrip(trip);
    _schedulePush();
  }

  void addStop(int dayIndex, TripStop stop) {
    final days = [...state.days];
    final stops = [...days[dayIndex].stops, stop]..sort((a, b) => a.time.compareTo(b.time));
    days[dayIndex] = days[dayIndex].copyWith(stops: stops);
    update(state.copyWith(days: days));
  }

  void removeStop(int dayIndex, String stopId) {
    final days = [...state.days];
    days[dayIndex] =
        days[dayIndex].copyWith(stops: days[dayIndex].stops.where((s) => s.id != stopId).toList());
    update(state.copyWith(days: days));
  }

  void setStopTime(int dayIndex, String stopId, int mins) {
    final days = [...state.days];
    days[dayIndex] = days[dayIndex].copyWith(
        stops: [for (final s in days[dayIndex].stops) s.id == stopId ? s.copyWith(time: mins) : s]);
    update(state.copyWith(days: days));
  }

  /// Swap a stop with its neighbour but keep the time slots in place,
  /// so the timeline stays chronological (prototype semantics).
  void moveStop(int dayIndex, int ix, int dir) {
    final stops = [...state.days[dayIndex].stops];
    final j = ix + dir;
    if (j < 0 || j >= stops.length) return;
    final a = stops[ix], b = stops[j];
    stops[ix] = b.copyWith(time: a.time);
    stops[j] = a.copyWith(time: b.time);
    final days = [...state.days];
    days[dayIndex] = days[dayIndex].copyWith(stops: stops);
    update(state.copyWith(days: days));
  }

  /// Move a stop from one index to another, possibly crossing days.
  void moveStopBetweenDays(int oldDayIx, int oldStopIx, int newDayIx, int newStopIx) {
    if (oldDayIx < 0 || oldDayIx >= state.days.length || newDayIx < 0 || newDayIx >= state.days.length) return;
    
    final oldStops = [...state.days[oldDayIx].stops];
    if (oldStopIx < 0 || oldStopIx >= oldStops.length) return;
    
    final stop = oldStops.removeAt(oldStopIx);
    
    if (oldDayIx == newDayIx) {
      if (newStopIx > oldStops.length) newStopIx = oldStops.length;
      oldStops.insert(newStopIx, stop);
      
      final days = [...state.days];
      days[oldDayIx] = days[oldDayIx].copyWith(stops: oldStops);
      update(state.copyWith(days: days));
    } else {
      final newStops = [...state.days[newDayIx].stops];
      if (newStopIx > newStops.length) newStopIx = newStops.length;
      
      int time = 9 * 60;
      if (newStops.isNotEmpty) {
        if (newStopIx == 0) {
          time = newStops.first.time - 60;
        } else if (newStopIx >= newStops.length) {
          time = newStops.last.time + 60;
        } else {
          time = (newStops[newStopIx - 1].time + newStops[newStopIx].time) ~/ 2;
        }
      }
      
      newStops.insert(newStopIx, stop.copyWith(time: time));
      
      final days = [...state.days];
      days[oldDayIx] = days[oldDayIx].copyWith(stops: oldStops);
      days[newDayIx] = days[newDayIx].copyWith(stops: newStops);
      update(state.copyWith(days: days));
    }
  }
  /// New stop from the catalog: 90 min after the last stop (clamped to
  /// 23:30), or the item's suggested time on an empty day.
  void addCatalogStop(int dayIndex, TripCatalogItem item) {
    final stops = state.days[dayIndex].stops;
    final parts = item.hint.split(':');
    final hinted = (int.tryParse(parts[0]) ?? 9) * 60 + (parts.length > 1 ? int.tryParse(parts[1]) ?? 0 : 0);
    final t = stops.isEmpty ? hinted : (stops.last.time + 90).clamp(0, 23 * 60 + 30);
    addStop(dayIndex, TripStop(id: tripNewId('s'), placeId: item.id, time: t, by: 'me'));
  }

  void addCustomStop(int dayIndex, String name) {
    final stops = state.days[dayIndex].stops;
    final t = stops.isEmpty ? 9 * 60 : (stops.last.time + 90).clamp(0, 23 * 60 + 30);
    addStop(dayIndex,
        TripStop(id: tripNewId('s'), placeId: '', time: t, by: 'me', custom: CustomStopInfo(name: name)));
  }

  void addPlaceStop(int dayIndex, Place place) {
    final stops = state.days[dayIndex].stops;
    final t = stops.isEmpty ? 9 * 60 : (stops.last.time + 90).clamp(0, 23 * 60 + 30);
    addStop(
      dayIndex,
      TripStop(
        id: tripNewId('s'),
        placeId: place.id,
        time: t,
        by: 'me',
        custom: place.id.startsWith('gplace_') ? CustomStopInfo(name: place.name) : null,
      ),
    );
  }

  void addDay() {
    update(state.copyWith(days: [...state.days, TripDay(id: tripNewId('d'), stops: const [])]));
  }

  void createTrip(String name, int days, String startDate) {
    update(Trip(
      name: name.trim().isEmpty ? 'Spotto Trip' : name.trim(),
      code: newTripCode(),
      startDate: startDate,
      collaborators: const [],
      days: [for (var i = 0; i < days; i++) TripDay(id: tripNewId('d'), stops: const [])],
    ));
  }

  void switchActiveTrip(String code) {
    final all = ref.read(savedTripsProvider);
    final target = all.firstWhere((t) => t.code == code, orElse: () => state);
    update(target);
  }

  void deleteTrip(String code) {
    ref.read(savedTripsProvider.notifier).deleteTrip(code);
    final remaining = ref.read(savedTripsProvider);
    if (remaining.isNotEmpty) {
      update(remaining.first);
    } else {
      createTrip('Spotto Trip', 3, todayIso());
    }
  }

  void updateTripDetails(String code, String newName, String newStartDate, int newDays) {
    final all = ref.read(savedTripsProvider);
    final target = all.firstWhere((t) => t.code == code, orElse: () => state);
    final trimmedName = newName.trim().isEmpty ? target.name : newName.trim();

    List<TripDay> days = [...target.days];
    if (newDays > days.length) {
      final toAdd = newDays - days.length;
      for (var i = 0; i < toAdd; i++) {
        days.add(TripDay(id: tripNewId('d'), stops: const []));
      }
    } else if (newDays < days.length && newDays > 0) {
      days = days.take(newDays).toList();
    }

    final updated = target.copyWith(
      name: trimmedName,
      startDate: newStartDate,
      days: days,
    );

    ref.read(savedTripsProvider.notifier).saveTrip(updated);
    if (state.code == code) {
      state = updated;
      ref.read(localStoreProvider).setJson(StoreKeys.trip, updated.toJson());
      _schedulePush();
    }
  }

  void addCollaborator(String id) {
    if (state.collaborators.contains(id)) return;
    update(state.copyWith(collaborators: [...state.collaborators, id]));
  }

  void removeCollaborator(String id) =>
      update(state.copyWith(collaborators: state.collaborators.where((c) => c != id).toList()));

  // ── inspiration links ──────────────────────────────────────────────

  void addInspiration(InspirationLink link) {
    if (state.inspirations.any((l) => l.url == link.url)) return; // dedupe by URL
    update(state.copyWith(inspirations: [link, ...state.inspirations]));
  }

  void removeInspiration(String id) {
    update(state.copyWith(inspirations: state.inspirations.where((l) => l.id != id).toList()));
  }

  void editInspirationNote(String id, String note) {
    update(state.copyWith(
      inspirations: [
        for (final l in state.inspirations) l.id == id ? l.copyWith(note: note) : l,
      ],
    ));
  }
}

final tripProvider = NotifierProvider<TripNotifier, Trip>(TripNotifier.new);

/// Saved trips collection — manages all user trips locally.
class SavedTripsNotifier extends Notifier<List<Trip>> {
  @override
  List<Trip> build() {
    final store = ref.read(localStoreProvider);
    final jsonList = store.getJsonList(StoreKeys.allTrips);
    if (jsonList != null && jsonList.isNotEmpty) {
      return [for (final j in jsonList) Trip.fromJson(j as Map<String, dynamic>)];
    }
    final seed = ref.read(tripProvider);
    return [seed];
  }

  void saveTrip(Trip trip) {
    final list = [...state];
    final index = list.indexWhere((t) => t.code == trip.code);
    if (index >= 0) {
      list[index] = trip;
    } else {
      list.insert(0, trip);
    }
    state = list;
    _persist();
  }

  void deleteTrip(String code) {
    state = state.where((t) => t.code != code).toList();
    _persist();
  }

  void _persist() {
    final store = ref.read(localStoreProvider);
    store.setJsonList(StoreKeys.allTrips, [for (final t in state) t.toJson()]);
  }
}

final savedTripsProvider = NotifierProvider<SavedTripsNotifier, List<Trip>>(SavedTripsNotifier.new);

// ── deal claims (server-issued single-use SPOTTO codes) ───────────

class ClaimsNotifier extends Notifier<Map<String, String>> {
  @override
  Map<String, String> build() {
    Future.microtask(_pull);
    final json = ref.read(localStoreProvider).getJson(StoreKeys.claims);
    return json?.map((k, v) => MapEntry(k, v as String)) ?? {};
  }

  Future<void> _pull() async {
    if (!ref.read(authProvider)) return;
    try {
      final res = await ref.read(apiProvider).dio.get('/me/claims');
      final map = <String, String>{
        for (final c in (res.data as List))
          (c as Map)['placeId'] as String: c['code'] as String,
      };
      if (map.isNotEmpty) {
        state = {...map, ...state};
        await ref.read(localStoreProvider).setJson(StoreKeys.claims, state);
      }
    } catch (_) {}
  }

  /// Claims a place's deal on the server → unique single-use code.
  /// Throws on network failure (callers show an error state). offerId
  /// links the claim back to the specific vendor Offer (when the deal
  /// came from one) so the vendor sees real claimed/redeemed stats for it.
  Future<String> claim(String placeId, {String? offerId, String offerLabel = '', String offerValue = ''}) async {
    final existing = state[placeId];
    if (existing != null) return existing;
    final res = await ref.read(apiProvider).dio.post('/claims', data: {
      'placeId': placeId,
      if (offerId case final id?) 'offerId': id,
      'offerLabel': offerLabel,
      'offerValue': offerValue,
    });
    final code = (res.data as Map)['code'] as String;
    state = {...state, placeId: code};
    ref.read(localStoreProvider).setJson(StoreKeys.claims, state);
    return code;
  }

  String? codeFor(String placeId) => state[placeId];
}

final claimsProvider = NotifierProvider<ClaimsNotifier, Map<String, String>>(ClaimsNotifier.new);

// ── bookings (real, auto-confirmed commitments — see server's booking.go) ─

/// A place's active bookable services — empty for a non-bookable place.
final placeServicesProvider =
    FutureProvider.family<List<BookableService>, String>((ref, placeId) async {
  final res = await ref.read(apiProvider).dio.get('/places/$placeId/services');
  return [
    for (final s in (res.data as List)) BookableService.fromJson((s as Map).cast<String, dynamic>()),
  ];
});

/// Raw availability response for one service — `query` is the caller-built
/// query string (`date=2026-08-10` for session/table, `checkIn=...&checkOut=...`
/// for stay), since the two service shapes need different params (see
/// server's handleServiceAvailability). UI reads `slots` or `remaining` out
/// of the decoded map depending on the service's type.
final serviceAvailabilityProvider = FutureProvider.family<
    Map<String, dynamic>, ({String placeId, String serviceId, String query})>((ref, args) async {
  final res = await ref
      .read(apiProvider)
      .dio
      .get('/places/${args.placeId}/services/${args.serviceId}/availability?${args.query}');
  return (res.data as Map).cast<String, dynamic>();
});

/// The traveller's own bookings — no optimistic write (unlike claims'
/// single key-flip, a booking is a genuinely new record with real fields),
/// so [create] just appends the server's authoritative response on success.
class BookingsNotifier extends Notifier<List<Booking>> {
  @override
  List<Booking> build() {
    Future.microtask(_pull);
    final cached = ref.read(localStoreProvider).getJsonList(StoreKeys.bookings);
    return cached == null
        ? []
        : [for (final b in cached) Booking.fromJson((b as Map).cast<String, dynamic>())];
  }

  Future<void> _pull() async {
    if (!ref.read(authProvider)) return;
    try {
      final res = await ref.read(apiProvider).dio.get('/me/bookings');
      state = [
        for (final b in (res.data as List)) Booking.fromJson((b as Map).cast<String, dynamic>()),
      ];
      _persist();
    } catch (_) {}
  }

  void _persist() => ref.read(localStoreProvider).setJsonList(
        StoreKeys.bookings,
        [for (final b in state) b.toJson()],
      );

  /// Creates a real booking. Throws on failure (caller shows the error via
  /// ApiClient.errorMessage) — same convention as ClaimsNotifier.claim.
  Future<Booking> create({
    required String placeId,
    required String serviceId,
    DateTime? startAt,
    DateTime? checkIn,
    DateTime? checkOut,
    required int partySize,
    String note = '',
  }) async {
    final res = await ref.read(apiProvider).dio.post('/bookings', data: {
      'placeId': placeId,
      'serviceId': serviceId,
      if (startAt != null) 'startAt': startAt.toUtc().toIso8601String(),
      if (checkIn != null) 'checkIn': checkIn.toIso8601String().split('T').first,
      if (checkOut != null) 'checkOut': checkOut.toIso8601String().split('T').first,
      'partySize': partySize,
      'note': note,
    });
    final booking = Booking.fromJson((res.data as Map).cast<String, dynamic>());
    state = [booking, ...state];
    _persist();
    return booking;
  }

  /// Optimistic status flip + rollback on failure — same technique as
  /// VibeCheckInsNotifier.rate.
  Future<void> cancel(String bookingId) async {
    final previous = state;
    state = [
      for (final b in state)
        if (b.id == bookingId) b.copyWith(status: 'cancelled') else b,
    ];
    _persist();
    try {
      await ref.read(apiProvider).dio.post('/bookings/$bookingId/cancel');
    } catch (e) {
      state = previous;
      _persist();
      rethrow;
    }
  }
}

final bookingsProvider = NotifierProvider<BookingsNotifier, List<Booking>>(BookingsNotifier.new);

// ── weather (server-owned, real via Tomorrow.io) ───────────────────

/// Best-effort device position for weather, rounded to 2dp (~1.1km) so
/// small GPS jitter doesn't spawn a fresh /weather fetch every refresh —
/// delegates to the shared bestEffortPosition() helper, falling back to
/// Weligama Bay on denied/disabled location.
final weatherLocationProvider = FutureProvider<LatLngAt>((ref) async {
  final (pos, _) = await bestEffortPosition();
  if (pos == null) return (lat: kFallbackLat, lng: kFallbackLng);
  double round(double v) => double.parse(v.toStringAsFixed(2));
  return (lat: round(pos.latitude), lng: round(pos.longitude));
});

/// Raw GET /weather for one location.
final weatherSnapshotProvider = FutureProvider.family<WeatherSnapshot, LatLngAt>((ref, at) async {
  final res = await ref.read(apiProvider).dio.get('/weather', queryParameters: {'lat': at.lat, 'lng': at.lng});
  return WeatherSnapshot.fromJson((res.data as Map).cast<String, dynamic>());
});

/// The provider screens actually watch — chains device location to the
/// weather fetch. Not autoDispose: keeps the snapshot warm for the session.
final currentWeatherProvider = FutureProvider<WeatherSnapshot>((ref) async {
  final at = await ref.watch(weatherLocationProvider.future);
  return ref.watch(weatherSnapshotProvider(at).future);
});

/// Single batched provider that fetches weather for Current Location + all Pinned Locations
/// in ONE single POST /weather/batch API request to minimize network traffic and API costs.
final batchedWeatherDeckProvider = FutureProvider<Map<String, WeatherSnapshot>>((ref) async {
  final currentAt = await ref.watch(weatherLocationProvider.future);
  final pins = ref.watch(weatherPinsProvider);

  final locations = <Map<String, dynamic>>[
    {'id': 'current', 'lat': currentAt.lat, 'lng': currentAt.lng},
    for (final pin in pins) {'id': pin.id, 'lat': pin.lat, 'lng': pin.lng},
  ];

  try {
    final res = await ref.read(apiProvider).dio.post('/weather/batch', data: {
      'locations': locations,
    });
    final rawResults = (res.data as Map)['results'] as Map? ?? {};
    final out = <String, WeatherSnapshot>{};
    for (final entry in rawResults.entries) {
      final id = entry.key.toString();
      final val = entry.value;
      if (val is Map) {
        out[id] = WeatherSnapshot.fromJson(val.cast<String, dynamic>());
      }
    }
    return out;
  } catch (_) {
    final out = <String, WeatherSnapshot>{};
    final curSnap = await ref.read(weatherSnapshotProvider(currentAt).future);
    out['current'] = curSnap;
    for (final pin in pins) {
      final at = (lat: pin.lat, lng: pin.lng);
      final snap = await ref.read(weatherSnapshotProvider(at).future);
      out[pin.id] = snap;
    }
    return out;
  }
});

/// Legacy sunny|cloudy|rainy string derived from live data — drives
/// guide_screen.dart's weather-aware greeting/suggestion chips locally.
final weatherConditionProvider = Provider<String>((ref) {
  return ref.watch(currentWeatherProvider).maybeWhen(
        data: (s) => s.now.condition,
        orElse: () => 'sunny',
      );
});

final paymentMethodsProvider = FutureProvider<List<PaymentCard>>((ref) async {
  try {
    final res = await ref.read(apiProvider).dio.get('/me/payment-methods');
    if (res.data is List) {
      return (res.data as List)
          .map((e) => PaymentCard.fromJson((e as Map).cast<String, dynamic>()))
          .toList();
    }
  } catch (_) {}
  return const [];
});

/// Locations the traveller has pinned on the Weather screen to swipe
/// between — e.g. checking Arugam Bay's surf weather while in Colombo.
/// Capped at 3 so the swipe deck stays quick to page through.
const kMaxWeatherPins = 3;

class WeatherPinsNotifier extends Notifier<List<PinnedLocation>> {
  @override
  List<PinnedLocation> build() {
    final raw = ref.read(localStoreProvider).getJsonList(StoreKeys.weatherPins) ?? const [];
    return [for (final p in raw) PinnedLocation.fromJson((p as Map).cast<String, dynamic>())];
  }

  void _persist(List<PinnedLocation> pins) {
    state = pins;
    ref.read(localStoreProvider).setJsonList(StoreKeys.weatherPins, [for (final p in pins) p.toJson()]);
  }

  /// No-ops past kMaxWeatherPins or on a duplicate id — callers should
  /// check `canAddMore`/existing ids first to give the user a clear reason
  /// rather than a silent no-op.
  void add(PinnedLocation pin) {
    if (state.length >= kMaxWeatherPins || state.any((p) => p.id == pin.id)) return;
    _persist([...state, pin]);
  }

  void remove(String id) => _persist(state.where((p) => p.id != id).toList());
}

final weatherPinsProvider = NotifierProvider<WeatherPinsNotifier, List<PinnedLocation>>(WeatherPinsNotifier.new);

// ── session-only traveller state (matches prototype useState) ─────

/// Active time-of-day filter: morning | afternoon | evening.
final timeFilterProvider = StateProvider<String>((ref) => 'afternoon');

/// One place's vibe state as submitted this session — a fresh three-
/// dimension snapshot every time (see VibeCheckInsNotifier.rate).
typedef VibeState = ({VibeAggregate vibe, VibeCheckin mine});

/// Session overrides for vibe check-ins, keyed by placeId. Starts empty —
/// the vibe widget falls back to placeDetailsProvider's `vibe`/`myVibe`
/// (from GET /places/{id}) until the traveller actually rates a place here,
/// so an entry only ever exists once they've tapped something this session.
class VibeCheckInsNotifier extends Notifier<Map<String, VibeState>> {
  @override
  Map<String, VibeState> build() => {};

  /// Submits a full check-in — pass only the dimension(s) that actually
  /// changed; the rest carry over from the current known "mine" value
  /// ([previousMine] when nothing's been submitted here yet this session,
  /// else 0), since the server always records a full three-dimension
  /// snapshot rather than editing one dimension in place.
  ///
  /// Optimistic: writes the traveller's own tap into [state] immediately,
  /// before the network round-trip, so the card confirms instantly instead
  /// of waiting on the request — [currentVibe] is the aggregate the caller
  /// already has on hand (there's no way to correctly recompute the average
  /// locally without knowing the server's true sample size, so it's kept
  /// as-is until the response reconciles it). Rolls back to whatever was
  /// there before on failure, so a failed submission doesn't leave a false
  /// confirmation on screen.
  Future<void> rate(
    String placeId, {
    int? productive,
    int? loud,
    int? crowded,
    required VibeCheckin? previousMine,
    required VibeAggregate currentVibe,
    required double lat,
    required double lng,
  }) async {
    final previousState = state[placeId];
    final current = previousState?.mine ?? previousMine;
    final optimisticMine = VibeCheckin(
      productive: productive ?? current?.productive ?? 0,
      loud: loud ?? current?.loud ?? 0,
      crowded: crowded ?? current?.crowded ?? 0,
      createdAt: DateTime.now(),
    );
    state = {...state, placeId: (vibe: currentVibe, mine: optimisticMine)};
    try {
      final res = await ref.read(apiProvider).dio.post('/places/$placeId/vibe-checkin', data: {
        'productive': optimisticMine.productive,
        'loud': optimisticMine.loud,
        'crowded': optimisticMine.crowded,
        'lat': lat,
        'lng': lng,
      });
      final resp = (res.data as Map).cast<String, dynamic>();
      state = {
        ...state,
        placeId: (
          vibe: VibeAggregate.fromJson((resp['vibe'] as Map).cast<String, dynamic>()),
          mine: VibeCheckin.fromJson((resp['myVibe'] as Map).cast<String, dynamic>()),
        ),
      };
    } catch (e) {
      if (previousState != null) {
        state = {...state, placeId: previousState};
      } else {
        state = {...state}..remove(placeId);
      }
      rethrow;
    }
  }
}

final vibeCheckInsProvider =
    NotifierProvider<VibeCheckInsNotifier, Map<String, VibeState>>(VibeCheckInsNotifier.new);

/// Artisan check-ins for the daily quest (optimistic, posted to the server).
/// Persisted locally so "already checked in" survives an app restart —
/// there's no GET to hydrate this from the server, so LocalStore is the
/// only record of it.
class CheckInsNotifier extends Notifier<Set<String>> {
  @override
  Set<String> build() {
    final saved = ref.read(localStoreProvider).getJsonList(StoreKeys.checkIns);
    return saved?.cast<String>().toSet() ?? {};
  }

  Future<void> checkIn(String placeId) async {
    if (state.contains(placeId)) return;
    state = {...state, placeId};
    ref.read(localStoreProvider).setJsonList(StoreKeys.checkIns, state.toList());
    if (ref.read(authProvider)) {
      try {
        await ref.read(apiProvider).dio.post('/checkins', data: {'placeId': placeId});
        // Refresh the daily quest progress
        ref.invalidate(dailyQuestProvider);
      } catch (_) {
        // Optimistic, ignore errors for MVP
      }
    }
  }
}

final checkInsProvider = NotifierProvider<CheckInsNotifier, Set<String>>(CheckInsNotifier.new);

final dailyQuestProvider = FutureProvider<DailyQuest>((ref) async {
  if (!ref.watch(authProvider)) return const DailyQuest(goal: 2, progress: 0, rewardDescription: 'Free Ceylon Tea', isComplete: false);
  try {
    final res = await ref.read(apiProvider).dio.get('/me/quest');
    return DailyQuest.fromJson((res.data as Map).cast<String, dynamic>());
  } catch (e) {
    return const DailyQuest(goal: 2, progress: 0, rewardDescription: 'Free Ceylon Tea', isComplete: false);
  }
});

/// Bookmarked / favorited places for the traveller — local-only (no server
/// endpoint yet), persisted so bookmarks survive an app restart.
class FavoritesNotifier extends Notifier<Set<String>> {
  @override
  Set<String> build() {
    final saved = ref.read(localStoreProvider).getJsonList(StoreKeys.favorites);
    return saved?.cast<String>().toSet() ?? {};
  }

  void toggle(String placeId) {
    if (state.contains(placeId)) {
      state = state.where((id) => id != placeId).toSet();
    } else {
      state = {...state, placeId};
    }
    ref.read(localStoreProvider).setJsonList(StoreKeys.favorites, state.toList());
  }
}

final favoritesProvider = NotifierProvider<FavoritesNotifier, Set<String>>(FavoritesNotifier.new);

/// Group mode: show group members on the Discover map (session-only).
final groupModeProvider = StateProvider<bool>((ref) => false);

// ── reviews (real traveller ratings; 1-2 stars flag the vendor) ───

/// How long a resolved reviews/photos fetch is served from cache before a
/// re-watch (e.g. reopening the same place's drawer) triggers a background
/// refetch — reopening within the window is instant, and a write (review
/// submit, photo upload) still calls `ref.invalidate(...)` for an immediate
/// refresh regardless of this timer.
const _placeDetailCacheTtl = Duration(minutes: 3);

/// Reviews for a single place, newest first. Call
/// `ref.invalidate(placeReviewsProvider(placeId))` after submitting to refresh.
final placeReviewsProvider =
    FutureProvider.family<List<Review>, String>((ref, placeId) async {
  final timer = Timer(_placeDetailCacheTtl, ref.invalidateSelf);
  ref.onDispose(timer.cancel);
  final res = await ref.read(apiProvider).dio.get('/places/$placeId/reviews');
  return [for (final r in (res.data as List)) Review.fromJson((r as Map).cast<String, dynamic>())];
});

/// Traveller-contributed photos for a place (media role "contrib").
/// Invalidate after uploading/deleting to refresh the place detail's photo
/// grid.
final placePhotosProvider =
    FutureProvider.family<List<PlacePhoto>, String>((ref, placeId) async {
  final timer = Timer(_placeDetailCacheTtl, ref.invalidateSelf);
  ref.onDispose(timer.cancel);
  final res = await ref.read(apiProvider).dio.get('/places/$placeId/photos');
  return [
    for (final m in (res.data as List))
      PlacePhoto.fromJson((m as Map).cast<String, dynamic>()),
  ];
});

/// Fetches a single place fresh from `GET /places/{id}` — unlike the bulk
/// `placesProvider` list, this is what the server enriches with live,
/// cached Google Places details (phone/website/wheelchairAccessible/
/// priceLevel) for imported spots. The drawer merges the result on top of
/// the list-derived Place it already has (see business_drawer.dart), so a
/// slow/failed fetch never blocks the sheet — it just keeps showing what it
/// already knew.
final placeDetailsProvider = FutureProvider.family<Place, String>((ref, placeId) async {
  final timer = Timer(_placeDetailCacheTtl, ref.invalidateSelf);
  ref.onDispose(timer.cancel);
  final res = await ref.read(apiProvider).dio.get('/places/$placeId');
  return Place.fromJson((res.data as Map).cast<String, dynamic>());
});

/// Submits (or edits) the current traveller's review for a place, then
/// refreshes [placeReviewsProvider] for that place.
Future<String> submitReview(WidgetRef ref, String placeId, int rating, String text) async {
  final res = await ref.read(apiProvider).dio.post('/places/$placeId/reviews', data: {
    'rating': rating,
    'text': text,
  });
  ref.invalidate(placeReviewsProvider(placeId));
  return (res.data as Map)['id'] as String? ?? '';
}
