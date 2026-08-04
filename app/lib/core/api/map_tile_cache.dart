// Map tile offline caching service for Spotto Android.
// Pre-fetches OSM tiles for the Weligama/southern Sri Lanka region so the
// Discover map remains usable without a network connection.
//
// Uses flutter_cache_manager for disk-based HTTP caching.
// Only OSM Nominatim tiles are cached here (not proprietary Google Maps SDK
// tiles which cannot legally be cached outside the SDK).

import 'dart:io';
import 'dart:math' as math;

import 'package:flutter/foundation.dart';
import 'package:flutter_cache_manager/flutter_cache_manager.dart';
import 'package:path_provider/path_provider.dart';

// ── Bounding boxes ────────────────────────────────────────────────────────

/// Outer bounds — full southern Sri Lanka coast.
const _kSouthLat = 5.85;
const _kNorthLat = 6.10;
const _kWestLng = 80.00;
const _kEastLng = 81.00;

/// Inner bounds — street-level coverage around Weligama Bay only.
const _kInnerSouthLat = 5.93;
const _kInnerNorthLat = 6.01;
const _kInnerWestLng = 80.39;
const _kInnerEastLng = 80.47;

/// Zoom levels for the outer (regional) bounding box.
const _kOuterZooms = [10, 11, 12];

/// Zoom levels for the inner (street-detail) bounding box.
const _kInnerZooms = [13, 14, 15];

// ── Custom CacheManager ──────────────────────────────────────────────────

/// Singleton CacheManager for map tiles — 30-day staleness, up to 3 000 tiles.
class MapTileCacheManager {
  static const _key = 'spotto_map_tiles';

  static final MapTileCacheManager _instance = MapTileCacheManager._();
  factory MapTileCacheManager() => _instance;
  MapTileCacheManager._();

  /// The underlying flutter_cache_manager instance.
  final CacheManager _manager = CacheManager(
    Config(
      _key,
      stalePeriod: const Duration(days: 30),
      maxNrOfCacheObjects: 3000,
    ),
  );

  CacheManager get manager => _manager;
}

// ── Tile coordinate math ──────────────────────────────────────────────────

int _lon2tile(double lon, int zoom) =>
    ((lon + 180.0) / 360.0 * math.pow(2, zoom)).floor();

int _lat2tile(double lat, int zoom) {
  final latRad = lat * math.pi / 180.0;
  return ((1.0 - math.log(math.tan(latRad) + 1.0 / math.cos(latRad)) / math.pi) /
          2.0 *
          math.pow(2, zoom))
      .floor();
}

Iterable<(int, int, int)> _tilesForBounds({
  required int zoom,
  required double southLat,
  required double northLat,
  required double westLng,
  required double eastLng,
}) sync* {
  final xMin = _lon2tile(westLng, zoom);
  final xMax = _lon2tile(eastLng, zoom);
  final yMin = _lat2tile(northLat, zoom); // north = smaller y
  final yMax = _lat2tile(southLat, zoom);
  for (var x = xMin; x <= xMax; x++) {
    for (var y = yMin; y <= yMax; y++) {
      yield (zoom, x, y);
    }
  }
}

/// OpenStreetMap tile URL — free, no API key required.
String _osmTileUrl(int zoom, int x, int y) =>
    'https://tile.openstreetmap.org/$zoom/$x/$y.png';

// ── Public API ─────────────────────────────────────────────────────────────

/// Summary of a completed pre-fetch pass.
class TileCacheResult {
  final int downloaded;
  final int skipped; // already in cache
  final int failed;
  const TileCacheResult({
    required this.downloaded,
    required this.skipped,
    required this.failed,
  });

  bool get hasErrors => failed > 0;

  @override
  String toString() =>
      'TileCacheResult(downloaded: $downloaded, skipped: $skipped, failed: $failed)';
}

/// Progress callback fired after each tile is processed.
typedef TileCacheProgress = void Function(int done, int total);

/// Pre-fetches all tiles for the configured bounding boxes.
/// Skips tiles already in cache.  Downloads up to [concurrency] tiles in
/// parallel, with a 300 ms gap between batches to respect OSM policy.
Future<TileCacheResult> prefetchMapTiles({
  TileCacheProgress? onProgress,
  @visibleForTesting int concurrency = 6,
}) async {
  // Build the full tile list.
  final tiles = <(int, int, int)>[];
  for (final zoom in _kOuterZooms) {
    tiles.addAll(_tilesForBounds(
      zoom: zoom,
      southLat: _kSouthLat,
      northLat: _kNorthLat,
      westLng: _kWestLng,
      eastLng: _kEastLng,
    ));
  }
  for (final zoom in _kInnerZooms) {
    tiles.addAll(_tilesForBounds(
      zoom: zoom,
      southLat: _kInnerSouthLat,
      northLat: _kInnerNorthLat,
      westLng: _kInnerWestLng,
      eastLng: _kInnerEastLng,
    ));
  }

  int downloaded = 0, skipped = 0, failed = 0, done = 0;
  final total = tiles.length;
  final manager = MapTileCacheManager().manager;

  for (var i = 0; i < tiles.length; i += concurrency) {
    final batch = tiles.sublist(i, math.min(i + concurrency, tiles.length));
    await Future.wait(batch.map((t) async {
      final (zoom, x, y) = t;
      final url = _osmTileUrl(zoom, x, y);
      try {
        // Check if already cached without a network hit.
        final cached = await manager.getFileFromCache(url);
        if (cached != null && await cached.file.exists()) {
          skipped++;
        } else {
          await manager.downloadFile(url, authHeaders: {
            'User-Agent': 'Spotto/1.0 (travel companion; contact: support@spotto.app)',
          });
          downloaded++;
        }
      } on SocketException {
        failed++;
      } catch (_) {
        failed++;
      }
      done++;
      onProgress?.call(done, total);
    }));
    // Polite delay between batches.
    await Future<void>.delayed(const Duration(milliseconds: 300));
  }

  return TileCacheResult(downloaded: downloaded, skipped: skipped, failed: failed);
}

/// Clears all cached map tiles from disk.
Future<void> clearMapTileCache() =>
    MapTileCacheManager().manager.emptyCache();

/// Returns approximate disk size of the tile cache in bytes.

Future<int> mapTileCacheSizeBytes() async {
  try {
    final tempDir = await getTemporaryDirectory();
    final cacheDir = Directory('${tempDir.path}/spotto_map_tiles');
    if (!await cacheDir.exists()) return 0;
    int total = 0;
    await for (final entity in cacheDir.list(recursive: true)) {
      if (entity is File) total += await entity.length();
    }
    return total;
  } catch (e) {
    return 0;
  }
}
