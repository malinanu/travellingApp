// A place card under a Local Guide reply: photo, "0.8 km · $$ · ★ 4.6",
// directions, and tap-through to the place drawer. Built from the server's
// `places` summaries (see server/internal/http/guide_places.go
// guidePlaceCard), or from the cached places list for replies from servers
// older than that.

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:url_launcher/url_launcher.dart';

import '../../../core/api/api_client.dart';
import '../../../core/theme/palette.dart';
import '../../../core/widgets/press_scale.dart';
import '../../../core/widgets/progressive_image.dart';
import '../../../core/widgets/spotto_icons.dart';
import '../../../data/models/models.dart';
import '../../../data/repositories/repositories.dart';
import '../place_detail/business_drawer.dart';

/// Most cards shown under one reply.
const kGuideMaxCards = 4;

class GuidePlace {
  final String id;
  final String name;
  final String category;
  final String region;
  final String priceLevel;
  final double rating;
  final double? distanceKm;
  final double lat;
  final double lng;
  final String coverThumb;
  final String coverMicro;
  final String icon;

  const GuidePlace({
    required this.id,
    required this.name,
    this.category = '',
    this.region = '',
    this.priceLevel = '',
    this.rating = 0,
    this.distanceKm,
    this.lat = 0,
    this.lng = 0,
    this.coverThumb = '',
    this.coverMicro = '',
    this.icon = '',
  });

  factory GuidePlace.fromJson(Map<String, dynamic> j) => GuidePlace(
    id: j['id'] as String? ?? '',
    name: j['name'] as String? ?? '',
    category: j['category'] as String? ?? '',
    region: j['region'] as String? ?? '',
    priceLevel: j['priceLevel'] as String? ?? '',
    rating: (j['rating'] as num?)?.toDouble() ?? 0,
    distanceKm: (j['distanceKm'] as num?)?.toDouble(),
    lat: (j['lat'] as num?)?.toDouble() ?? 0,
    lng: (j['lng'] as num?)?.toDouble() ?? 0,
    coverThumb: j['coverThumb'] as String? ?? '',
    coverMicro: j['coverMicro'] as String? ?? '',
  );

  /// Fallback for a reply that only carries place ids.
  factory GuidePlace.fromPlace(Place p) => GuidePlace(
    id: p.id,
    name: p.name,
    category: p.type,
    region: p.region,
    priceLevel: p.priceLevel,
    rating: p.rating,
    lat: p.lat,
    lng: p.lng,
    coverThumb: p.coverThumb,
    coverMicro: p.coverMicro,
    icon: p.icon,
  );

  /// "0.8 km · $$ · ★ 4.6", leaving out whatever isn't known.
  String get metaLine => [
    if (distanceKm != null) '${distanceKm! < 10 ? distanceKm!.toStringAsFixed(1) : distanceKm!.round()} km',
    if (priceLevel.isNotEmpty) priceLevel,
    if (rating > 0) '★ ${rating.toStringAsFixed(1)}',
    if (distanceKm == null && region.isNotEmpty) region,
  ].join(' · ');

  Uri? get directionsUrl => lat == 0 && lng == 0
      ? null
      : Uri.parse('https://www.google.com/maps/dir/?api=1&destination=$lat,$lng');
}

/// Cards for one reply: the server's summaries when it sent them, otherwise
/// [ids] resolved against [cached] (ids not in it are skipped).
List<GuidePlace> guideCardsFor(List<GuidePlace> summaries, List<String> ids, List<Place> cached) {
  if (summaries.isNotEmpty) return summaries.take(kGuideMaxCards).toList();
  final byId = {for (final p in cached) p.id: p};
  return [
    for (final id in ids)
      if (byId[id] != null) GuidePlace.fromPlace(byId[id]!),
  ].take(kGuideMaxCards).toList();
}

class GuidePlaceCard extends ConsumerStatefulWidget {
  final GuidePlace place;
  const GuidePlaceCard({super.key, required this.place});

  @override
  ConsumerState<GuidePlaceCard> createState() => _GuidePlaceCardState();
}

class _GuidePlaceCardState extends ConsumerState<GuidePlaceCard> {
  bool _opening = false;

  /// Opens the place drawer — straight from the cached places list, or
  /// after fetching the place if the guide found one the app hasn't loaded.
  Future<void> _open() async {
    final cached = ref.read(placesProvider).where((p) => p.id == widget.place.id).firstOrNull;
    if (cached != null) {
      showBusinessDrawer(context, cached);
      return;
    }
    setState(() => _opening = true);
    try {
      final place = await ref.read(placeDetailsProvider(widget.place.id).future);
      if (mounted) showBusinessDrawer(context, place);
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(ApiClient.errorMessage(e))));
      }
    } finally {
      if (mounted) setState(() => _opening = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final p = widget.place;
    final onSurface = Theme.of(context).colorScheme.onSurface;
    final meta = p.metaLine;
    final directions = p.directionsUrl;
    return PressScale(
      scale: 0.98,
      onTap: _opening ? null : _open,
      child: Container(
        constraints: BoxConstraints(maxWidth: MediaQuery.of(context).size.width * 0.84),
        padding: const EdgeInsets.all(8),
        decoration: BoxDecoration(
          color: Theme.of(context).colorScheme.surface,
          borderRadius: BorderRadius.circular(16),
          boxShadow: const [BoxShadow(color: Color(0x2E103D3A), blurRadius: 12, offset: Offset(0, 2))],
        ),
        child: Row(
          children: [
            ClipRRect(
              borderRadius: BorderRadius.circular(12),
              child: SizedBox(
                width: 64,
                height: 64,
                child: p.coverThumb.isEmpty
                    ? Container(
                        color: SpottoColors.accentTint,
                        child: Icon(spottoIcon(p.icon), size: 22, color: SpottoColors.accent),
                      )
                    : ProgressiveImage(
                        url: resolveMediaUrl(p.coverThumb),
                        microThumb: p.coverMicro,
                        width: 64,
                        height: 64,
                        errorBuilder: (_) => Container(
                          color: SpottoColors.accentTint,
                          child: const Icon(Icons.place, size: 22, color: SpottoColors.accent),
                        ),
                      ),
              ),
            ),
            const SizedBox(width: 10),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    p.name,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: TextStyle(fontSize: 13.5, fontWeight: FontWeight.w800, color: onSurface),
                  ),
                  if (p.category.isNotEmpty)
                    Text(
                      p.category,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: const TextStyle(fontSize: 11.5, fontWeight: FontWeight.w600, color: SpottoColors.muted),
                    ),
                  if (meta.isNotEmpty)
                    Padding(
                      padding: const EdgeInsets.only(top: 2),
                      child: Text(
                        meta,
                        style: TextStyle(fontSize: 12, fontWeight: FontWeight.w700, color: onSurface),
                      ),
                    ),
                ],
              ),
            ),
            if (_opening)
              const Padding(
                padding: EdgeInsets.all(10),
                child: SizedBox(width: 16, height: 16, child: CircularProgressIndicator(strokeWidth: 2)),
              )
            else if (directions != null)
              IconButton(
                tooltip: 'Directions',
                onPressed: () => launchUrl(directions, mode: LaunchMode.externalApplication),
                icon: const Icon(Icons.directions, color: SpottoColors.accent),
              ),
          ],
        ),
      ),
    );
  }
}
