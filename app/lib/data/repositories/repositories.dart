// State layer: Riverpod notifiers backed by the Spotto Go API, with the last
// good server response cached in LocalStore as an offline snapshot. Mutation
// method signatures are unchanged from the mock era so screens barely change.

import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:geolocator/geolocator.dart';

import '../../core/api/api_client.dart';
import '../../core/storage/local_store.dart';
import '../mock/places.dart' as seed;
import '../mock/trip_catalog.dart';
import '../models/models.dart';

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

  Future<void> _adoptSession(Map<String, dynamic> data) async {
    final store = ref.read(localStoreProvider);
    await store.setToken(data['token'] as String);
    await store.setBool(StoreKeys.auth, true);

    // Adopt server-side prefs so a returning user skips onboarding.
    final user = data['user'] as Map<String, dynamic>? ?? const {};
    final style = user['travelStyle'] as String? ?? '';
    if (style.isNotEmpty) {
      ref.read(prefsProvider.notifier).saveLocal(UserPrefs(
            style: style,
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
      state = (res.data as Map).cast<String, dynamic>();
      seed.me = Traveler.fromMe(state);
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
        .put('/me', data: {'travelStyle': prefs.style, 'interests': prefs.interests})
        .ignore();
  }
}

final prefsProvider = NotifierProvider<PrefsNotifier, UserPrefs?>(PrefsNotifier.new);

// ── places (server-owned, seed as offline fallback) ───────────────

class PlacesNotifier extends Notifier<List<Place>> {
  @override
  List<Place> build() {
    Future.microtask(refresh);
    final cached = ref.read(localStoreProvider).getJsonList(StoreKeys.places);
    if (cached != null && cached.isNotEmpty) {
      try {
        final places = [for (final p in cached) Place.fromJson((p as Map).cast<String, dynamic>())];
        _syncTripCatalog(places);
        return places;
      } catch (_) {}
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
      final res = await ref.read(apiProvider).dio.get('/places');
      final list = res.data as List;
      if (list.isEmpty) return;
      final places = [for (final p in list) Place.fromJson((p as Map).cast<String, dynamic>())];
      state = places;
      _syncTripCatalog(places);
      await ref.read(localStoreProvider).setJsonList(StoreKeys.places, list);
    } catch (e) {
      // Offline or unauthenticated — keep the cached/seed state, but log so
      // a real failure (bad API_URL, server error) isn't indistinguishable
      // from "not logged in yet".
      debugPrint('placesProvider refresh failed: $e');
    }
  }
}

final placesProvider = NotifierProvider<PlacesNotifier, List<Place>>(PlacesNotifier.new);

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
      var permission = await Geolocator.checkPermission();
      if (permission == LocationPermission.denied) {
        permission = await Geolocator.requestPermission();
      }
      if (permission == LocationPermission.denied || permission == LocationPermission.deniedForever) {
        return;
      }
      if (!await Geolocator.isLocationServiceEnabled()) return;
      final pos = await Geolocator.getCurrentPosition(
        locationSettings: const LocationSettings(accuracy: LocationAccuracy.medium),
      );
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

  void addDay() {
    if (state.days.length >= 4) return;
    update(state.copyWith(days: [...state.days, TripDay(id: tripNewId('d'), stops: const [])]));
  }

  void createTrip(String name, int days, String startDate) {
    update(Trip(
      name: name.trim().isEmpty ? 'My Weligama Trip' : name.trim(),
      code: newTripCode(),
      startDate: startDate,
      collaborators: const [],
      days: [for (var i = 0; i < days; i++) TripDay(id: tripNewId('d'), stops: const [])],
    ));
  }

  void addCollaborator(String id) {
    if (state.collaborators.contains(id)) return;
    update(state.copyWith(collaborators: [...state.collaborators, id]));
  }

  void removeCollaborator(String id) =>
      update(state.copyWith(collaborators: state.collaborators.where((c) => c != id).toList()));
}

final tripProvider = NotifierProvider<TripNotifier, Trip>(TripNotifier.new);

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
      if (offerId != null) 'offerId': offerId,
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

// ── session-only traveller state (matches prototype useState) ─────

/// Current weather: sunny | cloudy | rainy.
final weatherProvider = StateProvider<String>((ref) => 'sunny');

/// Active time-of-day filter: morning | afternoon | evening.
final timeFilterProvider = StateProvider<String>((ref) => 'afternoon');

/// Vibe votes per place. Initialized from each place's real `vibes` counts
/// (returned by the server with /places); a vote's response carries the
/// server's updated counts, which are what actually get stored — an
/// optimistic local bump is only used as an offline fallback.
class VibeVotesNotifier extends Notifier<Map<String, Map<String, int>>> {
  @override
  Map<String, Map<String, int>> build() => {
        for (final p in ref.watch(placesProvider)) p.id: Map<String, int>.from(p.vibes),
      };

  Future<void> vote(String placeId, String vibeId) async {
    if (!ref.read(authProvider)) return;
    final optimistic = Map<String, int>.from(state[placeId] ?? {});
    optimistic[vibeId] = (optimistic[vibeId] ?? 0) + 1;
    state = {...state, placeId: optimistic};
    try {
      final res = await ref.read(apiProvider).dio.post('/places/$placeId/vibes/$vibeId');
      final counts = (res.data['vibes'] as Map).map(
        (k, v) => MapEntry(k as String, (v as num).toInt()),
      );
      state = {...state, placeId: counts};
    } catch (_) {
      // Offline — keep the optimistic bump.
    }
  }
}

final vibeVotesProvider =
    NotifierProvider<VibeVotesNotifier, Map<String, Map<String, int>>>(VibeVotesNotifier.new);

/// Artisan check-ins for the daily quest (optimistic, posted to the server).
class CheckInsNotifier extends Notifier<Set<String>> {
  @override
  Set<String> build() => {};

  void checkIn(String placeId) {
    if (state.contains(placeId)) return;
    state = {...state, placeId};
    if (ref.read(authProvider)) {
      ref.read(apiProvider).dio.post('/checkins', data: {'placeId': placeId}).ignore();
    }
  }
}

final checkInsProvider = NotifierProvider<CheckInsNotifier, Set<String>>(CheckInsNotifier.new);

/// Group mode: show group members on the Discover map (session-only).
final groupModeProvider = StateProvider<bool>((ref) => false);

// ── reviews (real traveller ratings; 1-2 stars flag the vendor) ───

/// Reviews for a single place, newest first. Call
/// `ref.invalidate(placeReviewsProvider(placeId))` after submitting to refresh.
final placeReviewsProvider =
    FutureProvider.family<List<Review>, String>((ref, placeId) async {
  final res = await ref.read(apiProvider).dio.get('/places/$placeId/reviews');
  return [for (final r in (res.data as List)) Review.fromJson((r as Map).cast<String, dynamic>())];
});

/// Traveller-contributed photo URLs for a place (media role "contrib").
/// Invalidate after uploading to refresh the place detail's photo strip.
final placePhotosProvider =
    FutureProvider.family<List<String>, String>((ref, placeId) async {
  final res = await ref.read(apiProvider).dio.get('/places/$placeId/photos');
  return [for (final m in (res.data as List)) (m as Map)['url'] as String];
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
