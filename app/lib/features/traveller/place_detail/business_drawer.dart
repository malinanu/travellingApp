// Spot detail bottom sheet — port of BusinessDrawer.jsx: illustration
// header, stats, deal claim, vibe voting, artisan check-in, menu, actions.

import 'dart:typed_data';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:image_picker/image_picker.dart';
import 'package:qr_flutter/qr_flutter.dart';
import 'package:url_launcher/url_launcher.dart';
import 'package:video_player/video_player.dart';

import '../../../core/api/api_client.dart';
import '../../../core/api/upload_service.dart';
import '../../../core/theme/palette.dart';
import '../../../core/widgets/illustration.dart';
import '../../../core/widgets/pill.dart';
import '../../../core/widgets/press_scale.dart';
import '../../../core/widgets/sheet_scaffold.dart';
import '../../../core/widgets/spotto_icons.dart';
import '../../../core/widgets/stat_chip.dart';
import '../../../data/mock/places.dart' as data;
import '../../../data/models/models.dart';
import '../../../data/repositories/repositories.dart';

/// Google Maps search query for a place — lat/lng is unambiguous and needs
/// no town/region data, unlike the old hardcoded "place name + Weligama"
/// string that broke for every place outside that one town.
String _mapsQuery(Place place) {
  if (place.lat != 0 || place.lng != 0) {
    return '${place.lat},${place.lng}';
  }
  final locality = place.region.isNotEmpty ? place.region : 'Sri Lanka';
  return Uri.encodeComponent('${place.name}, $locality');
}

Future<void> showBusinessDrawer(BuildContext context, Place place) {
  return showModalBottomSheet(
    context: context,
    isScrollControlled: true,
    backgroundColor: Colors.transparent,
    builder: (context) => DraggableScrollableSheet(
      initialChildSize: 0.72,
      maxChildSize: 0.95,
      minChildSize: 0.5,
      expand: false,
      builder: (context, controller) => SheetScaffold(
        child: SingleChildScrollView(
          controller: controller,
          child: BusinessDrawerBody(place: place),
        ),
      ),
    ),
  );
}

class BusinessDrawerBody extends ConsumerWidget {
  final Place place;
  const BusinessDrawerBody({super.key, required this.place});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final text = Theme.of(context).textTheme;
    final claimedCode = ref.watch(claimsProvider)[place.id];

    // Live Google Places details (phone/website/wheelchair/price level) —
    // fetched lazily per-place and cached server-side; only overrides a
    // field when it actually has something the list-derived `place` didn't,
    // and never blocks the sheet on a slow/failed fetch.
    final details = ref.watch(placeDetailsProvider(place.id)).valueOrNull;
    final contact = (details != null && !details.contact.isEmpty) ? details.contact : place.contact;
    final wheelchairAccessible = details?.wheelchairAccessible ?? place.wheelchairAccessible;
    final priceLevel = (details != null && details.priceLevel.isNotEmpty)
        ? details.priceLevel
        : place.priceLevel;
    final favorites = ref.watch(favoritesProvider);
    final isFavorite = favorites.contains(place.id);

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        // illustration header
        Padding(
          padding: const EdgeInsets.fromLTRB(16, 6, 16, 0),
          child: ClipRRect(
            borderRadius: BorderRadius.circular(22),
            child: SizedBox(
              height: 170,
              child: Stack(
                fit: StackFit.expand,
                children: [
                  // Real uploaded cover photo when the vendor has one, else
                  // fall back to the illustration.
                  if (place.cover.isNotEmpty)
                    Image.network(
                      resolveMediaUrl(place.cover),
                      fit: BoxFit.cover,
                      errorBuilder: (_, __, ___) => Illustration(kind: place.photo),
                    )
                  else
                    Illustration(kind: place.photo),
                  // Bottom scrim so the white name/subtitle stays readable
                  // over a photographic cover.
                  if (place.cover.isNotEmpty)
                    const DecoratedBox(
                      decoration: BoxDecoration(
                        gradient: LinearGradient(
                          begin: Alignment.topCenter,
                          end: Alignment.bottomCenter,
                          colors: [Colors.transparent, Colors.black54],
                        ),
                      ),
                    ),
                  Positioned(
                    left: 16,
                    bottom: 12,
                    right: 16,
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(place.name, style: text.titleLarge?.copyWith(color: Colors.white)),
                        Text(
                            [place.type, place.distance, priceLevel]
                                .where((s) => s.isNotEmpty)
                                .join(' · '),
                            style: text.bodyMedium
                                ?.copyWith(color: Colors.white.withValues(alpha: 0.9))),
                      ],
                    ),
                  ),
                  Positioned(
                    right: 12,
                    top: 12,
                    child: Container(
                      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
                      decoration: BoxDecoration(
                        color: Colors.white,
                        borderRadius: BorderRadius.circular(999),
                      ),
                      child: Row(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          const Icon(Icons.star, size: 16, color: SpottoColors.sun),
                          const SizedBox(width: 4),
                          Text('${place.rating}', style: text.labelLarge),
                          Text(' (${place.reviews})', style: text.labelSmall),
                        ],
                      ),
                    ),
                  ),
                ],
              ),
            ),
          ),
        ),
        // Uploaded gallery photos + a tappable promo-video thumbnail.
        if (place.gallery.isNotEmpty || place.promoVideo.isNotEmpty)
          SizedBox(
            height: 88,
            child: ListView(
              scrollDirection: Axis.horizontal,
              padding: const EdgeInsets.fromLTRB(16, 12, 16, 0),
              children: [
                if (place.promoVideo.isNotEmpty)
                  Padding(
                    padding: const EdgeInsets.only(right: 8),
                    child: PressScale(
                      onTap: () => showDialog<void>(
                        context: context,
                        builder: (_) => _PromoVideoDialog(url: resolveMediaUrl(place.promoVideo)),
                      ),
                      child: Container(
                        width: 118,
                        height: 88,
                        decoration: BoxDecoration(
                          color: Colors.black87,
                          borderRadius: BorderRadius.circular(12),
                        ),
                        child: const Center(
                          child: Icon(Icons.play_circle_fill, color: Colors.white, size: 34),
                        ),
                      ),
                    ),
                  ),
                for (final url in place.gallery)
                  Padding(
                    padding: const EdgeInsets.only(right: 8),
                    child: ClipRRect(
                      borderRadius: BorderRadius.circular(12),
                      child: Image.network(resolveMediaUrl(url),
                          width: 118,
                          height: 88,
                          fit: BoxFit.cover,
                          errorBuilder: (_, __, ___) => const SizedBox(width: 118, height: 88)),
                    ),
                  ),
              ],
            ),
          ),
        Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              // stat chips — WiFi only means something once a vendor has
              // actually entered it (score 0 is the "never set" default,
              // meaningless for e.g. a public landmark), so it's hidden
              // rather than showing a misleading "0.0".
              Row(
                children: [
                  if (place.wifi.score > 0) ...[
                    Expanded(
                      child: StatChip(
                          icon: Icons.wifi,
                          value: '${place.wifi.score}',
                          label: '${place.wifi.speed} · ${place.wifi.label}'),
                    ),
                    const SizedBox(width: 10),
                  ],
                  Expanded(
                    child: _HoursStat(hours: place.hours, openNow: place.openNow),
                  ),
                ],
              ),
              const SizedBox(height: 14),

              // full street address, tappable to the same maps query used
              // by the Directions button below.
              if (place.address.isNotEmpty) ...[
                PressScale(
                  scale: 0.98,
                  onTap: () => launchUrl(
                    Uri.parse(
                        'https://www.google.com/maps/search/?api=1&query=${_mapsQuery(place)}'),
                    mode: LaunchMode.externalApplication,
                  ),
                  child: Row(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      const Icon(Icons.place_outlined, size: 16, color: SpottoColors.muted),
                      const SizedBox(width: 6),
                      Expanded(
                        child: Text(place.address,
                            style: text.bodySmall?.copyWith(color: SpottoColors.muted)),
                      ),
                    ],
                  ),
                ),
                const SizedBox(height: 10),
              ],

              // wheelchair accessibility, when known from live Places details
              if (wheelchairAccessible != null) ...[
                Row(
                  children: [
                    Icon(
                      wheelchairAccessible ? Icons.accessible : Icons.not_accessible,
                      size: 16,
                      color: SpottoColors.muted,
                    ),
                    const SizedBox(width: 6),
                    Text(
                      wheelchairAccessible ? 'Wheelchair accessible' : 'Not wheelchair accessible',
                      style: text.bodySmall?.copyWith(color: SpottoColors.muted),
                    ),
                  ],
                ),
                const SizedBox(height: 10),
              ],

              // deal claim
              if (place.deal.active) ...[
                _DealCard(place: place, claimedCode: claimedCode),
                const SizedBox(height: 14),
              ],

              // vibe check
              _VibeCheck(placeId: place.id),
              const SizedBox(height: 14),

              // artisan quest check-in
              if (place.artisan) ...[
                _CheckInButton(place: place),
                const SizedBox(height: 14),
              ],

              // tags
              Wrap(
                spacing: 8,
                runSpacing: 8,
                children: [for (final t in place.tags) Pill(label: t)],
              ),
              const SizedBox(height: 14),
              Text(place.blurb, style: text.bodyLarge),

              // traveller-contributed photos
              const SizedBox(height: 18),
              _TravellerPhotos(place: place),

              // menu
              if (place.menu.isNotEmpty) ...[
                const SizedBox(height: 18),
                Text('Menu', style: text.titleMedium),
                const SizedBox(height: 8),
                for (final section in place.menu) ...[
                  Text(section.name,
                      style: text.labelMedium?.copyWith(color: SpottoColors.accent)),
                  const SizedBox(height: 6),
                  for (final item in section.items)
                    Padding(
                      padding: const EdgeInsets.only(bottom: 10),
                      child: Row(
                        children: [
                          ClipRRect(
                            borderRadius: BorderRadius.circular(12),
                            child: item.img.isNotEmpty
                                ? Image.network(resolveMediaUrl(item.img),
                                    width: 40,
                                    height: 40,
                                    fit: BoxFit.cover,
                                    errorBuilder: (_, __, ___) => _MenuIconTile(icon: item.icon))
                                : _MenuIconTile(icon: item.icon),
                          ),
                          const SizedBox(width: 12),
                          Expanded(
                            child: Column(
                              crossAxisAlignment: CrossAxisAlignment.start,
                              children: [
                                Text(item.name, style: text.bodyLarge),
                                Text(item.desc, style: text.bodySmall),
                              ],
                            ),
                          ),
                          Text('Rs ${item.price}', style: text.labelLarge),
                        ],
                      ),
                    ),
                  const SizedBox(height: 6),
                ],
              ],

              const SizedBox(height: 18),
              _ReviewsSection(place: place),

              if (!contact.isEmpty) ...[
                const SizedBox(height: 18),
                _ContactLinks(contact: contact),
              ],

              // bottom actions
              const SizedBox(height: 18),
              Row(
                children: [
                  Expanded(
                    child: FilledButton.icon(
                      style: FilledButton.styleFrom(
                        backgroundColor: SpottoColors.ink,
                        shape: RoundedRectangleBorder(
                            borderRadius: BorderRadius.circular(16)),
                      ),
                      onPressed: () => launchUrl(
                        Uri.parse(
                          'https://www.google.com/maps/search/?api=1&query=${_mapsQuery(place)}',
                        ),
                        mode: LaunchMode.externalApplication,
                      ),
                      icon: const Icon(Icons.near_me, size: 17),
                      label: const Text('Directions'),
                    ),
                  ),
                  const SizedBox(width: 8),
                  FilledButton.icon(
                    style: FilledButton.styleFrom(
                      backgroundColor: SpottoColors.accent,
                      shape: RoundedRectangleBorder(
                          borderRadius: BorderRadius.circular(16)),
                    ),
                    onPressed: () {
                      final trip = ref.read(tripProvider);
                      showDialog(
                        context: context,
                        builder: (ctx) => AlertDialog(
                          shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(20)),
                          title: Text('Add to ${trip.name}'),
                          content: Column(
                            mainAxisSize: MainAxisSize.min,
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              Text('Select a day for ${place.name}:'),
                              const SizedBox(height: 12),
                              for (var i = 0; i < trip.days.length; i++)
                                ListTile(
                                  dense: true,
                                  leading: const Icon(Icons.calendar_today_rounded, color: SpottoColors.accent),
                                  title: Text('Day ${i + 1} (${trip.days[i].stops.length} stops)'),
                                  onTap: () {
                                    ref.read(tripProvider.notifier).addPlaceStop(i, place);
                                    Navigator.pop(ctx);
                                    ScaffoldMessenger.of(context).showSnackBar(
                                      SnackBar(content: Text('Added ${place.name} to Day ${i + 1} of your trip!')),
                                    );
                                  },
                                ),
                            ],
                          ),
                        ),
                      );
                    },
                    icon: const Icon(Icons.calendar_today_rounded, size: 17),
                    label: const Text('Add to Trip'),
                  ),
                  const SizedBox(width: 8),
                  PressScale(
                    scale: 0.95,
                    onTap: () {
                      ref.read(favoritesProvider.notifier).toggle(place.id);
                      ScaffoldMessenger.of(context).showSnackBar(
                        SnackBar(
                          content: Text(isFavorite ? 'Removed from bookmarks' : 'Saved to bookmarks'),
                        ),
                      );
                    },
                    child: Container(
                      width: 48,
                      height: 48,
                      decoration: BoxDecoration(
                        color: isFavorite ? SpottoColors.accentTint : SpottoColors.sand,
                        borderRadius: BorderRadius.circular(16),
                      ),
                      child: Icon(isFavorite ? Icons.bookmark : Icons.bookmark_outline,
                          size: 19, color: SpottoColors.ink),
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 24),
            ],
          ),
        ),
      ],
    );
  }
}

/// One parsed "Day: range" entry from a Place's `hours` string, e.g.
/// `"Monday: Open 24 hours, Tuesday: Open 24 hours, ..."`.
class _DayHours {
  final String day;
  final String range;
  const _DayHours(this.day, this.range);
}

const _weekdayNames = [
  'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday', 'Sunday'
];

/// Parses the comma-separated `"Day: range"` hours string. Returns an empty
/// list (never throws) for any shape it doesn't recognize, so the caller
/// can fall back to showing the raw string.
List<_DayHours> _parseHours(String raw) {
  final out = <_DayHours>[];
  for (final part in raw.split(', ')) {
    final idx = part.indexOf(': ');
    if (idx == -1) continue;
    out.add(_DayHours(part.substring(0, idx).trim(), part.substring(idx + 2).trim()));
  }
  return out;
}

/// The Open/Closed stat chip, collapsed to a single line instead of dumping
/// all 7 days — "Open 24 hours every day" when every day matches, otherwise
/// today's range, with the full week available on tap only when the days
/// actually differ (nothing to expand when they're all the same).
class _HoursStat extends StatefulWidget {
  final String hours;
  final bool openNow;
  const _HoursStat({required this.hours, required this.openNow});

  @override
  State<_HoursStat> createState() => _HoursStatState();
}

class _HoursStatState extends State<_HoursStat> {
  bool _expanded = false;

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    final days = _parseHours(widget.hours);
    final ranges = days.map((d) => d.range).toSet();
    final expandable = ranges.length > 1;

    String summary;
    if (days.isEmpty) {
      summary = widget.hours;
    } else if (!expandable) {
      summary = ranges.first;
    } else {
      final todayName = _weekdayNames[(DateTime.now().weekday - 1) % 7];
      final today = days.firstWhere((d) => d.day == todayName, orElse: () => days.first);
      summary = 'Today: ${today.range}';
    }

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        PressScale(
          onTap: expandable ? () => setState(() => _expanded = !_expanded) : null,
          child: StatChip(
            icon: Icons.schedule,
            value: widget.openNow ? 'Open' : 'Closed',
            label: summary,
            iconColor: widget.openNow ? SpottoColors.accent2 : SpottoColors.muted,
          ),
        ),
        if (expandable) ...[
          const SizedBox(height: 4),
          Center(
            child: Text(
              _expanded ? 'Hide full week ▴' : 'Full week ▾',
              style: text.labelSmall?.copyWith(color: SpottoColors.muted),
            ),
          ),
        ],
        if (_expanded)
          Padding(
            padding: const EdgeInsets.only(top: 8),
            child: Column(
              children: [
                for (final d in days)
                  Padding(
                    padding: const EdgeInsets.symmetric(vertical: 2),
                    child: Row(
                      mainAxisAlignment: MainAxisAlignment.spaceBetween,
                      children: [
                        Text(d.day, style: text.bodySmall),
                        Text(d.range, style: text.bodySmall?.copyWith(color: SpottoColors.muted)),
                      ],
                    ),
                  ),
              ],
            ),
          ),
      ],
    );
  }
}

class _VibeCheck extends ConsumerStatefulWidget {
  final String placeId;
  const _VibeCheck({required this.placeId});

  @override
  ConsumerState<_VibeCheck> createState() => _VibeCheckState();
}

class _VibeCheckState extends ConsumerState<_VibeCheck> {
  String? _justVoted;

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    final votes = ref.watch(vibeVotesProvider)[widget.placeId] ?? const <String, int>{};
    final total = data.vibes.fold(0, (a, v) => a + (votes[v.id] ?? 0));
    final lead = [...data.vibes]
      ..sort((a, b) => (votes[b.id] ?? 0).compareTo(votes[a.id] ?? 0));

    return Container(
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: SpottoColors.sand,
        borderRadius: BorderRadius.circular(16),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              const Icon(Icons.monitor_heart_outlined, size: 14, color: SpottoColors.muted),
              const SizedBox(width: 5),
              Text('CURRENT VIBE',
                  style: text.labelSmall
                      ?.copyWith(color: SpottoColors.muted, letterSpacing: 0.6)),
              const Spacer(),
              if (total > 0)
                Row(
                  children: [
                    Icon(spottoIcon(lead.first.icon),
                        size: 13, color: Color(lead.first.color)),
                    const SizedBox(width: 4),
                    Text('Mostly ${lead.first.label.toLowerCase()} · $total votes',
                        style: text.labelSmall?.copyWith(color: Color(lead.first.color))),
                  ],
                ),
            ],
          ),
          const SizedBox(height: 10),
          Row(
            children: [
              for (final (i, v) in data.vibes.indexed) ...[
                if (i > 0) const SizedBox(width: 8),
                Expanded(
                  child: PressScale(
                    scale: 0.95,
                    onTap: () {
                      ref.read(vibeVotesProvider.notifier).vote(widget.placeId, v.id);
                      setState(() => _justVoted = v.id);
                    },
                    child: _vibeOption(text, v, votes, total),
                  ),
                ),
              ],
            ],
          ),
        ],
      ),
    );
  }

  Widget _vibeOption(TextTheme text, VibeTag v, Map<String, int> votes, int total) {
    final count = votes[v.id] ?? 0;
    final pct = total > 0 ? count / total : 0.0;
    final mine = _justVoted == v.id;
    final color = Color(v.color);

    return Container(
      padding: const EdgeInsets.all(10),
      decoration: BoxDecoration(
        color: mine ? color : Colors.white,
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: mine ? color : SpottoColors.line, width: 1.5),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(spottoIcon(v.icon), size: 18, color: mine ? Colors.white : color),
          const SizedBox(height: 4),
          Text(v.label,
              style: TextStyle(
                  fontSize: 12,
                  fontWeight: FontWeight.w800,
                  color: mine ? Colors.white : SpottoColors.ink)),
          const SizedBox(height: 6),
          ClipRRect(
            borderRadius: BorderRadius.circular(999),
            child: Container(
              height: 6,
              color: mine
                  ? Colors.white.withValues(alpha: 0.3)
                  : SpottoColors.ink.withValues(alpha: 0.08),
              child: AnimatedFractionallySizedBox(
                duration: const Duration(milliseconds: 500),
                alignment: Alignment.centerLeft,
                widthFactor: pct,
                child: Container(color: mine ? Colors.white : color),
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class _CheckInButton extends ConsumerWidget {
  final Place place;
  const _CheckInButton({required this.place});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final didCheckIn = ref.watch(checkInsProvider).contains(place.id);

    return PressScale(
      scale: 0.98,
      onTap: didCheckIn
          ? null
          : () => ref.read(checkInsProvider.notifier).checkIn(place.id),
      child: Container(
        width: double.infinity,
        padding: const EdgeInsets.symmetric(vertical: 12),
        decoration: BoxDecoration(
          color: didCheckIn ? SpottoColors.accentTint : SpottoColors.sun,
          borderRadius: BorderRadius.circular(16),
        ),
        child: Row(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Icon(didCheckIn ? Icons.verified_outlined : Icons.explore,
                size: 17,
                color: didCheckIn ? SpottoColors.accent : const Color(0xFF5A3B00)),
            const SizedBox(width: 6),
            Text(
              didCheckIn
                  ? 'Checked in — quest progress saved'
                  : 'Check in for Daily Quest',
              style: TextStyle(
                fontSize: 14,
                fontWeight: FontWeight.w800,
                color: didCheckIn ? SpottoColors.accent : const Color(0xFF5A3B00),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _DealCard extends ConsumerStatefulWidget {
  final Place place;
  final String? claimedCode;
  const _DealCard({required this.place, required this.claimedCode});

  @override
  ConsumerState<_DealCard> createState() => _DealCardState();
}

class _DealCardState extends ConsumerState<_DealCard> {
  @override
  void initState() {
    super.initState();
    // Real traveller impression, once per time this card is actually shown
    // (initState runs once per widget instance, not on every rebuild) —
    // lets the vendor portal show a genuine "Views" count per offer.
    final offerId = widget.place.deal.id;
    if (offerId != null) _trackView(offerId);
  }

  Future<void> _trackView(String offerId) async {
    try {
      await ref.read(apiProvider).dio.post('/offers/$offerId/view');
    } catch (_) {
      // Fire-and-forget — a failed view ping shouldn't affect the UI.
    }
  }

  @override
  Widget build(BuildContext context) {
    final place = widget.place;
    final claimedCode = widget.claimedCode;
    final text = Theme.of(context).textTheme;
    final claimed = claimedCode != null;

    return AnimatedContainer(
      duration: const Duration(milliseconds: 250),
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: claimed ? SpottoColors.ink : const Color(0xFFFDF3D7),
        borderRadius: BorderRadius.circular(18),
        border: Border.all(
            color: claimed ? SpottoColors.ink : const Color(0xFFF2DCA0), width: 1.5),
      ),
      child: claimed
          ? Row(
              crossAxisAlignment: CrossAxisAlignment.center,
              children: [
                // White backing is required for the QR to actually scan —
                // this card's background is SpottoColors.ink (dark), and a
                // QR rendered directly on a dark background has no contrast.
                Container(
                  padding: const EdgeInsets.all(6),
                  decoration: BoxDecoration(
                    color: Colors.white,
                    borderRadius: BorderRadius.circular(10),
                  ),
                  child: QrImageView(data: claimedCode, size: 64, gapless: false),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Text('Show at counter',
                          style: text.labelMedium
                              ?.copyWith(color: SpottoColors.paleTeal)),
                      const SizedBox(height: 4),
                      Text(claimedCode,
                          style: text.headlineSmall
                              ?.copyWith(color: Colors.white, letterSpacing: 2)),
                      const SizedBox(height: 4),
                      Text(place.deal.label ?? '',
                          style: text.bodySmall
                              ?.copyWith(color: SpottoColors.paleTeal)),
                    ],
                  ),
                ),
              ],
            )
          : Row(
              children: [
                Container(
                  padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
                  decoration: BoxDecoration(
                    color: SpottoColors.sun,
                    borderRadius: BorderRadius.circular(8),
                  ),
                  child: Text(place.deal.tag ?? 'Deal',
                      style: text.labelSmall?.copyWith(color: SpottoColors.ink)),
                ),
                const SizedBox(width: 10),
                Expanded(child: Text(place.deal.label ?? '', style: text.bodyMedium)),
                FilledButton(
                  style: FilledButton.styleFrom(
                    backgroundColor: SpottoColors.sun,
                    foregroundColor: SpottoColors.ink,
                    padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
                  ),
                  onPressed: () async {
                    final messenger = ScaffoldMessenger.of(context);
                    try {
                      await ref.read(claimsProvider.notifier).claim(
                            place.id,
                            offerId: place.deal.id,
                            offerLabel: place.deal.label ?? '',
                            offerValue: place.deal.tag ?? '',
                          );
                    } catch (e) {
                      messenger.showSnackBar(
                          SnackBar(content: Text(ApiClient.errorMessage(e))));
                    }
                  },
                  child: const Text('Claim'),
                ),
              ],
            ),
    );
  }
}

/// Tappable phone/website/social row — only the fields the vendor actually
/// filled in are shown, so this renders nothing for the many businesses
/// that haven't set any contact info yet (see !place.contact.isEmpty guard
/// at the call site).
class _ContactLinks extends StatelessWidget {
  final ContactInfo contact;
  const _ContactLinks({required this.contact});

  Future<void> _open(String url) =>
      launchUrl(Uri.parse(url), mode: LaunchMode.externalApplication);

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    final chips = <Widget>[];

    void addChip(String? value, IconData icon, String label, String Function(String) toUrl) {
      if (value == null || value.isEmpty) return;
      chips.add(PressScale(
        scale: 0.96,
        onTap: () => _open(toUrl(value)),
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
          decoration: BoxDecoration(
            color: SpottoColors.sand,
            borderRadius: BorderRadius.circular(14),
          ),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(icon, size: 16, color: SpottoColors.ink),
              const SizedBox(width: 6),
              Text(label, style: text.labelMedium?.copyWith(color: SpottoColors.ink)),
            ],
          ),
        ),
      ));
    }

    addChip(contact.phone, Icons.call_outlined, 'Call', (v) => 'tel:$v');
    addChip(contact.website, Icons.language, 'Website',
        (v) => v.startsWith('http') ? v : 'https://$v');
    addChip(contact.instagram, Icons.camera_alt_outlined, 'Instagram',
        (v) => v.startsWith('http') ? v : 'https://instagram.com/${v.replaceAll('@', '')}');
    addChip(contact.facebook, Icons.groups_outlined, 'Facebook',
        (v) => v.startsWith('http') ? v : 'https://facebook.com/$v');

    if (chips.isEmpty) return const SizedBox.shrink();
    return Wrap(spacing: 8, runSpacing: 8, children: chips);
  }
}

/// Reviews: existing list (rating + text) plus a form to submit or edit
/// your own. Ratings of 1-2 count against the vendor for the admin map's
/// red-flag threshold — this is the real write path behind that feature.
class _ReviewsSection extends ConsumerStatefulWidget {
  final Place place;
  const _ReviewsSection({required this.place});

  @override
  ConsumerState<_ReviewsSection> createState() => _ReviewsSectionState();
}

class _ReviewsSectionState extends ConsumerState<_ReviewsSection> {
  int _draftRating = 0;
  final _textController = TextEditingController();
  bool _submitting = false;
  final List<_PickedImage> _photos = [];

  @override
  void dispose() {
    _textController.dispose();
    super.dispose();
  }

  Future<void> _pickPhoto() async {
    final f = await ImagePicker().pickImage(
        source: ImageSource.gallery, maxWidth: 1600, imageQuality: 85);
    if (f == null) return;
    final contentType = mimeForUpload(f.mimeType, f.name);
    if (contentType == null) {
      if (mounted) {
        ScaffoldMessenger.of(context)
            .showSnackBar(const SnackBar(content: Text('Unsupported image type.')));
      }
      return;
    }
    final bytes = await f.readAsBytes();
    setState(() => _photos.add(_PickedImage(bytes, contentType)));
  }

  Future<void> _submit() async {
    if (_draftRating == 0 || _submitting) return;
    setState(() => _submitting = true);
    final messenger = ScaffoldMessenger.of(context);
    try {
      final reviewId =
          await submitReview(ref, widget.place.id, _draftRating, _textController.text.trim());
      // Attach any picked photos to the review just created/edited.
      for (final photo in _photos) {
        await ref.read(uploadServiceProvider).upload(
              bytes: photo.bytes,
              contentType: photo.contentType,
              ownerType: 'review',
              ownerId: reviewId,
              role: 'review',
            );
      }
      if (_photos.isNotEmpty) ref.invalidate(placeReviewsProvider(widget.place.id));
      _textController.clear();
      setState(() {
        _draftRating = 0;
        _photos.clear();
      });
    } catch (e) {
      messenger.showSnackBar(SnackBar(content: Text(ApiClient.errorMessage(e))));
    } finally {
      if (mounted) setState(() => _submitting = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    final reviewsAsync = ref.watch(placeReviewsProvider(widget.place.id));

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text('Reviews', style: text.titleMedium),
        const SizedBox(height: 10),

        // write/edit form
        Container(
          padding: const EdgeInsets.all(14),
          decoration: BoxDecoration(
            color: SpottoColors.sand,
            borderRadius: BorderRadius.circular(16),
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                children: [
                  for (var i = 1; i <= 5; i++)
                    PressScale(
                      scale: 0.85,
                      onTap: () => setState(() => _draftRating = i),
                      child: Padding(
                        padding: const EdgeInsets.only(right: 4),
                        child: Icon(
                          i <= _draftRating ? Icons.star : Icons.star_border,
                          color: SpottoColors.sun,
                          size: 26,
                        ),
                      ),
                    ),
                ],
              ),
              const SizedBox(height: 10),
              TextField(
                controller: _textController,
                maxLines: 2,
                decoration: const InputDecoration(
                  hintText: 'Share how it went (optional)',
                  filled: true,
                  fillColor: Colors.white,
                ),
              ),
              const SizedBox(height: 10),
              SizedBox(
                height: 56,
                child: ListView(
                  scrollDirection: Axis.horizontal,
                  children: [
                    PressScale(
                      onTap: _pickPhoto,
                      child: Container(
                        width: 56,
                        height: 56,
                        margin: const EdgeInsets.only(right: 8),
                        decoration: BoxDecoration(
                          color: Colors.white,
                          borderRadius: BorderRadius.circular(12),
                          border: Border.all(color: SpottoColors.accentTint, width: 1.5),
                        ),
                        child: const Icon(Icons.add_a_photo_outlined,
                            color: SpottoColors.accent, size: 20),
                      ),
                    ),
                    for (var i = 0; i < _photos.length; i++)
                      Padding(
                        padding: const EdgeInsets.only(right: 8),
                        child: Stack(
                          children: [
                            ClipRRect(
                              borderRadius: BorderRadius.circular(12),
                              child: Image.memory(_photos[i].bytes,
                                  width: 56, height: 56, fit: BoxFit.cover),
                            ),
                            Positioned(
                              right: 0,
                              top: 0,
                              child: PressScale(
                                onTap: () => setState(() => _photos.removeAt(i)),
                                child: Container(
                                  decoration: const BoxDecoration(
                                      color: Colors.black54, shape: BoxShape.circle),
                                  padding: const EdgeInsets.all(2),
                                  child: const Icon(Icons.close, size: 12, color: Colors.white),
                                ),
                              ),
                            ),
                          ],
                        ),
                      ),
                  ],
                ),
              ),
              const SizedBox(height: 10),
              SizedBox(
                width: double.infinity,
                child: FilledButton(
                  onPressed: _draftRating == 0 || _submitting ? null : _submit,
                  child: Text(_submitting ? 'Posting…' : 'Post review'),
                ),
              ),
            ],
          ),
        ),
        const SizedBox(height: 12),

        reviewsAsync.when(
          data: (reviews) {
            if (reviews.isEmpty) {
              return Text('No reviews yet — be the first.',
                  style: text.bodySmall?.copyWith(color: SpottoColors.muted));
            }
            return Column(
              children: [
                for (final r in reviews) ...[
                  _ReviewTile(review: r),
                  const SizedBox(height: 8),
                ],
              ],
            );
          },
          loading: () => const Padding(
            padding: EdgeInsets.symmetric(vertical: 12),
            child: Center(child: CircularProgressIndicator(strokeWidth: 2)),
          ),
          error: (_, __) => Text('Couldn\'t load reviews.',
              style: text.bodySmall?.copyWith(color: SpottoColors.muted)),
        ),
      ],
    );
  }
}

class _ReviewTile extends StatelessWidget {
  final Review review;
  const _ReviewTile({required this.review});

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.circular(14),
        border: Border.all(color: SpottoColors.ink.withValues(alpha: 0.06)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Row(
                children: [
                  for (var i = 1; i <= 5; i++)
                    Icon(
                      i <= review.rating ? Icons.star : Icons.star_border,
                      color: SpottoColors.sun,
                      size: 15,
                    ),
                ],
              ),
              if (review.travellerName != null && review.travellerName!.isNotEmpty) ...[
                const SizedBox(width: 8),
                Text(review.travellerName!,
                    style: text.labelSmall?.copyWith(fontWeight: FontWeight.w800)),
              ],
            ],
          ),
          if (review.text.isNotEmpty) ...[
            const SizedBox(height: 6),
            Text(review.text, style: text.bodySmall),
          ],
          if (review.photos.isNotEmpty) ...[
            const SizedBox(height: 8),
            SizedBox(
              height: 64,
              child: ListView(
                scrollDirection: Axis.horizontal,
                children: [
                  for (final url in review.photos)
                    Padding(
                      padding: const EdgeInsets.only(right: 8),
                      child: ClipRRect(
                        borderRadius: BorderRadius.circular(10),
                        child: Image.network(resolveMediaUrl(url),
                            width: 64,
                            height: 64,
                            fit: BoxFit.cover,
                            errorBuilder: (_, __, ___) => const SizedBox(width: 64, height: 64)),
                      ),
                    ),
                ],
              ),
            ),
          ],
        ],
      ),
    );
  }
}

/// Traveller-contributed photos for a spot: a horizontal strip of uploaded
/// photos plus an "Add" tile that picks and uploads one to R2 (role contrib).
class _TravellerPhotos extends ConsumerStatefulWidget {
  final Place place;
  const _TravellerPhotos({required this.place});

  @override
  ConsumerState<_TravellerPhotos> createState() => _TravellerPhotosState();
}

class _TravellerPhotosState extends ConsumerState<_TravellerPhotos> {
  bool _uploading = false;

  Future<void> _add() async {
    if (_uploading) return;
    final XFile? file = await ImagePicker().pickImage(
      source: ImageSource.gallery,
      maxWidth: 1600,
      imageQuality: 85,
    );
    if (file == null) return;
    final contentType = mimeForUpload(file.mimeType, file.name);
    if (contentType == null) {
      _snack('Unsupported image type.');
      return;
    }
    setState(() => _uploading = true);
    try {
      final bytes = await file.readAsBytes();
      await ref.read(uploadServiceProvider).upload(
            bytes: bytes,
            contentType: contentType,
            ownerType: 'place',
            ownerId: widget.place.id,
            role: 'contrib',
          );
      ref.invalidate(placePhotosProvider(widget.place.id));
    } catch (_) {
      _snack('Upload failed — try again.');
    } finally {
      if (mounted) setState(() => _uploading = false);
    }
  }

  void _snack(String message) {
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(message)));
  }

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    final photos = ref.watch(placePhotosProvider(widget.place.id)).valueOrNull ?? const <String>[];
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text('Traveller photos', style: text.titleMedium),
        const SizedBox(height: 8),
        SizedBox(
          height: 84,
          child: ListView(
            scrollDirection: Axis.horizontal,
            children: [
              PressScale(
                onTap: _add,
                child: Container(
                  width: 84,
                  height: 84,
                  margin: const EdgeInsets.only(right: 8),
                  decoration: BoxDecoration(
                    color: SpottoColors.accentTint,
                    borderRadius: BorderRadius.circular(14),
                  ),
                  child: Center(
                    child: _uploading
                        ? const SizedBox(
                            width: 20,
                            height: 20,
                            child: CircularProgressIndicator(
                                strokeWidth: 2.5, color: SpottoColors.accent),
                          )
                        : const Icon(Icons.add_a_photo_outlined,
                            color: SpottoColors.accent, size: 24),
                  ),
                ),
              ),
              for (final url in photos)
                Padding(
                  padding: const EdgeInsets.only(right: 8),
                  child: ClipRRect(
                    borderRadius: BorderRadius.circular(14),
                    child: Image.network(resolveMediaUrl(url),
                        width: 84,
                        height: 84,
                        fit: BoxFit.cover,
                        errorBuilder: (_, __, ___) => const SizedBox(width: 84, height: 84)),
                  ),
                ),
            ],
          ),
        ),
      ],
    );
  }
}

/// Fallback icon tile shown for a menu item that has no uploaded photo.
class _MenuIconTile extends StatelessWidget {
  final String icon;
  const _MenuIconTile({required this.icon});

  @override
  Widget build(BuildContext context) {
    return Container(
      width: 40,
      height: 40,
      decoration: BoxDecoration(
        color: SpottoColors.accentTint,
        borderRadius: BorderRadius.circular(12),
      ),
      child: Icon(spottoIcon(icon), size: 20, color: SpottoColors.accent),
    );
  }
}

/// A photo the traveller picked for a review, held in memory (with its MIME
/// type) until the review is posted and the bytes are uploaded to R2.
class _PickedImage {
  final Uint8List bytes;
  final String contentType;
  const _PickedImage(this.bytes, this.contentType);
}

/// Plays a vendor's promo video (an R2 URL) in-app in a dialog.
class _PromoVideoDialog extends StatefulWidget {
  final String url;
  const _PromoVideoDialog({required this.url});

  @override
  State<_PromoVideoDialog> createState() => _PromoVideoDialogState();
}

class _PromoVideoDialogState extends State<_PromoVideoDialog> {
  late final VideoPlayerController _controller;
  bool _ready = false;
  bool _failed = false;

  @override
  void initState() {
    super.initState();
    _controller = VideoPlayerController.networkUrl(Uri.parse(widget.url))
      ..initialize().then((_) {
        if (!mounted) return;
        setState(() => _ready = true);
        _controller.play();
      }).catchError((_) {
        if (mounted) setState(() => _failed = true);
      });
    _controller.addListener(() {
      if (mounted) setState(() {});
    });
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Dialog(
      backgroundColor: Colors.black,
      insetPadding: const EdgeInsets.all(16),
      child: Stack(
        alignment: Alignment.center,
        children: [
          if (_failed)
            const Padding(
              padding: EdgeInsets.all(40),
              child: Text("Couldn't play this video.",
                  style: TextStyle(color: Colors.white)),
            )
          else if (_ready)
            AspectRatio(
              aspectRatio: _controller.value.aspectRatio == 0
                  ? 16 / 9
                  : _controller.value.aspectRatio,
              child: GestureDetector(
                onTap: () => setState(() =>
                    _controller.value.isPlaying ? _controller.pause() : _controller.play()),
                child: Stack(
                  alignment: Alignment.center,
                  children: [
                    VideoPlayer(_controller),
                    if (!_controller.value.isPlaying)
                      const Icon(Icons.play_circle_fill, color: Colors.white70, size: 56),
                  ],
                ),
              ),
            )
          else
            const Padding(
              padding: EdgeInsets.all(40),
              child: CircularProgressIndicator(color: Colors.white),
            ),
          Positioned(
            right: 4,
            top: 4,
            child: IconButton(
              icon: const Icon(Icons.close, color: Colors.white),
              onPressed: () => Navigator.of(context).pop(),
            ),
          ),
        ],
      ),
    );
  }
}
