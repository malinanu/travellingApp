// State layer: Riverpod notifiers that load seed data, overlay persisted
// state from LocalStore, and write back on mutation. Mirrors the prototype's
// useState + localStorage pairs.

import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/storage/local_store.dart';
import '../mock/places.dart' as seed;
import '../mock/trip_catalog.dart';
import '../mock/vendor_seed.dart';
import '../models/models.dart';

/// Overridden in main() with the initialized instance.
final localStoreProvider = Provider<LocalStore>((ref) => throw UnimplementedError());

// ── role ──────────────────────────────────────────────────────────

class RoleNotifier extends Notifier<String?> {
  @override
  String? build() => ref.read(localStoreProvider).getString(StoreKeys.role);

  void set(String role) {
    state = role;
    ref.read(localStoreProvider).setString(StoreKeys.role, role);
  }
}

final roleProvider = NotifierProvider<RoleNotifier, String?>(RoleNotifier.new);

// ── traveller auth + prefs ────────────────────────────────────────

class AuthNotifier extends Notifier<bool> {
  @override
  bool build() => ref.read(localStoreProvider).getBool(StoreKeys.auth);

  void signIn() {
    state = true;
    ref.read(localStoreProvider).setBool(StoreKeys.auth, true);
  }

  void signOut() {
    state = false;
    final store = ref.read(localStoreProvider);
    store.setBool(StoreKeys.auth, false);
    store.remove(StoreKeys.prefs);
    ref.invalidate(prefsProvider);
  }
}

final authProvider = NotifierProvider<AuthNotifier, bool>(AuthNotifier.new);

class PrefsNotifier extends Notifier<UserPrefs?> {
  @override
  UserPrefs? build() {
    final json = ref.read(localStoreProvider).getJson(StoreKeys.prefs);
    return json == null ? null : UserPrefs.fromJson(json);
  }

  void save(UserPrefs prefs) {
    state = prefs;
    ref.read(localStoreProvider).setJson(StoreKeys.prefs, prefs.toJson());
  }
}

final prefsProvider = NotifierProvider<PrefsNotifier, UserPrefs?>(PrefsNotifier.new);

// ── group ─────────────────────────────────────────────────────────

class GroupNotifier extends Notifier<Group> {
  @override
  Group build() {
    final json = ref.read(localStoreProvider).getJson(StoreKeys.group);
    return json == null ? seed.groupSeed : Group.fromJson(json);
  }

  void _persist(Group g) {
    state = g;
    ref.read(localStoreProvider).setJson(StoreKeys.group, g.toJson());
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

final groupProvider = NotifierProvider<GroupNotifier, Group>(GroupNotifier.new);

// ── trip ──────────────────────────────────────────────────────────

class TripNotifier extends Notifier<Trip> {
  @override
  Trip build() {
    final json = ref.read(localStoreProvider).getJson(StoreKeys.trip);
    return json == null ? tripSeed : Trip.fromJson(json);
  }

  void update(Trip trip) {
    state = trip;
    ref.read(localStoreProvider).setJson(StoreKeys.trip, trip.toJson());
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

// ── deal claims (cross-app: traveller claims, vendor validates) ───

class ClaimsNotifier extends Notifier<Map<String, String>> {
  @override
  Map<String, String> build() {
    final json = ref.read(localStoreProvider).getJson(StoreKeys.claims);
    return json?.map((k, v) => MapEntry(k, v as String)) ?? {};
  }

  /// Claim a place's deal → `SPOTTO-` + first 4 of id, uppercased. Persisted.
  String claim(String placeId) {
    final existing = state[placeId];
    if (existing != null) return existing;
    final code = 'SPOTTO-${placeId.substring(0, placeId.length < 4 ? placeId.length : 4).toUpperCase()}';
    state = {...state, placeId: code};
    ref.read(localStoreProvider).setJson(StoreKeys.claims, state);
    return code;
  }

  String? codeFor(String placeId) => state[placeId];

  bool isClaimedCode(String code) => state.containsValue(code.toUpperCase());
}

final claimsProvider = NotifierProvider<ClaimsNotifier, Map<String, String>>(ClaimsNotifier.new);

// ── session-only traveller state (matches prototype useState) ─────

/// Current weather: sunny | cloudy | rainy.
final weatherProvider = StateProvider<String>((ref) => 'sunny');

/// Active time-of-day filter: morning | afternoon | evening.
final timeFilterProvider = StateProvider<String>((ref) => 'afternoon');

/// Vibe votes per place (seeded, session-only like the prototype).
class VibeVotesNotifier extends Notifier<Map<String, Map<String, int>>> {
  @override
  Map<String, Map<String, int>> build() =>
      seed.vibeSeed.map((k, v) => MapEntry(k, Map<String, int>.from(v)));

  void vote(String placeId, String vibeId) {
    final place = Map<String, int>.from(state[placeId] ?? {});
    place[vibeId] = (place[vibeId] ?? 0) + 1;
    state = {...state, placeId: place};
  }
}

final vibeVotesProvider =
    NotifierProvider<VibeVotesNotifier, Map<String, Map<String, int>>>(VibeVotesNotifier.new);

/// Artisan check-ins for the daily quest (session-only).
final checkInsProvider = StateProvider<Set<String>>((ref) => {});

/// Group mode: show group members on the Discover map (session-only).
final groupModeProvider = StateProvider<bool>((ref) => false);

// ── vendor ────────────────────────────────────────────────────────

class VendorAuthNotifier extends Notifier<VendorBusiness?> {
  @override
  VendorBusiness? build() {
    final json = ref.read(localStoreProvider).getJson(StoreKeys.vendorAuth);
    return json == null ? null : VendorBusiness.fromJson(json);
  }

  void signIn([VendorBusiness? business]) {
    final b = business ?? vendorSeed;
    state = b;
    ref.read(localStoreProvider).setJson(StoreKeys.vendorAuth, b.toJson());
  }

  void updateBusiness(VendorBusiness b) {
    state = b;
    ref.read(localStoreProvider).setJson(StoreKeys.vendorAuth, b.toJson());
  }

  void signOut() {
    state = null;
    ref.read(localStoreProvider).remove(StoreKeys.vendorAuth);
  }
}

final vendorAuthProvider =
    NotifierProvider<VendorAuthNotifier, VendorBusiness?>(VendorAuthNotifier.new);

class OffersNotifier extends Notifier<List<Offer>> {
  @override
  List<Offer> build() => offersSeed;

  void add(Offer offer) => state = [offer, ...state];
  void setStatus(String id, String status) =>
      state = [for (final o in state) o.id == id ? o.copyWith(status: status) : o];
  void replace(Offer offer) => state = [for (final o in state) o.id == offer.id ? offer : o];
  void remove(String id) => state = state.where((o) => o.id != id).toList();
}

final offersProvider = NotifierProvider<OffersNotifier, List<Offer>>(OffersNotifier.new);

/// Vendor's menu (session-only, like the prototype's useState).
class VendorMenuNotifier extends Notifier<List<MenuSection>> {
  @override
  List<MenuSection> build() => vendorMenuSeed;

  void upsertItem(String sectionId, MenuItem item) {
    state = [
      for (final s in state)
        s.id != sectionId
            ? MenuSection(id: s.id, name: s.name, items: s.items.where((i) => i.id != item.id).toList())
            : MenuSection(
                id: s.id,
                name: s.name,
                items: s.items.any((i) => i.id == item.id)
                    ? [for (final i in s.items) i.id == item.id ? item : i]
                    : [...s.items, item],
              )
    ];
  }

  void removeItem(String itemId) {
    state = [
      for (final s in state)
        MenuSection(id: s.id, name: s.name, items: s.items.where((i) => i.id != itemId).toList())
    ];
  }
}

final vendorMenuProvider =
    NotifierProvider<VendorMenuNotifier, List<MenuSection>>(VendorMenuNotifier.new);

/// Vendor's public profile settings (session-only).
class VendorProfileNotifier extends Notifier<VendorProfile> {
  @override
  VendorProfile build() => vendorProfileSeed;

  void set(VendorProfile p) => state = p;
}

final vendorProfileProvider =
    NotifierProvider<VendorProfileNotifier, VendorProfile>(VendorProfileNotifier.new);

/// Bell notifications (session-only).
class VendorNotifsNotifier extends Notifier<List<VendorNotif>> {
  @override
  List<VendorNotif> build() => vendorNotifsSeed;

  void markOne(String id) =>
      state = [for (final n in state) n.id == id ? n.copyWith(unread: false) : n];
  void markAll() => state = [for (final n in state) n.copyWith(unread: false)];
}

final vendorNotifsProvider =
    NotifierProvider<VendorNotifsNotifier, List<VendorNotif>>(VendorNotifsNotifier.new);

/// Result of validating a code at the vendor counter.
class ValidationResult {
  final bool valid;
  final String code;
  final String? name;
  final String? detail; // country/style line
  final String? offer;
  final String? value;
  const ValidationResult({
    required this.valid,
    required this.code,
    this.name,
    this.detail,
    this.offer,
    this.value,
  });
}

/// Widget-side entry point: `ref.read(codeValidatorProvider)(code)`.
final codeValidatorProvider =
    Provider<ValidationResult Function(String)>((ref) => (raw) => validateCode(ref, raw));

/// Validates seeded demo codes AND codes the traveller genuinely claimed
/// in this install (the cross-app flow).
ValidationResult validateCode(Ref ref, String raw) {
  final code = raw.trim().toUpperCase();

  final known = validCodes[code];
  if (known != null) {
    return ValidationResult(
      valid: true,
      code: code,
      name: known.name,
      detail: '${known.country} · ${known.style}',
      offer: known.offer,
      value: known.value,
    );
  }

  final claims = ref.read(claimsProvider);
  final placeId = claims.entries.where((e) => e.value == code).map((e) => e.key).firstOrNull;
  if (placeId != null) {
    final place = seed.places.where((p) => p.id == placeId).firstOrNull;
    final style = ref.read(prefsProvider)?.style;
    final styleLabel =
        seed.travelStyles.where((s) => s.id == style).map((s) => s.label).firstOrNull;
    return ValidationResult(
      valid: true,
      code: code,
      name: 'You (this device)',
      detail: styleLabel ?? 'Traveller',
      offer: place?.deal.label ?? 'Claimed deal',
      value: place?.deal.tag,
    );
  }

  final legacy = validCodesLegacy[code];
  if (legacy != null) {
    return ValidationResult(
      valid: true,
      code: code,
      name: legacy.name,
      detail: '${legacy.country} · ${legacy.style}',
      offer: legacy.offer,
      value: legacy.value,
    );
  }

  return ValidationResult(valid: false, code: code);
}
