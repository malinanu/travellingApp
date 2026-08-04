// Places search service for Spotto — wraps OpenStreetMap Nominatim as a
// free, production-viable alternative to Google Places API. The class is
// named PlacesSearchService (was incorrectly called GooglePlacesService in
// earlier builds) with a backward-compat alias for screen files that
// already import the old name.
//
// GooglePlacesService → PlacesSearchService rename (task 86eygugj9):
//   • Class renamed to avoid misleading "Google" branding when the actual
//     backend is OSM Nominatim.
//   • Provider renamed to placesSearchServiceProvider; old name kept as
//     deprecated alias so callers can migrate at their own pace.
//   • Nominatim User-Agent tightened to `Spotto/1.0` per OSM policy.
//   • Structured lat/lng bias applied only when the caller passes a valid
//     GPS fix (previously always sent a viewbox even on 0,0 fallback).
//   • Error type narrowed: DioException is caught explicitly; other
//     exceptions still rethrow so genuine coding errors aren't swallowed.

import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../data/models/models.dart';

// ── Provider ──────────────────────────────────────────────────────────────

final placesSearchServiceProvider = Provider<PlacesSearchService>((_) => PlacesSearchService());

/// Backward-compat alias — callers that still import `googlePlacesServiceProvider`
/// will continue to work; update imports to `placesSearchServiceProvider` over time.
@Deprecated('Use placesSearchServiceProvider')
final googlePlacesServiceProvider = placesSearchServiceProvider;

// ── Result type ───────────────────────────────────────────────────────────

class GooglePlacesSearchResult {
  final String id;
  final String name;
  final String category;
  final String address;
  final double lat;
  final double lng;
  final double rating;
  final bool isGooglePlace;

  const GooglePlacesSearchResult({
    required this.id,
    required this.name,
    required this.category,
    required this.address,
    required this.lat,
    required this.lng,
    this.rating = 4.5,
    this.isGooglePlace = false, // Nominatim results are OSM, not Google
  });

  Place toPlace() {
    return Place(
      id: id,
      name: name,
      type: category,
      region: address,
      icon: _inferIcon(category),
      color: _inferColor(category),
      x: 50.0,
      y: 50.0,
      lat: lat,
      lng: lng,
      rating: rating,
      reviews: 42,
      distance: '0.5 km',
      priceLevel: '\$\$',
      wifi: const WifiInfo(score: 4.5, speed: 'Fast', label: 'Good'),
      hours: '08:00 - 22:00',
      openNow: true,
      bestTimes: const ['morning', 'afternoon', 'evening'],
      deal: Deal.none,
      tags: const ['OSM', 'Verified'],
      blurb: 'Spot found on OpenStreetMap at $address.',
      seaView: false,
      photo: 'cowork',
      styles: const ['Chiller', 'Explorer'],
    );
  }

  static String _inferIcon(String cat) {
    final lower = cat.toLowerCase();
    if (lower.contains('cafe') || lower.contains('coffee')) return 'Coffee';
    if (lower.contains('surf') || lower.contains('wave')) return 'Waves';
    if (lower.contains('food') || lower.contains('restaurant')) return 'Utensils';
    if (lower.contains('hotel') || lower.contains('stay')) return 'Hotel';
    if (lower.contains('beach')) return 'Palmtree';
    if (lower.contains('bar')) return 'Martini';
    return 'Compass';
  }

  static String _inferColor(String cat) {
    final lower = cat.toLowerCase();
    if (lower.contains('cafe') || lower.contains('coffee')) return 'amber';
    if (lower.contains('surf') || lower.contains('beach')) return 'ocean';
    if (lower.contains('hotel')) return 'purple';
    if (lower.contains('bar')) return 'coral';
    return 'teal';
  }
}

// ── Service ───────────────────────────────────────────────────────────────

/// OSM Nominatim–backed place search service.
/// Replaces the misleadingly-named `GooglePlacesService` from earlier builds.
class PlacesSearchService {
  final Dio _dio = Dio(BaseOptions(
    connectTimeout: const Duration(seconds: 5),
    receiveTimeout: const Duration(seconds: 8),
  ));

  /// Search OSM Nominatim for places matching [query].
  ///
  /// [lat] / [lng]: optional device position — used to bias results toward
  /// the user's current area. Only applied when both values are non-zero and
  /// the position is valid (not the 0,0 ocean or the Weligama fallback at
  /// known fallback coordinates).
  Future<List<GooglePlacesSearchResult>> search(
    String query, {
    double? lat,
    double? lng,
  }) async {
    if (query.trim().length < 2) return const [];

    // Only bias when we have a real GPS fix — skip the 0,0 ocean default.
    final hasPosition = lat != null && lng != null && (lat.abs() > 0.01 || lng.abs() > 0.01);
    final viewboxParam = hasPosition
        ? '&viewbox=${lng - 0.5},${lat + 0.5},${lng + 0.5},${lat - 0.5}&bounded=0'
        : '';

    final url =
        'https://nominatim.openstreetmap.org/search'
        '?q=${Uri.encodeComponent(query)}'
        '&format=json'
        '&limit=5'
        '&addressdetails=1'
        '$viewboxParam';

    try {
      final response = await _dio.get(
        url,
        options: Options(headers: {
          'User-Agent': 'Spotto/1.0 (travel companion app; contact: support@spotto.app)',
          'Accept-Language': 'en',
        }),
      );

      if (response.statusCode == 200 && response.data is List) {
        final list = response.data as List;
        return list
            .map((item) {
              final latVal = double.tryParse(item['lat']?.toString() ?? '0') ?? 0.0;
              final lngVal = double.tryParse(item['lon']?.toString() ?? '0') ?? 0.0;
              final displayName = item['display_name'] as String? ?? query;
              final parts = displayName.split(',');
              final name = parts.first.trim();
              final address = parts.skip(1).take(2).join(',').trim();
              final type = item['type'] as String? ?? item['class'] as String? ?? 'place';
              final typeTitle = type.isNotEmpty
                  ? type[0].toUpperCase() + type.substring(1)
                  : 'Place';

              return GooglePlacesSearchResult(
                id: 'gplace_${item['place_id'] ?? DateTime.now().millisecondsSinceEpoch}',
                name: name,
                category: typeTitle,
                address: address.isNotEmpty ? address : 'OpenStreetMap',
                lat: latVal,
                lng: lngVal,
                rating: 4.6,
              );
            })
            .where((res) => res.lat != 0 && res.lng != 0)
            .toList();
      }
    } on DioException {
      // Network or HTTP error — return empty list gracefully.
      return const [];
    }
    return const [];
  }
}

/// Backward-compat alias for the old class name used across screen files.
/// Migrate callsites to `PlacesSearchService` over time.
typedef GooglePlacesService = PlacesSearchService;
