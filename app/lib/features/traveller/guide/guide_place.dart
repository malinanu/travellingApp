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
import '../../../core/utils/vibe_matching.dart';
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
  final String description;
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
  final int? vibeMatchPercent;
  final String? vibeBadge;

  const GuidePlace({
    required this.id,
    required this.name,
    this.description = '',
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
    this.vibeMatchPercent,
    this.vibeBadge,
  });

  factory GuidePlace.fromJson(Map<String, dynamic> j) => GuidePlace(
    id: j['id'] as String? ?? '',
    name: j['name'] as String? ?? '',
    description: j['description'] as String? ?? '',
    category: j['category'] as String? ?? '',
    region: j['region'] as String? ?? '',
    priceLevel: j['priceLevel'] as String? ?? '',
    rating: (j['rating'] as num?)?.toDouble() ?? 0,
    distanceKm: (j['distanceKm'] as num?)?.toDouble(),
    lat: (j['lat'] as num?)?.toDouble() ?? 0,
    lng: (j['lng'] as num?)?.toDouble() ?? 0,
    coverThumb: j['coverThumb'] as String? ?? '',
    coverMicro: j['coverMicro'] as String? ?? '',
    vibeMatchPercent: (j['vibeMatchPercent'] as num?)?.toInt(),
    vibeBadge: j['vibeBadge'] as String?,
  );

  /// Fallback for a reply that only carries place ids.
  factory GuidePlace.fromPlace(Place p, [String description = '', int? vibeMatchPercent, String? vibeBadge]) => GuidePlace(
    id: p.id,
    name: p.name,
    description: description,
    category: p.type,
    region: p.region,
    priceLevel: p.priceLevel,
    rating: p.rating,
    lat: p.lat,
    lng: p.lng,
    coverThumb: p.coverThumb,
    coverMicro: p.coverMicro,
    icon: p.icon,
    vibeMatchPercent: vibeMatchPercent,
    vibeBadge: vibeBadge,
  );

  GuidePlace copyWith({
    String? id,
    String? name,
    String? description,
    String? category,
    String? region,
    String? priceLevel,
    double? rating,
    double? distanceKm,
    double? lat,
    double? lng,
    String? coverThumb,
    String? coverMicro,
    String? icon,
    int? vibeMatchPercent,
    String? vibeBadge,
  }) => GuidePlace(
    id: id ?? this.id,
    name: name ?? this.name,
    description: description ?? this.description,
    category: category ?? this.category,
    region: region ?? this.region,
    priceLevel: priceLevel ?? this.priceLevel,
    rating: rating ?? this.rating,
    distanceKm: distanceKm ?? this.distanceKm,
    lat: lat ?? this.lat,
    lng: lng ?? this.lng,
    coverThumb: coverThumb ?? this.coverThumb,
    coverMicro: coverMicro ?? this.coverMicro,
    icon: icon ?? this.icon,
    vibeMatchPercent: vibeMatchPercent ?? this.vibeMatchPercent,
    vibeBadge: vibeBadge ?? this.vibeBadge,
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
    if (description.isNotEmpty) 'description': description,
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
    if (vibeMatchPercent != null) 'vibeMatchPercent': vibeMatchPercent,
    if (vibeBadge != null) 'vibeBadge': vibeBadge,
  };
}

/// Cards for one reply: the server's summaries when it sent them, otherwise
/// [ids] resolved against [cached] (ids not in it are skipped).
/// If neither produced cards, scans [replyText] to match known place names
/// as a resilient fallback. Backfills [GuidePlace.description] from [replyText]
/// if empty.
List<GuidePlace> guideCardsFor(
  List<GuidePlace> summaries,
  List<String> ids,
  List<Place> cached, [
  String replyText = '',
  UserPrefs? userPrefs,
]) {
  List<GuidePlace> cards;
  if (summaries.isNotEmpty) {
    cards = summaries.take(kGuideMaxCards).toList();
  } else {
    final byId = {for (final p in cached) p.id: p};
    final resolved = <GuidePlace>[];
    for (final id in ids) {
      final p = byId[id];
      if (p != null) {
        int? vibePercent;
        String? vibeBadge;
        if (userPrefs != null && (userPrefs.styles.isNotEmpty || userPrefs.interests.isNotEmpty)) {
          final vm = calculateVibeMatch(place: p, prefs: userPrefs);
          if (vm.percentage >= 65) {
            vibePercent = vm.percentage;
            vibeBadge = '$vibePercent% Vibe';
          }
        }
        resolved.add(GuidePlace.fromPlace(p, '', vibePercent, vibeBadge));
      }
    }
    if (resolved.isNotEmpty) {
      cards = resolved.take(kGuideMaxCards).toList();
    } else if (replyText.isNotEmpty) {
      final lower = replyText.toLowerCase();
      final matched = <GuidePlace>[];
      for (final p in cached) {
        if (p.name.length >= 4 && lower.contains(p.name.toLowerCase())) {
          int? vibePercent;
          String? vibeBadge;
          if (userPrefs != null && (userPrefs.styles.isNotEmpty || userPrefs.interests.isNotEmpty)) {
            final vm = calculateVibeMatch(place: p, prefs: userPrefs);
            if (vm.percentage >= 65) {
              vibePercent = vm.percentage;
              vibeBadge = '$vibePercent% Vibe';
            }
          }
          matched.add(GuidePlace.fromPlace(p, '', vibePercent, vibeBadge));
          if (matched.length >= kGuideMaxCards) break;
        }
      }
      cards = matched;
    } else {
      cards = const [];
    }
  }

  if (replyText.isNotEmpty && cards.isNotEmpty) {
    cards = [
      for (final c in cards)
        if (c.description.isEmpty)
          c.copyWith(description: extractPlaceDescriptionFromText(replyText, c.name))
        else
          c,
    ];
  }

  return cards;
}

String _normalizeApostrophes(String s) =>
    s.replaceAll('’', "'").replaceAll('‘', "'").replaceAll('`', "'").replaceAll('´', "'");

/// Extracts the 1-2 sentence recommendation or description associated with
/// [placeName] from the guide's reply [text].
String extractPlaceDescriptionFromText(String text, String placeName) {
  final name = placeName.trim();
  if (name.isEmpty || text.isEmpty) return '';
  final lines = text.split('\n');
  final normText = _normalizeApostrophes(text);
  final normLines = normText.split('\n');
  final normName = _normalizeApostrophes(name);
  final lowerName = normName.toLowerCase();
  final altName = lowerName.startsWith('the ') ? lowerName.substring(4).trim() : '';

  // First pass: list items or bullets
  for (var i = 0; i < lines.length; i++) {
    final trimmed = lines[i].trim();
    if (trimmed.isEmpty || !_isListItemOrHeader(trimmed)) continue;
    final lowerLine = normLines[i].toLowerCase();
    var idx = lowerLine.indexOf(lowerName);
    var matchedLen = normName.length;
    if (idx == -1 && altName.isNotEmpty) {
      idx = lowerLine.indexOf(altName);
      matchedLen = altName.length;
    }
    if (idx == -1) continue;

    var desc = _cleanPlaceDescriptionLine(trimmed.substring(idx + matchedLen));
    desc = _appendContinuationLines(lines, i + 1, desc);
    if (desc.length >= 6) return desc;
  }

  // Second pass: any line mentioning the place
  for (var i = 0; i < lines.length; i++) {
    final trimmed = lines[i].trim();
    if (trimmed.isEmpty) continue;
    final lowerLine = normLines[i].toLowerCase();
    var idx = lowerLine.indexOf(lowerName);
    var matchedLen = normName.length;
    if (idx == -1 && altName.isNotEmpty) {
      idx = lowerLine.indexOf(altName);
      matchedLen = altName.length;
    }
    if (idx == -1) continue;

    var desc = _cleanPlaceDescriptionLine(trimmed.substring(idx + matchedLen));
    desc = _appendContinuationLines(lines, i + 1, desc);
    if (desc.length >= 6) return desc;
  }

  return '';
}

String _appendContinuationLines(List<String> lines, int startIdx, String initialDesc) {
  var desc = initialDesc;
  if (startIdx >= lines.length) return desc;
  final rawNext = lines[startIdx];
  final nextTrimmed = rawNext.trim();
  if (nextTrimmed.isEmpty || _isListItemOrHeader(nextTrimmed)) return desc;

  if (desc.isEmpty) {
    return _cleanPlaceDescriptionLine(nextTrimmed);
  }

  final trimmedDesc = desc.trim();
  final lastChar = trimmedDesc.isNotEmpty ? trimmedDesc[trimmedDesc.length - 1] : '';
  final isIndented = rawNext.startsWith('  ') || rawNext.startsWith('\t');
  if ((lastChar == '.' || lastChar == '!' || lastChar == '?') && !isIndented) {
    return desc;
  }

  return '$desc $nextTrimmed';
}

String _cleanPlaceDescriptionLine(String raw) {
  var s = raw.trim();
  s = s.replaceAll(RegExp(r'''^[*_`'"]+'''), '').trim();

  // Strip markdown link URL if name was inside [Name](url)
  if (s.startsWith('](')) {
    final closeIdx = s.indexOf(')');
    if (closeIdx != -1 && closeIdx < 200) {
      s = s.substring(closeIdx + 1).trim();
    }
  }

  // Strip parentheses with metadata right after name, e.g. "(0.8 km · $$): "
  if (s.startsWith('(')) {
    final closeIdx = s.indexOf(')');
    if (closeIdx != -1 && closeIdx < 80) {
      s = s.substring(closeIdx + 1).trim();
    }
  }

  s = s.replaceAll(RegExp(r'^[ \t:\-–—·,]+'), '').trim();
  s = s.replaceAll(RegExp(r'''^[*_`'"]+'''), '').trim();
  s = s.replaceAll(RegExp(r'''[*_`'"]+$'''), '').trim();
  return s;
}

bool _isListItemOrHeader(String s) {
  if (s.startsWith('#') || s.startsWith('-') || s.startsWith('*') || s.startsWith('+')) {
    return true;
  }
  return RegExp(r'^\d+[\.\)]\s').hasMatch(s);
}

class GuidePlaceCard extends ConsumerStatefulWidget {
  final GuidePlace place;
  final String? description;
  const GuidePlaceCard({super.key, required this.place, this.description});

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
    final vibePercent = p.vibeMatchPercent;
    final badgeLabel = (p.vibeBadge != null && p.vibeBadge!.isNotEmpty)
        ? p.vibeBadge!
        : (vibePercent != null ? '$vibePercent% Vibe' : null);
    final rawDesc = (widget.description != null && widget.description!.trim().isNotEmpty)
        ? widget.description!.trim()
        : p.description.trim();
    final desc = rawDesc.replaceAll('**', '').replaceAll('__', '');

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
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          mainAxisSize: MainAxisSize.min,
          children: [
            Row(
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
                          if (badgeLabel != null && (vibePercent == null || vibePercent >= 65))
                            Container(
                              padding: const EdgeInsets.symmetric(horizontal: 5, vertical: 1.5),
                              decoration: BoxDecoration(
                                color: const Color(0xFFE0F2F1),
                                borderRadius: BorderRadius.circular(5),
                                border: Border.all(color: const Color(0xFF00796B).withValues(alpha: 0.25), width: 0.8),
                              ),
                              child: Row(
                                mainAxisSize: MainAxisSize.min,
                                children: [
                                  const Icon(Icons.bolt_rounded, size: 11, color: Color(0xFF00796B)),
                                  const SizedBox(width: 1.5),
                                  Text(
                                    badgeLabel,
                                    style: const TextStyle(
                                      fontSize: 10,
                                      fontWeight: FontWeight.w800,
                                      color: Color(0xFF004D40),
                                    ),
                                  ),
                                ],
                              ),
                            ),
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
                          if (p.region.isNotEmpty)
                            Container(
                              padding: const EdgeInsets.symmetric(horizontal: 5, vertical: 1.5),
                              decoration: BoxDecoration(
                                color: SpottoColors.line.withValues(alpha: 0.5),
                                borderRadius: BorderRadius.circular(5),
                              ),
                              child: Row(
                                mainAxisSize: MainAxisSize.min,
                                children: [
                                  const Icon(Icons.place_outlined, size: 10, color: SpottoColors.muted),
                                  const SizedBox(width: 1.5),
                                  Text(
                                    p.region,
                                    style: const TextStyle(
                                      fontSize: 10,
                                      fontWeight: FontWeight.w700,
                                      color: SpottoColors.muted,
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
            if (desc.isNotEmpty) ...[
              const SizedBox(height: 8),
              Container(
                width: double.infinity,
                padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 7.5),
                decoration: BoxDecoration(
                  color: SpottoColors.accentTint.withValues(alpha: 0.35),
                  borderRadius: BorderRadius.circular(10),
                  border: Border.all(
                    color: SpottoColors.accent.withValues(alpha: 0.12),
                    width: 1,
                  ),
                ),
                child: Row(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Padding(
                      padding: const EdgeInsets.only(top: 1.5, right: 6),
                      child: Icon(
                        Icons.format_quote_rounded,
                        size: 13.5,
                        color: SpottoColors.accent.withValues(alpha: 0.75),
                      ),
                    ),
                    Expanded(
                      child: Text(
                        desc,
                        style: TextStyle(
                          fontSize: 12.8,
                          height: 1.45,
                          fontWeight: FontWeight.w500,
                          color: onSurface.withValues(alpha: 0.88),
                        ),
                      ),
                    ),
                  ],
                ),
              ),
            ],
          ],
        ),
      ),
    );
  }
}

/// Parsed sections of a guide turn that recommends places.
class PlaceTurnContent {
  final String intro;
  final Map<String, String> placeDescriptions;
  final String outro;

  const PlaceTurnContent({
    this.intro = '',
    this.placeDescriptions = const {},
    this.outro = '',
  });
}

/// Separates the guide's reply text into an introductory greeting/context,
/// specific descriptions for each recommended place, and a concluding outro
/// (e.g. follow-up questions).
PlaceTurnContent parsePlaceTurnContent(String rawText, List<GuidePlace> places) {
  if (places.isEmpty || rawText.trim().isEmpty) {
    return PlaceTurnContent(intro: rawText.trim());
  }

  final lines = rawText.split('\n');
  final normText = _normalizeApostrophes(rawText);
  final normLines = normText.split('\n');

  final placeLineIndices = <int>[];
  final descriptions = <String, String>{};

  for (final p in places) {
    final normName = _normalizeApostrophes(p.name.trim());
    final nameLower = normName.toLowerCase();
    if (nameLower.isEmpty) continue;
    final altName = nameLower.startsWith('the ') ? nameLower.substring(4).trim() : '';

    // Prefer list item / header line
    var foundIdx = -1;
    for (var i = 0; i < lines.length; i++) {
      final lineLower = normLines[i].toLowerCase();
      final hasName = lineLower.contains(nameLower) || (altName.isNotEmpty && lineLower.contains(altName));
      if (_isListItemOrHeader(lines[i].trim()) && hasName) {
        foundIdx = i;
        break;
      }
    }
    // Fallback: any line
    if (foundIdx == -1) {
      for (var i = 0; i < lines.length; i++) {
        final lineLower = normLines[i].toLowerCase();
        final hasName = lineLower.contains(nameLower) || (altName.isNotEmpty && lineLower.contains(altName));
        if (hasName) {
          foundIdx = i;
          break;
        }
      }
    }

    if (foundIdx != -1) {
      placeLineIndices.add(foundIdx);
      final desc = p.description.isNotEmpty
          ? p.description
          : extractPlaceDescriptionFromText(rawText, p.name);
      if (desc.isNotEmpty) {
        descriptions[p.name] = desc;
        descriptions[p.id] = desc;
      }
    }
  }

  if (placeLineIndices.isEmpty) {
    return PlaceTurnContent(intro: rawText.trim());
  }

  placeLineIndices.sort();
  final firstPlaceLine = placeLineIndices.first;

  // Intro: lines before firstPlaceLine
  final introLines = lines.sublist(0, firstPlaceLine);
  final intro = introLines.join('\n').trim();

  // Find where the place recommendations section ends.
  var lastPlaceEnd = placeLineIndices.last;

  // Check if lastPlaceEnd has continuation line(s)
  if (lastPlaceEnd + 1 < lines.length) {
    final nextRaw = lines[lastPlaceEnd + 1];
    final nextTrimmed = nextRaw.trim();
    if (nextTrimmed.isNotEmpty && !_isListItemOrHeader(nextTrimmed)) {
      final prevTrimmed = lines[lastPlaceEnd].trim();
      final prevEnded = prevTrimmed.endsWith('.') || prevTrimmed.endsWith('!') || prevTrimmed.endsWith('?');
      final isIndented = nextRaw.startsWith('  ') || nextRaw.startsWith('\t');
      if (!prevEnded || isIndented) {
        lastPlaceEnd++;
      }
    }
  }

  // Advance past any trailing list items in the same recommendation block (e.g. uncataloged items)
  var cursor = lastPlaceEnd;
  while (cursor + 1 < lines.length) {
    final nextTrimmed = lines[cursor + 1].trim();
    if (nextTrimmed.isEmpty) {
      // Look ahead: is the next non-empty line another list item?
      var hasMoreListItems = false;
      for (var k = cursor + 2; k < lines.length; k++) {
        final kTrimmed = lines[k].trim();
        if (kTrimmed.isEmpty) continue;
        if (_isListItemOrHeader(kTrimmed)) {
          hasMoreListItems = true;
        }
        break;
      }
      if (hasMoreListItems) {
        cursor++;
        continue;
      }
      break;
    }
    if (_isListItemOrHeader(nextTrimmed)) {
      cursor++;
      // check if this list item has a continuation line
      if (cursor + 1 < lines.length) {
        final contRaw = lines[cursor + 1];
        final contTrimmed = contRaw.trim();
        final prevEnded = nextTrimmed.endsWith('.') || nextTrimmed.endsWith('!') || nextTrimmed.endsWith('?');
        final isIndented = contRaw.startsWith('  ') || contRaw.startsWith('\t');
        if (contTrimmed.isNotEmpty && !_isListItemOrHeader(contTrimmed) && (!prevEnded || isIndented)) {
          cursor++;
        }
      }
      continue;
    }
    break;
  }
  lastPlaceEnd = cursor;

  // Outro: lines after lastPlaceEnd
  var outro = '';
  if (lastPlaceEnd + 1 < lines.length) {
    final outroLines = lines.sublist(lastPlaceEnd + 1);
    outro = outroLines.join('\n').trim();
  }

  return PlaceTurnContent(
    intro: intro,
    placeDescriptions: descriptions,
    outro: outro,
  );
}

