// A place card under a Local Guide reply: photo, "0.8 km · $$ · ★ 4.6",
// directions, and tap-through to the place drawer. Built from the server's
// `places` summaries (see server/internal/http/guide_places.go
// guidePlaceCard), or from the cached places list for replies from servers
// older than that.

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:url_launcher/url_launcher.dart';

import '../../../core/api/api_client.dart';
import '../../../core/theme/palette.dart';
import '../../../core/widgets/press_scale.dart';
import '../../../core/widgets/progressive_image.dart';
import '../../../core/widgets/spotto_icons.dart';
import '../../../data/models/models.dart';
import '../../../data/repositories/repositories.dart';
import '../place_detail/business_drawer.dart';
import 'guide_cards.dart';
import 'guide_payload.dart';

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

  /// "800 m · $$ · ★ 4.6" or "122 km · $$ · ★ 4.6", leaving out whatever isn't known.
  String get metaLine => [
    if (distanceKm != null)
      distanceKm! < 1.0
          ? '${(distanceKm! * 1000).round()} m'
          : '${distanceKm! < 10 ? distanceKm!.toStringAsFixed(1) : distanceKm!.round()} km',
    if (priceLevel.isNotEmpty) priceLevel,
    if (rating > 0) '★ ${rating.toStringAsFixed(1)}',
    if (distanceKm == null && region.isNotEmpty) region,
  ].join(' · ');

  Uri? get directionsUrl => lat == 0 && lng == 0
      ? null
      : Uri.parse('https://www.google.com/maps/dir/?api=1&destination=$lat,$lng');

  Map<String, dynamic> toJson() => {
    'id': id,
    'name': name,
    'category': category,
    'region': region,
    'priceLevel': priceLevel,
    'rating': rating,
    if (distanceKm != null) 'distanceKm': distanceKm,
    'lat': lat,
    'lng': lng,
    'coverThumb': coverThumb,
    'coverMicro': coverMicro,
    'icon': icon,
  };
}

/// Cards for one reply: the server's summaries when it sent them, otherwise
/// [ids] resolved against [cached] (ids not in it are skipped).
/// If neither produced cards, scans [replyText] to match known place names
/// as a resilient fallback.
List<GuidePlace> guideCardsFor(
  List<GuidePlace> summaries,
  List<String> ids,
  List<Place> cached, [
  String replyText = '',
]) {
  if (summaries.isNotEmpty) return summaries.take(kGuideMaxCards).toList();
  final byId = {for (final p in cached) p.id: p};
  final cards = [
    for (final id in ids)
      if (byId[id] != null) GuidePlace.fromPlace(byId[id]!),
  ];
  if (cards.isNotEmpty) return cards.take(kGuideMaxCards).toList();

  // Resilient fallback: if the reply mentioned known places in text, show their cards
  if (replyText.isNotEmpty) {
    final lower = replyText.toLowerCase();
    final matched = <GuidePlace>[];
    for (final p in cached) {
      if (p.name.length >= 4 && lower.contains(p.name.toLowerCase())) {
        matched.add(GuidePlace.fromPlace(p));
        if (matched.length >= kGuideMaxCards) break;
      }
    }
    if (matched.isNotEmpty) return matched;
  }
  return const [];
}

class GuidePlaceCard extends ConsumerStatefulWidget {
  final GuidePlace place;
  const GuidePlaceCard({super.key, required this.place});

  @override
  ConsumerState<GuidePlaceCard> createState() => _GuidePlaceCardState();
}

class _GuidePlaceCardState extends ConsumerState<GuidePlaceCard> {
  bool _opening = false;
  bool _adding = false;

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

  Future<void> _addToTrip() async {
    final p = widget.place;
    setState(() => _adding = true);
    try {
      final isGeo = p.id.startsWith('gp_') || p.id.startsWith('g:');
      final cleanId = p.id.replaceFirst(RegExp(r'^(p:|g:)'), '');
      final action = GuideCalendarAction(
        title: p.name,
        dayCount: 1,
        stops: [
          GuideCalendarStop(
            dayIndex: 0,
            place: GuidePlanPlace(
              ref: isGeo ? 'g:$cleanId' : 'p:$cleanId',
              placeId: isGeo ? null : cleanId,
              geoPlaceId: isGeo ? cleanId : null,
              name: p.name,
              kind: p.category,
              rating: p.rating > 0 ? p.rating : null,
              lat: p.lat,
              lng: p.lng,
              photoUrl: p.coverThumb.isNotEmpty ? p.coverThumb : null,
            ),
          ),
        ],
      );
      final name = await applyGuideCalendar(context, ref, action, 'ask');
      if (name != null && mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text('Added ${p.name} to $name'),
            action: SnackBarAction(label: 'Open', onPressed: () => context.go('/t/trip')),
          ),
        );
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(ApiClient.errorMessage(e))));
      }
    } finally {
      if (mounted) setState(() => _adding = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final p = widget.place;
    final onSurface = Theme.of(context).colorScheme.onSurface;
    final directions = p.directionsUrl;
    return PressScale(
      scale: 0.98,
      onTap: (_opening || _adding) ? null : _open,
      child: Container(
        constraints: BoxConstraints(maxWidth: MediaQuery.of(context).size.width * 0.86),
        padding: const EdgeInsets.all(10),
        decoration: BoxDecoration(
          color: Theme.of(context).colorScheme.surface,
          borderRadius: BorderRadius.circular(16),
          boxShadow: const [BoxShadow(color: Color(0x2E103D3A), blurRadius: 12, offset: Offset(0, 2))],
        ),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.center,
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
                mainAxisSize: MainAxisSize.min,
                children: [
                  Text(
                    p.name,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: TextStyle(fontSize: 13.5, fontWeight: FontWeight.w800, color: onSurface),
                  ),
                  const SizedBox(height: 3),
                  Wrap(
                    spacing: 4,
                    runSpacing: 4,
                    crossAxisAlignment: WrapCrossAlignment.center,
                    children: [
                      if (p.category.isNotEmpty)
                        Container(
                          padding: const EdgeInsets.symmetric(horizontal: 5, vertical: 1.5),
                          decoration: BoxDecoration(
                            color: SpottoColors.accentTint,
                            borderRadius: BorderRadius.circular(5),
                          ),
                          child: Text(
                            p.category,
                            style: const TextStyle(
                              fontSize: 10,
                              fontWeight: FontWeight.w700,
                              color: SpottoColors.accent,
                            ),
                          ),
                        ),
                      if (p.rating > 0)
                        Container(
                          padding: const EdgeInsets.symmetric(horizontal: 5, vertical: 1.5),
                          decoration: BoxDecoration(
                            color: const Color(0xFFFEF3C7),
                            borderRadius: BorderRadius.circular(5),
                          ),
                          child: Row(
                            mainAxisSize: MainAxisSize.min,
                            children: [
                              const Icon(Icons.star_rounded, size: 11, color: Color(0xFFD97706)),
                              const SizedBox(width: 1.5),
                              Text(
                                p.rating.toStringAsFixed(1),
                                style: const TextStyle(
                                  fontSize: 10,
                                  fontWeight: FontWeight.w800,
                                  color: Color(0xFF92400E),
                                ),
                              ),
                            ],
                          ),
                        ),
                      if (p.distanceKm != null)
                        Container(
                          padding: const EdgeInsets.symmetric(horizontal: 5, vertical: 1.5),
                          decoration: BoxDecoration(
                            color: SpottoColors.line.withValues(alpha: 0.5),
                            borderRadius: BorderRadius.circular(5),
                          ),
                          child: Text(
                            p.distanceKm! < 1.0
                                ? '${(p.distanceKm! * 1000).round()}m'
                                : '${p.distanceKm! < 10 ? p.distanceKm!.toStringAsFixed(1) : p.distanceKm!.round()}km',
                            style: const TextStyle(
                              fontSize: 10,
                              fontWeight: FontWeight.w700,
                              color: SpottoColors.muted,
                            ),
                          ),
                        ),
                      if (p.priceLevel.isNotEmpty)
                        Container(
                          padding: const EdgeInsets.symmetric(horizontal: 5, vertical: 1.5),
                          decoration: BoxDecoration(
                            color: SpottoColors.line.withValues(alpha: 0.5),
                            borderRadius: BorderRadius.circular(5),
                          ),
                          child: Text(
                            p.priceLevel,
                            style: const TextStyle(
                              fontSize: 10,
                              fontWeight: FontWeight.w700,
                              color: SpottoColors.muted,
                            ),
                          ),
                        ),
                    ],
                  ),
                ],
              ),
            ),
            const SizedBox(width: 6),
            if (_opening || _adding)
              const Padding(
                padding: EdgeInsets.all(8),
                child: SizedBox(width: 18, height: 18, child: CircularProgressIndicator(strokeWidth: 2)),
              )
            else
              Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  PressScale(
                    scale: 0.92,
                    onTap: _addToTrip,
                    child: Container(
                      padding: const EdgeInsets.symmetric(horizontal: 7, vertical: 5),
                      decoration: BoxDecoration(
                        color: SpottoColors.accentTint,
                        borderRadius: BorderRadius.circular(8),
                      ),
                      child: const Row(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          Icon(Icons.add_rounded, size: 14, color: SpottoColors.accent),
                          SizedBox(width: 1),
                          Text(
                            'Trip',
                            style: TextStyle(
                              fontSize: 11,
                              fontWeight: FontWeight.w700,
                              color: SpottoColors.accent,
                            ),
                          ),
                        ],
                      ),
                    ),
                  ),
                  if (directions != null) ...[
                    const SizedBox(width: 4),
                    PressScale(
                      scale: 0.92,
                      onTap: () => launchUrl(directions, mode: LaunchMode.externalApplication),
                      child: Container(
                        padding: const EdgeInsets.all(5),
                        decoration: BoxDecoration(
                          color: SpottoColors.line.withValues(alpha: 0.45),
                          borderRadius: BorderRadius.circular(8),
                        ),
                        child: const Icon(Icons.directions_outlined, size: 16, color: SpottoColors.accent),
                      ),
                    ),
                  ],
                ],
              ),
          ],
        ),
      ),
    );
  }
}
