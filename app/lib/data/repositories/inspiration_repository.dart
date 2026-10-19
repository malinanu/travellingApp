// Inspiration Board state: the traveller's saved reels (server-side, see
// GET/POST /me/inspirations) and their "save for later" places
// (/me/saved-places). Both keep a local copy so the board opens instantly
// and works offline; the server copy wins on every pull.

import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/api/api_client.dart';
import '../../core/storage/local_store.dart';
import '../models/inspiration.dart';
import 'repositories.dart';

/// How often the board re-checks links the server is still watching.
const boardPollInterval = Duration(seconds: 3);

class BoardLinksNotifier extends Notifier<List<BoardLink>> {
  Timer? _poll;

  @override
  List<BoardLink> build() {
    ref.onDispose(() => _poll?.cancel());
    Future.microtask(() async {
      await _migrateTripLinks();
      await refresh();
    });
    final cached = ref.read(localStoreProvider).getJsonList(StoreKeys.boardLinks);
    return cached == null
        ? const []
        : [for (final l in cached) BoardLink.fromJson((l as Map).cast<String, dynamic>())];
  }

  Future<void> refresh() async {
    if (!ref.read(authProvider)) return;
    try {
      final res = await ref.read(apiProvider).dio.get('/me/inspirations');
      if (!ref.mounted) return;
      final list = (res.data as Map)['links'] as List? ?? const [];
      _set([for (final l in list) BoardLink.fromJson((l as Map).cast<String, dynamic>())]);
    } catch (e) {
      debugPrint('board links refresh failed: ${ApiClient.errorMessage(e)}');
    }
  }

  /// Saves a link and returns it (already processed when someone saved the
  /// same reel before). Throws with a readable message on failure.
  Future<BoardLink> add(String url, {String note = ''}) async {
    final res = await ref.read(apiProvider).dio.post('/me/inspirations', data: {'url': url, 'note': note});
    final link = BoardLink.fromJson((res.data as Map).cast<String, dynamic>());
    if (ref.mounted) _set([link, ...state.where((l) => l.id != link.id)]);
    return link;
  }

  Future<void> remove(String id) async {
    final before = state;
    _set(state.where((l) => l.id != id).toList());
    try {
      await ref.read(apiProvider).dio.delete('/me/inspirations/$id');
    } catch (e) {
      if (ref.mounted) _set(before);
      rethrow;
    }
  }

  Future<void> retry(String id) async {
    final res = await ref.read(apiProvider).dio.post('/me/inspirations/$id/retry');
    if (!ref.mounted) return;
    _replace(BoardLink.fromJson((res.data as Map).cast<String, dynamic>()));
  }

  Future<BoardLink?> fetch(String id) async {
    try {
      final res = await ref.read(apiProvider).dio.get('/me/inspirations/$id');
      if (!ref.mounted) return null;
      final link = BoardLink.fromJson((res.data as Map).cast<String, dynamic>());
      _replace(link);
      return link;
    } catch (e) {
      debugPrint('board link $id fetch failed: ${ApiClient.errorMessage(e)}');
      return null;
    }
  }

  void _replace(BoardLink link) => _set([for (final l in state) l.id == link.id ? link : l]);

  void _set(List<BoardLink> links) {
    state = links;
    ref.read(localStoreProvider).setJsonList(StoreKeys.boardLinks, [for (final l in links) _toJson(l)]);
    _schedulePoll();
  }

  /// Polls while any link is still being watched; stops once all are done.
  void _schedulePoll() {
    _poll?.cancel();
    if (!state.any((l) => l.working)) return;
    _poll = Timer(boardPollInterval, () {
      if (ref.mounted) refresh();
    });
  }

  /// Links saved by older app versions lived only on the phone, inside the
  /// trip. Upload them once, then drop them from the trip.
  Future<void> _migrateTripLinks() async {
    final store = ref.read(localStoreProvider);
    if (!ref.read(authProvider) || store.getBool(StoreKeys.boardMigrated)) return;
    final old = ref.read(tripProvider).inspirations;
    for (final l in old) {
      try {
        await ref.read(apiProvider).dio.post('/me/inspirations', data: {'url': l.url, 'note': l.note});
        if (!ref.mounted) return;
      } catch (e) {
        // Unsupported or failing links (e.g. a plain website) stay in the
        // trip rather than disappearing.
        debugPrint('board migration skipped ${l.url}: ${ApiClient.errorMessage(e)}');
        continue;
      }
      ref.read(tripProvider.notifier).removeInspiration(l.id);
    }
    store.setBool(StoreKeys.boardMigrated, true);
  }
}

Map<String, dynamic> _toJson(BoardLink l) => {
      'id': l.id,
      'url': l.url,
      'note': l.note,
      'createdAt': l.createdAt.toIso8601String(),
      'platform': l.platform,
      'status': l.status,
      'error': l.error,
      'caption': l.caption,
      'author': l.author,
      'thumbnailUrl': l.thumbnailUrl,
      'places': [
        for (final p in l.places)
          {
            'name': p.name,
            'kind': p.kind,
            'area': p.area,
            'city': p.city,
            'country': p.country,
            'evidence': p.evidence,
            if (p.atSec != null) 'atSec': p.atSec,
            'confidence': p.confidence,
            if (p.geoPlace != null) 'geoPlace': p.geoPlace!.toJson(),
            if (p.placeId != null) 'placeId': p.placeId,
          },
      ],
    };

final boardLinksProvider = NotifierProvider<BoardLinksNotifier, List<BoardLink>>(BoardLinksNotifier.new);

class SavedPlacesNotifier extends Notifier<List<SavedPlace>> {
  @override
  List<SavedPlace> build() {
    Future.microtask(refresh);
    final cached = ref.read(localStoreProvider).getJsonList(StoreKeys.savedPlaces);
    return cached == null
        ? const []
        : [for (final p in cached) SavedPlace.fromJson((p as Map).cast<String, dynamic>())];
  }

  Future<void> refresh() async {
    if (!ref.read(authProvider)) return;
    try {
      final res = await ref.read(apiProvider).dio.get('/me/saved-places');
      if (!ref.mounted) return;
      final list = (res.data as Map)['places'] as List? ?? const [];
      _set([for (final p in list) SavedPlace.fromJson((p as Map).cast<String, dynamic>())]);
      if (ref.mounted) await _uploadLocalFavorites();
    } catch (e) {
      debugPrint('saved places refresh failed: ${ApiClient.errorMessage(e)}');
    }
  }

  SavedPlace? forGeoPlace(String geoPlaceId) => state.where((p) => p.geoPlaceId == geoPlaceId).firstOrNull;
  SavedPlace? forCatalogPlace(String placeId) => state.where((p) => p.placeId == placeId).firstOrNull;

  /// Saves a Google place from a reel. Returns the saved entry.
  Future<SavedPlace> saveGeoPlace(String geoPlaceId, {String? sourceLinkId}) =>
      _save({'geoPlaceId': geoPlaceId, 'sourceLinkId': ?sourceLinkId});

  Future<SavedPlace> saveCatalogPlace(String placeId, {String? sourceLinkId}) =>
      _save({'placeId': placeId, 'sourceLinkId': ?sourceLinkId});

  Future<SavedPlace> _save(Map<String, dynamic> body) async {
    final res = await ref.read(apiProvider).dio.post('/me/saved-places', data: body);
    final saved = SavedPlace.fromJson((res.data as Map).cast<String, dynamic>());
    if (ref.mounted) _set([saved, ...state.where((p) => p.id != saved.id)]);
    return saved;
  }

  Future<void> remove(String id) async {
    final before = state;
    _set(state.where((p) => p.id != id).toList());
    try {
      await ref.read(apiProvider).dio.delete('/me/saved-places/$id');
    } catch (e) {
      if (ref.mounted) _set(before);
      rethrow;
    }
  }

  void _set(List<SavedPlace> places) {
    state = places;
    ref.read(localStoreProvider).setJsonList(StoreKeys.savedPlaces, [for (final p in places) p.toJson()]);
    // Keep catalog bookmarks (map hearts, Favorites filter) in step.
    ref.read(favoritesProvider.notifier).syncFromServer({for (final p in places) ?p.placeId});
  }

  /// Bookmarks made before saved places lived on the server: upload once.
  Future<void> _uploadLocalFavorites() async {
    final store = ref.read(localStoreProvider);
    if (store.getBool(StoreKeys.favoritesUploaded)) return;
    final local = (store.getJsonList(StoreKeys.favorites) ?? const []).cast<String>();
    for (final id in local) {
      if (forCatalogPlace(id) != null) continue;
      try {
        await saveCatalogPlace(id);
      } catch (e) {
        debugPrint('favorite upload skipped $id: ${ApiClient.errorMessage(e)}');
      }
    }
    store.setBool(StoreKeys.favoritesUploaded, true);
  }
}

final savedPlacesProvider = NotifierProvider<SavedPlacesNotifier, List<SavedPlace>>(SavedPlacesNotifier.new);
