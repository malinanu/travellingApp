// Spot detail bottom sheet — port of BusinessDrawer.jsx: illustration
// header, stats, deal claim, vibe voting, artisan check-in, menu, actions.

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:geolocator/geolocator.dart';
import 'package:image_picker/image_picker.dart';
import 'package:qr_flutter/qr_flutter.dart';
import 'package:url_launcher/url_launcher.dart';
import 'package:video_player/video_player.dart';

import '../../../core/api/api_client.dart';
import '../../../core/api/upload_service.dart';
import '../../../core/theme/palette.dart';
import '../../../core/utils/debounced_saver.dart';
import '../../../core/widgets/illustration.dart';
import '../../../core/widgets/pill.dart';
import '../../../core/widgets/press_scale.dart';
import '../../../core/widgets/progressive_image.dart';
import '../../../core/widgets/sheet_scaffold.dart';
import '../../../core/widgets/spotto_icons.dart';
import '../../../core/widgets/stat_chip.dart';
import '../../../data/mock/places.dart' as data;
import '../../../data/models/models.dart';
import '../../../data/repositories/repositories.dart';
import '../../../data/vendor_categories.dart';

/// Shared cap for both the review-photo picker and the traveller-photo
/// picker (see _ReviewsSectionState._pickPhoto and
/// _TravellerPhotosState._add) — kept as one constant so the two limits
/// can't silently drift apart.
const kMaxPhotosPerUpload = 5;

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
    final details = ref.watch(placeDetailsProvider(place.id)).value;
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
                  // fall back to the illustration. Same thumb-only tier as
                  // the spots-list card — there's no viewer for covers, so
                  // the ~320px thumbnail is plenty even at this larger size.
                  if (place.cover.isNotEmpty)
                    ProgressiveImage(
                      url: resolveMediaUrl(place.coverThumb.isNotEmpty ? place.coverThumb : place.cover),
                      microThumb: place.coverMicro,
                      fit: BoxFit.cover,
                      errorBuilder: (_) => Illustration(kind: place.photo),
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
                for (final (i, url) in place.gallery.indexed)
                  Padding(
                    padding: const EdgeInsets.only(right: 8),
                    // Thumb-tier for the tile itself (no blur placeholder —
                    // a lower-traffic 88px strip); tapping still escalates
                    // to the real full-resolution images in the viewer.
                    child: PressScale(
                      onTap: () => showDialog<void>(
                        context: context,
                        builder: (_) => _PhotoViewerDialog(
                          urls: [for (final g in place.gallery) resolveMediaUrl(g)],
                          initialIndex: i,
                        ),
                      ),
                      child: ProgressiveImage(
                        url: resolveMediaUrl(
                            i < place.galleryThumbs.length && place.galleryThumbs[i].isNotEmpty
                                ? place.galleryThumbs[i]
                                : url),
                        width: 118,
                        height: 88,
                        fit: BoxFit.cover,
                        borderRadius: BorderRadius.circular(12),
                        errorBuilder: (_) => const SizedBox(width: 118, height: 88),
                      ),
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

              // real booking (session/table/stay — see placeServicesProvider)
              _BookingSection(place: place),

              // vibe check
              _VibeCheck(place: place),
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

/// Matches the server's own enforcement (vibeCheckinGeofenceMeters in
/// traveller.go) — this copy is only for UI gating (show/hide the tappable
/// bars); the server is what actually enforces it on submit.
const kVibeCheckinGeofenceMeters = 150.0;

/// data.vibes filtered to the dimensions meaningful for a place's category
/// (see relevantVibeDimensions) — e.g. a beach bar shows Loud + Crowded
/// only, no meaningless Productive slider.
List<VibeTag> _relevantVibes(String categoryId) {
  final relevant = relevantVibeDimensions(categoryId);
  return [for (final v in data.vibes) if (relevant.contains(v.id)) v];
}

/// Reads one dimension's value out of a VibeAggregate/VibeCheckin by the
/// shared vibe id (see data.vibes) — keeps _VibeCheck's build method from
/// hardcoding a productive/loud/crowded switch three times over.
double _vibeAverageFor(VibeAggregate v, String id) => switch (id) {
      'productive' => v.productive,
      'loud' => v.loud,
      'crowded' => v.crowded,
      _ => 0,
    };
int? _vibeMineFor(VibeCheckin? v, String id) => switch (id) {
      'productive' => v?.productive,
      'loud' => v?.loud,
      'crowded' => v?.crowded,
      _ => null,
    };

class _VibeCheck extends ConsumerStatefulWidget {
  final Place place;
  const _VibeCheck({required this.place});

  @override
  ConsumerState<_VibeCheck> createState() => _VibeCheckState();
}

class _VibeCheckState extends ConsumerState<_VibeCheck> {
  // Tapping a vibe segment used to fire an immediate network POST per tap —
  // three dimensions rated in one visit meant three separate round-trips,
  // even though the server's endpoint always accepted a full 3-dimension
  // snapshot in one call. Taps are now purely local (instant, no network,
  // any 1-5 value on any dimension in any order) and only reach the server
  // as one combined request, debounced ~1s after the last tap and always
  // forced on dispose() (the card closing) — see _setDraft/_flush below.
  int? _draftProductive, _draftLoud, _draftCrowded;
  bool _dirty = false;
  // Seed the drafts from the provider's last-known `mine` exactly once, on
  // the first build — never again, so a later unrelated provider update
  // (e.g. this same place rated from another screen) can't clobber an
  // in-progress local draft mid-session.
  bool _seeded = false;
  // dispose() can't ref.watch — these remember the latest values a build()
  // saw, for a flush that might not run until the widget is going away.
  LatLngAt? _lastDevicePos;
  VibeAggregate? _lastVibe;
  late final DebouncedSaver _saver;

  @override
  void initState() {
    super.initState();
    _saver = DebouncedSaver(save: _flush);
    // devicePositionProvider is cached for the whole app session, not
    // re-checked on its own — without this, a traveller who was out of
    // range (or had location denied) when they last resolved a position
    // would stay stuck seeing the same stale "too far away"/"enable
    // location" state every time they reopen a place's drawer, even after
    // actually walking into range or granting permission. Re-checking
    // once per drawer open (not on every rebuild — initState runs once
    // per widget instance) keeps that cheap: bestEffortPosition tries the
    // OS's cached last-known position first, which resolves in <5ms.
    Future.microtask(() {
      if (mounted) ref.invalidate(devicePositionProvider);
    });
  }

  @override
  void dispose() {
    if (_dirty) _saver.flush(); // fire-and-forget — widget is going away regardless
    _saver.dispose();
    super.dispose();
  }

  int? _draftFor(String vibeId) => switch (vibeId) {
        'productive' => _draftProductive,
        'loud' => _draftLoud,
        'crowded' => _draftCrowded,
        _ => null,
      };

  void _setDraft(String vibeId, int value) {
    HapticFeedback.selectionClick();
    setState(() {
      switch (vibeId) {
        case 'productive':
          _draftProductive = value;
        case 'loud':
          _draftLoud = value;
        case 'crowded':
          _draftCrowded = value;
      }
      _dirty = true;
    });
    _saver.schedule();
  }

  Future<void> _flush() async {
    if (!_dirty || _lastDevicePos == null) return;
    try {
      await ref.read(vibeCheckInsProvider.notifier).rate(
            widget.place.id,
            productive: _draftProductive ?? 0,
            loud: _draftLoud ?? 0,
            crowded: _draftCrowded ?? 0,
            previousMine: null, // irrelevant — all 3 dimensions are always explicit now
            currentVibe: _lastVibe ?? const VibeAggregate(),
            lat: _lastDevicePos!.lat,
            lng: _lastDevicePos!.lng,
          );
      _dirty = false;
    } catch (e) {
      // Best-effort — matches _DealCardState._trackView's fire-and-forget
      // convention elsewhere in this file. Only surface an error if the
      // card is still open to see it; a dispose()-triggered flush has
      // nothing left to show it to.
      if (mounted) {
        ScaffoldMessenger.of(context)
            .showSnackBar(SnackBar(content: Text(ApiClient.errorMessage(e))));
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;

    // GET /places/{id} is the only response that includes the caller's own
    // latest check-in (myVibe) — the bulk list this sheet was opened from
    // never does, same reasoning as the contact/wheelchair/priceLevel
    // overrides above in BusinessDrawerBody. Falls back to the list-derived
    // place while that detail fetch is still in flight.
    final details = ref.watch(placeDetailsProvider(widget.place.id)).value;
    final serverVibe = details?.vibe ?? widget.place.vibe;
    final serverMine = details?.myVibe ?? widget.place.myVibe;

    // A session override exists once the traveller has actually rated this
    // place here — see VibeCheckInsNotifier's doc comment.
    final override = ref.watch(vibeCheckInsProvider)[widget.place.id];
    final vibe = override?.vibe ?? serverVibe;
    final mine = override?.mine ?? serverMine;
    _lastVibe = vibe;
    if (!_seeded) {
      _draftProductive = _vibeMineFor(mine, 'productive');
      _draftLoud = _vibeMineFor(mine, 'loud');
      _draftCrowded = _vibeMineFor(mine, 'crowded');
      _seeded = true;
    }

    final devicePosAsync = ref.watch(devicePositionProvider);
    final resolved = devicePosAsync.value;
    final devicePos = resolved?.$1;
    final locationResult = resolved?.$2;
    _lastDevicePos = devicePos;
    // devicePositionProvider always eventually resolves to a real position
    // or the Weligama Bay fallback — it never stays null forever — so
    // "still loading" and "resolved but too far away" are two genuinely
    // different states that deserve different copy, not the same disabled
    // segments + "Rate this once you're here." for both.
    final resolvingLocation = devicePosAsync.isLoading && resolved == null;
    // Gate on an actual granted position, not just "resolved" — resolved
    // is also true when permission was denied/location services are off,
    // in which case devicePos is the Weligama Bay fallback, not the
    // traveller's real position, and must never count toward the geofence
    // or be sent as the check-in's lat/lng.
    final granted = locationResult == DeviceLocationResult.granted;
    final locationDenied = !resolvingLocation && !granted;
    final withinGeofence = granted &&
        devicePos != null &&
        Geolocator.distanceBetween(
              devicePos.lat,
              devicePos.lng,
              widget.place.lat,
              widget.place.lng,
            ) <=
            kVibeCheckinGeofenceMeters;

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
              if (vibe.sampleSize > 0)
                Text('${vibe.sampleSize} traveller${vibe.sampleSize == 1 ? '' : 's'}',
                    style: text.labelSmall?.copyWith(color: SpottoColors.muted)),
            ],
          ),
          const SizedBox(height: 12),
          Row(
            children: [
              // Only show dimensions meaningful for this place's category
              // (e.g. "productive" doesn't mean anything for a beach bar) —
              // see relevantVibeDimensions.
              for (final (i, v) in _relevantVibes(widget.place.categoryId).indexed) ...[
                if (i > 0) const SizedBox(width: 8),
                Expanded(
                  child: _VibeCard(
                    label: v.label,
                    icon: spottoIcon(v.icon),
                    color: Color(v.color),
                    average: _vibeAverageFor(vibe, v.id),
                    mine: _draftFor(v.id),
                    enabled: withinGeofence,
                    onTap: !granted ? null : (value) => _setDraft(v.id, value),
                  ),
                ),
              ],
            ],
          ),
          if (!withinGeofence) ...[
            const SizedBox(height: 8),
            if (resolvingLocation)
              Text('Finding your location…',
                  style: text.labelSmall
                      ?.copyWith(color: SpottoColors.muted, fontStyle: FontStyle.italic))
            else if (locationDenied)
              GestureDetector(
                onTap: () async {
                  if (locationResult == DeviceLocationResult.deniedForever) {
                    await Geolocator.openAppSettings();
                  } else {
                    ref.invalidate(devicePositionProvider);
                  }
                },
                child: Text(
                  locationResult == DeviceLocationResult.deniedForever
                      ? 'Location is off — tap to enable it in Settings.'
                      : 'Enable location to rate this spot — tap to try again.',
                  style: text.labelSmall?.copyWith(
                      color: SpottoColors.accent,
                      fontStyle: FontStyle.italic,
                      decoration: TextDecoration.underline),
                ),
              )
            else
              Text("Rate this once you're here.",
                  style: text.labelSmall
                      ?.copyWith(color: SpottoColors.muted, fontStyle: FontStyle.italic)),
          ],
        ],
      ),
    );
  }
}

/// One vibe dimension, styled like the old one-tap vote card (solid-color
/// fill once you've rated it, white icon/border) with a 1-5 check-in scale:
/// a 5-segment tappable strip, each segment one point.
class _VibeCard extends StatelessWidget {
  final String label;
  final IconData icon;
  final Color color;
  final double average; // 0-5
  final int? mine; // 0-5 (this session's local draft), or null if never rated
  final bool enabled;
  final ValueChanged<int>? onTap; // drafted value: 1-5 — see _VibeCheckState._setDraft

  const _VibeCard({
    required this.label,
    required this.icon,
    required this.color,
    required this.average,
    required this.mine,
    required this.enabled,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    final isMine = mine != null;
    // Once you've rated, the bar shows YOUR rating (this session's local
    // draft — see _VibeCheckState), not the crowd average, which is a
    // separately-tracked figure that only reconciles once the debounced/
    // on-close flush actually reaches the server. Driving the segment
    // count from `average` even when isMine was the original bug behind
    // "I tap 6 and it shows something else" — `mine` is always immediate,
    // `average` never is.
    final filled = (isMine ? mine!.toDouble() : average).round().clamp(0, 5);
    final segmentOn = isMine ? Colors.white : color;
    final segmentOff =
        isMine ? Colors.white.withValues(alpha: 0.3) : SpottoColors.ink.withValues(alpha: 0.08);

    return Container(
      padding: const EdgeInsets.all(10),
      decoration: BoxDecoration(
        color: isMine ? color : Colors.white,
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: isMine ? color : SpottoColors.line, width: 1.5),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(icon, size: 18, color: isMine ? Colors.white : color),
          const SizedBox(height: 4),
          Text(label,
              style: TextStyle(
                  fontSize: 12,
                  fontWeight: FontWeight.w800,
                  color: isMine ? Colors.white : SpottoColors.ink)),
          const SizedBox(height: 6),
          Row(
            children: [
              for (var i = 1; i <= 5; i++)
                Expanded(
                  child: Padding(
                    padding: EdgeInsets.only(right: i < 5 ? 2 : 0),
                    child: GestureDetector(
                      onTap: enabled && onTap != null
                          ? () {
                              HapticFeedback.selectionClick();
                              onTap!(i);
                            }
                          : null,
                      child: Container(
                        height: 8,
                        decoration: BoxDecoration(
                          color: i <= filled ? segmentOn : segmentOff,
                          borderRadius: BorderRadius.circular(3),
                        ),
                      ),
                    ),
                  ),
                ),
            ],
          ),
          if (isMine) ...[
            const SizedBox(height: 6),
            Text('You: $mine',
                style: const TextStyle(
                    fontSize: 10, fontWeight: FontWeight.w700, color: Colors.white)),
          ],
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

/// Real booking flow — shown when a place has at least one active bookable
/// service (see placeServicesProvider), completely data-driven (not gated
/// by category), mirroring _DealCard's unclaimed/claimed two-state pattern:
/// a form to pick a slot/date-range + party size, then a QR/code
/// confirmation once booked.
class _BookingSection extends ConsumerStatefulWidget {
  final Place place;
  const _BookingSection({required this.place});

  @override
  ConsumerState<_BookingSection> createState() => _BookingSectionState();
}

class _BookingSectionState extends ConsumerState<_BookingSection> {
  String? _selectedServiceId;
  DateTime _selectedDate = DateTime.now();
  DateTime? _selectedSlotStart;
  DateTime? _checkIn;
  DateTime? _checkOut;
  int _partySize = 2;
  bool _submitting = false;
  Booking? _confirmed;

  @override
  Widget build(BuildContext context) {
    final servicesAsync = ref.watch(placeServicesProvider(widget.place.id));
    return servicesAsync.when(
      data: (services) {
        if (services.isEmpty) return const SizedBox.shrink();
        final service = services.firstWhere(
          (s) => s.id == _selectedServiceId,
          orElse: () => services.first,
        );
        _selectedServiceId ??= service.id;
        if (_partySize < service.partySizeMin) _partySize = service.partySizeMin;

        return Padding(
          padding: const EdgeInsets.only(bottom: 14),
          child: _confirmed != null
              ? _BookingConfirmedCard(
                  booking: _confirmed!,
                  onDone: () => setState(() => _confirmed = null),
                )
              : _BookingForm(
                  place: widget.place,
                  services: services,
                  selected: service,
                  onSelectService: (id) => setState(() {
                    _selectedServiceId = id;
                    _selectedSlotStart = null;
                  }),
                  selectedDate: _selectedDate,
                  onSelectDate: (d) => setState(() {
                    _selectedDate = d;
                    _selectedSlotStart = null;
                  }),
                  selectedSlotStart: _selectedSlotStart,
                  onSelectSlot: (s) => setState(() => _selectedSlotStart = s),
                  checkIn: _checkIn,
                  checkOut: _checkOut,
                  onCheckIn: (d) => setState(() => _checkIn = d),
                  onCheckOut: (d) => setState(() => _checkOut = d),
                  partySize: _partySize,
                  onPartySize: (n) => setState(() => _partySize = n),
                  submitting: _submitting,
                  onSubmit: _submit,
                ),
        );
      },
      loading: () => const SizedBox.shrink(),
      error: (_, _) => const SizedBox.shrink(),
    );
  }

  Future<void> _submit(BookableService service) async {
    final messenger = ScaffoldMessenger.of(context);
    if (service.type == 'stay') {
      if (_checkIn == null || _checkOut == null) return;
    } else if (_selectedSlotStart == null) {
      return;
    }
    setState(() => _submitting = true);
    try {
      final booking = await ref.read(bookingsProvider.notifier).create(
            placeId: widget.place.id,
            serviceId: service.id,
            startAt: service.type == 'stay' ? null : _selectedSlotStart,
            checkIn: service.type == 'stay' ? _checkIn : null,
            checkOut: service.type == 'stay' ? _checkOut : null,
            partySize: _partySize,
          );
      if (mounted) {
        setState(() {
          _confirmed = booking;
          _submitting = false;
        });
      }
    } catch (e) {
      if (mounted) setState(() => _submitting = false);
      messenger.showSnackBar(SnackBar(content: Text(ApiClient.errorMessage(e))));
    }
  }
}

class _BookingForm extends StatelessWidget {
  final Place place;
  final List<BookableService> services;
  final BookableService selected;
  final ValueChanged<String> onSelectService;
  final DateTime selectedDate;
  final ValueChanged<DateTime> onSelectDate;
  final DateTime? selectedSlotStart;
  final ValueChanged<DateTime> onSelectSlot;
  final DateTime? checkIn;
  final DateTime? checkOut;
  final ValueChanged<DateTime> onCheckIn;
  final ValueChanged<DateTime> onCheckOut;
  final int partySize;
  final ValueChanged<int> onPartySize;
  final bool submitting;
  final Future<void> Function(BookableService) onSubmit;

  const _BookingForm({
    required this.place,
    required this.services,
    required this.selected,
    required this.onSelectService,
    required this.selectedDate,
    required this.onSelectDate,
    required this.selectedSlotStart,
    required this.onSelectSlot,
    required this.checkIn,
    required this.checkOut,
    required this.onCheckIn,
    required this.onCheckOut,
    required this.partySize,
    required this.onPartySize,
    required this.submitting,
    required this.onSubmit,
  });

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    final isStay = selected.type == 'stay';
    final canSubmit = isStay ? (checkIn != null && checkOut != null) : selectedSlotStart != null;

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
              const Icon(Icons.event_available_outlined, size: 14, color: SpottoColors.muted),
              const SizedBox(width: 5),
              Text('BOOK',
                  style: text.labelSmall?.copyWith(color: SpottoColors.muted, letterSpacing: 0.6)),
            ],
          ),
          const SizedBox(height: 10),
          if (services.length > 1) ...[
            Wrap(
              spacing: 6,
              runSpacing: 6,
              children: [
                for (final s in services)
                  ChoiceChip(
                    label: Text(s.name, style: const TextStyle(fontSize: 12, fontWeight: FontWeight.w700)),
                    selected: s.id == selected.id,
                    onSelected: (_) => onSelectService(s.id),
                  ),
              ],
            ),
            const SizedBox(height: 10),
          ] else
            Text(selected.name, style: text.titleSmall?.copyWith(fontWeight: FontWeight.w800)),
          if (selected.priceLabel.isNotEmpty)
            Padding(
              padding: const EdgeInsets.only(top: 2, bottom: 8),
              child: Text(selected.priceLabel,
                  style: text.bodySmall?.copyWith(color: SpottoColors.muted, fontWeight: FontWeight.w700)),
            ),
          if (isStay) ...[
            Row(
              children: [
                Expanded(
                  child: _DateButton(
                    label: 'Check-in',
                    value: checkIn,
                    onTap: () async {
                      final picked = await showDatePicker(
                        context: context,
                        initialDate: DateTime.now().add(const Duration(days: 1)),
                        firstDate: DateTime.now(),
                        lastDate: DateTime.now().add(const Duration(days: 365)),
                      );
                      if (picked != null) onCheckIn(picked);
                    },
                  ),
                ),
                const SizedBox(width: 8),
                Expanded(
                  child: _DateButton(
                    label: 'Check-out',
                    value: checkOut,
                    onTap: () async {
                      final picked = await showDatePicker(
                        context: context,
                        initialDate: (checkIn ?? DateTime.now()).add(const Duration(days: 1)),
                        firstDate: checkIn ?? DateTime.now(),
                        lastDate: DateTime.now().add(const Duration(days: 366)),
                      );
                      if (picked != null) onCheckOut(picked);
                    },
                  ),
                ),
              ],
            ),
          ] else ...[
            SizedBox(
              height: 40,
              child: ListView.separated(
                scrollDirection: Axis.horizontal,
                itemCount: 14,
                separatorBuilder: (_, _) => const SizedBox(width: 6),
                itemBuilder: (context, i) {
                  final d = DateTime.now().add(Duration(days: i));
                  final on = d.year == selectedDate.year &&
                      d.month == selectedDate.month &&
                      d.day == selectedDate.day;
                  return PressScale(
                    onTap: () => onSelectDate(d),
                    child: Container(
                      width: 52,
                      alignment: Alignment.center,
                      decoration: BoxDecoration(
                        color: on ? SpottoColors.accent : Colors.white,
                        borderRadius: BorderRadius.circular(10),
                        border: Border.all(color: on ? SpottoColors.accent : SpottoColors.line),
                      ),
                      child: Column(
                        mainAxisAlignment: MainAxisAlignment.center,
                        children: [
                          Text(_weekdayShort(d.weekday),
                              style: TextStyle(
                                  fontSize: 10,
                                  fontWeight: FontWeight.w700,
                                  color: on ? Colors.white : SpottoColors.muted)),
                          Text('${d.day}',
                              style: TextStyle(
                                  fontSize: 13,
                                  fontWeight: FontWeight.w800,
                                  color: on ? Colors.white : SpottoColors.ink)),
                        ],
                      ),
                    ),
                  );
                },
              ),
            ),
            const SizedBox(height: 10),
            _SlotPicker(
              place: place,
              service: selected,
              date: selectedDate,
              selected: selectedSlotStart,
              onSelect: onSelectSlot,
            ),
          ],
          const SizedBox(height: 10),
          Row(
            children: [
              Text(isStay ? 'Guests' : 'Party size',
                  style: text.bodySmall?.copyWith(fontWeight: FontWeight.w800)),
              const Spacer(),
              IconButton(
                icon: const Icon(Icons.remove_circle_outline, size: 20),
                onPressed: partySize > (isStay ? 1 : selected.partySizeMin)
                    ? () => onPartySize(partySize - 1)
                    : null,
              ),
              Text('$partySize', style: text.titleSmall?.copyWith(fontWeight: FontWeight.w800)),
              IconButton(
                icon: const Icon(Icons.add_circle_outline, size: 20),
                onPressed: partySize < (isStay ? 20 : selected.partySizeMax)
                    ? () => onPartySize(partySize + 1)
                    : null,
              ),
            ],
          ),
          const SizedBox(height: 6),
          SizedBox(
            width: double.infinity,
            child: FilledButton(
              onPressed: canSubmit && !submitting ? () => onSubmit(selected) : null,
              child: Text(submitting ? 'Booking…' : 'Book now'),
            ),
          ),
        ],
      ),
    );
  }
}

String _weekdayShort(int weekday) => const ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'][weekday - 1];

class _DateButton extends StatelessWidget {
  final String label;
  final DateTime? value;
  final VoidCallback onTap;
  const _DateButton({required this.label, required this.value, required this.onTap});

  @override
  Widget build(BuildContext context) {
    return PressScale(
      onTap: onTap,
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 10),
        decoration: BoxDecoration(
          color: Colors.white,
          borderRadius: BorderRadius.circular(10),
          border: Border.all(color: SpottoColors.line),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(label,
                style: const TextStyle(fontSize: 10.5, fontWeight: FontWeight.w700, color: SpottoColors.muted)),
            Text(
              value == null ? 'Select' : '${value!.month}/${value!.day}/${value!.year}',
              style: const TextStyle(fontSize: 13, fontWeight: FontWeight.w800),
            ),
          ],
        ),
      ),
    );
  }
}

class _SlotPicker extends ConsumerWidget {
  final Place place;
  final BookableService service;
  final DateTime date;
  final DateTime? selected;
  final ValueChanged<DateTime> onSelect;
  const _SlotPicker({
    required this.place,
    required this.service,
    required this.date,
    required this.selected,
    required this.onSelect,
  });

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final dateStr =
        '${date.year.toString().padLeft(4, '0')}-${date.month.toString().padLeft(2, '0')}-${date.day.toString().padLeft(2, '0')}';
    final async = ref.watch(
      serviceAvailabilityProvider((placeId: place.id, serviceId: service.id, query: 'date=$dateStr')),
    );
    return async.when(
      data: (data) {
        final slots = [
          for (final s in (data['slots'] as List? ?? const []))
            BookingSlot.fromJson((s as Map).cast<String, dynamic>()),
        ];
        if (slots.isEmpty) {
          return const Text('No availability this day',
              style: TextStyle(fontSize: 12.5, fontWeight: FontWeight.w600, color: SpottoColors.muted));
        }
        return Wrap(
          spacing: 6,
          runSpacing: 6,
          children: [
            for (final slot in slots)
              PressScale(
                onTap: slot.remaining > 0 ? () => onSelect(slot.start) : null,
                child: Container(
                  padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 7),
                  decoration: BoxDecoration(
                    color: selected == slot.start
                        ? SpottoColors.accent
                        : (slot.remaining > 0 ? Colors.white : SpottoColors.line.withValues(alpha: 0.4)),
                    borderRadius: BorderRadius.circular(10),
                    border: Border.all(color: selected == slot.start ? SpottoColors.accent : SpottoColors.line),
                  ),
                  child: Text(
                    '${slot.start.hour.toString().padLeft(2, '0')}:${slot.start.minute.toString().padLeft(2, '0')}',
                    style: TextStyle(
                      fontSize: 12.5,
                      fontWeight: FontWeight.w700,
                      color: selected == slot.start
                          ? Colors.white
                          : (slot.remaining > 0 ? SpottoColors.ink : SpottoColors.muted),
                    ),
                  ),
                ),
              ),
          ],
        );
      },
      loading: () => const SizedBox(
        height: 32,
        child: Center(child: SizedBox(width: 16, height: 16, child: CircularProgressIndicator(strokeWidth: 2))),
      ),
      error: (_, _) => const Text('Could not load availability',
          style: TextStyle(fontSize: 12.5, fontWeight: FontWeight.w600, color: SpottoColors.muted)),
    );
  }
}

class _BookingConfirmedCard extends StatelessWidget {
  final Booking booking;
  final VoidCallback onDone;
  const _BookingConfirmedCard({required this.booking, required this.onDone});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(color: SpottoColors.ink, borderRadius: BorderRadius.circular(16)),
      child: Row(
        children: [
          Container(
            padding: const EdgeInsets.all(6),
            decoration: BoxDecoration(color: Colors.white, borderRadius: BorderRadius.circular(8)),
            child: QrImageView(data: booking.code, size: 64, gapless: false),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                const Text('Booked — show at check-in',
                    style: TextStyle(fontSize: 11.5, fontWeight: FontWeight.w700, color: Colors.white70)),
                const SizedBox(height: 3),
                Text(booking.code,
                    style: const TextStyle(
                        fontSize: 17, fontWeight: FontWeight.w800, color: Colors.white, letterSpacing: 0.5)),
                const SizedBox(height: 3),
                Text('Party of ${booking.partySize}',
                    style: const TextStyle(fontSize: 12.5, fontWeight: FontWeight.w600, color: Colors.white70)),
              ],
            ),
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
    final remaining = kMaxPhotosPerUpload - _photos.length;
    if (remaining <= 0) return;
    final files = await ImagePicker().pickMultiImage(
        maxWidth: 1600, imageQuality: 85, limit: remaining);
    if (files.isEmpty) return;
    final picked = <_PickedImage>[];
    var rejected = false;
    for (final f in files) {
      final contentType = mimeForUpload(f.mimeType, f.name);
      if (contentType == null) {
        rejected = true;
        continue;
      }
      picked.add(_PickedImage(await f.readAsBytes(), contentType));
    }
    if (mounted && rejected) {
      ScaffoldMessenger.of(context)
          .showSnackBar(const SnackBar(content: Text('Unsupported image type.')));
    }
    if (picked.isNotEmpty) setState(() => _photos.addAll(picked));
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
                    if (_photos.length < kMaxPhotosPerUpload)
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
            Builder(builder: (context) {
              // The grid shows the small thumbnail derivative (falling back
              // to the full image for photos uploaded before thumbnails
              // existed); the full-screen viewer always uses the original.
              final fullUrls = [for (final p in review.photos) resolveMediaUrl(p.url)];
              final thumbUrls = [
                for (final p in review.photos)
                  resolveMediaUrl(p.thumbUrl.isNotEmpty ? p.thumbUrl : p.url),
              ];
              return SizedBox(
                height: 84,
                child: ListView(
                  scrollDirection: Axis.horizontal,
                  children: [
                    for (final (i, url) in thumbUrls.indexed)
                      Padding(
                        padding: const EdgeInsets.only(right: 8),
                        child: GestureDetector(
                          onTap: () => showDialog<void>(
                            context: context,
                            // Tapping still escalates to the genuine
                            // full-resolution image — only the grid tile
                            // itself is capped at the thumb tier.
                            builder: (_) => _PhotoViewerDialog(urls: fullUrls, initialIndex: i),
                          ),
                          child: ProgressiveImage(
                            url: url,
                            microThumb: review.photos[i].microThumb,
                            width: 84,
                            height: 84,
                            fit: BoxFit.cover,
                            borderRadius: BorderRadius.circular(14),
                            errorBuilder: (_) =>
                                Container(width: 84, height: 84, color: SpottoColors.sand),
                          ),
                        ),
                      ),
                  ],
                ),
              );
            }),
          ],
        ],
      ),
    );
  }
}

/// Short relative-time label for a photo's createdAt — no existing helper
/// in this file to reuse, and simple enough not to warrant pulling in intl
/// just for this. Falls back to the ISO string's date slice once it's old
/// enough that "40d ago" stops being a useful unit.
String _timeAgo(String iso) {
  final parsed = DateTime.tryParse(iso);
  if (parsed == null) return '';
  final diff = DateTime.now().toUtc().difference(parsed.toUtc());
  if (diff.inMinutes < 1) return 'Just now';
  if (diff.inMinutes < 60) return '${diff.inMinutes}m ago';
  if (diff.inHours < 24) return '${diff.inHours}h ago';
  if (diff.inDays < 7) return '${diff.inDays}d ago';
  if (diff.inDays < 30) return '${(diff.inDays / 7).floor()}w ago';
  return iso.length >= 10 ? iso.substring(0, 10) : iso;
}

/// The tappable "add a photo" tile shared by the empty state and the grid —
/// sizes to whatever its parent gives it (a fixed SizedBox in the empty
/// state, a GridView cell otherwise).
class _AddPhotoTile extends StatelessWidget {
  final bool uploading;
  final VoidCallback onTap;
  const _AddPhotoTile({required this.uploading, required this.onTap});

  @override
  Widget build(BuildContext context) {
    return PressScale(
      onTap: uploading ? null : onTap,
      child: Container(
        decoration: BoxDecoration(
          color: SpottoColors.accentTint,
          borderRadius: BorderRadius.circular(14),
        ),
        child: Center(
          child: uploading
              ? const SizedBox(
                  width: 20,
                  height: 20,
                  child: CircularProgressIndicator(strokeWidth: 2.5, color: SpottoColors.accent),
                )
              : const Icon(Icons.add_a_photo_outlined, color: SpottoColors.accent, size: 24),
        ),
      ),
    );
  }
}

/// Traveller-contributed photos for a spot: a grid of uploaded photos (so
/// nothing's hidden off-screen the way a horizontal strip could) plus an
/// "Add" tile that picks and uploads to R2 (role contrib). Each photo shows
/// a relative "posted X ago" label, and carries a delete control when it's
/// one the signed-in traveller posted themselves.
class _TravellerPhotos extends ConsumerStatefulWidget {
  final Place place;
  const _TravellerPhotos({required this.place});

  @override
  ConsumerState<_TravellerPhotos> createState() => _TravellerPhotosState();
}

class _TravellerPhotosState extends ConsumerState<_TravellerPhotos> {
  bool _uploading = false;
  String? _deletingId;

  Future<void> _add() async {
    if (_uploading) return;
    final current =
        ref.read(placePhotosProvider(widget.place.id)).value ?? const <PlacePhoto>[];
    final remaining = kMaxPhotosPerUpload - current.length;
    if (remaining <= 0) return;
    final files = await ImagePicker().pickMultiImage(
      maxWidth: 1600,
      imageQuality: 85,
      limit: remaining,
    );
    if (files.isEmpty) return;
    setState(() => _uploading = true);
    var anyRejected = false;
    var anyFailed = false;
    try {
      for (final file in files) {
        final contentType = mimeForUpload(file.mimeType, file.name);
        if (contentType == null) {
          anyRejected = true;
          continue;
        }
        try {
          final bytes = await file.readAsBytes();
          await ref.read(uploadServiceProvider).upload(
                bytes: bytes,
                contentType: contentType,
                ownerType: 'place',
                ownerId: widget.place.id,
                role: 'contrib',
              );
        } catch (_) {
          anyFailed = true;
        }
      }
      ref.invalidate(placePhotosProvider(widget.place.id));
    } finally {
      if (mounted) setState(() => _uploading = false);
    }
    if (anyRejected) _snack('Unsupported image type.');
    if (anyFailed) _snack('Some photos failed to upload — try again.');
  }

  Future<void> _delete(PlacePhoto photo) async {
    if (_deletingId != null) return;
    setState(() => _deletingId = photo.id);
    try {
      await ref.read(uploadServiceProvider).delete(photo.id);
      ref.invalidate(placePhotosProvider(widget.place.id));
    } catch (e) {
      _snack(ApiClient.errorMessage(e));
    } finally {
      if (mounted) setState(() => _deletingId = null);
    }
  }

  void _snack(String message) {
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(message)));
  }

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    final photos =
        ref.watch(placePhotosProvider(widget.place.id)).value ?? const <PlacePhoto>[];
    final urls = [for (final p in photos) resolveMediaUrl(p.url)];
    // Grid tiles load the small thumbnail (falling back to the full image
    // for photos uploaded before thumbnails existed); the full-screen
    // viewer always opens the full-resolution `urls` above.
    final thumbUrls = [
      for (final p in photos) resolveMediaUrl(p.thumbUrl.isNotEmpty ? p.thumbUrl : p.url),
    ];
    final myId = ref.watch(meProvider)?['id'] as String?;
    final canAddMore = photos.length < kMaxPhotosPerUpload;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text('Traveller photos', style: text.titleMedium),
        const SizedBox(height: 8),
        if (photos.isEmpty)
          Row(
            children: [
              SizedBox(
                width: 84,
                height: 84,
                child: _AddPhotoTile(uploading: _uploading, onTap: _add),
              ),
              const SizedBox(width: 10),
              Expanded(
                child: Text('No photos yet — be the first to share one.',
                    style: text.bodySmall?.copyWith(color: SpottoColors.muted)),
              ),
            ],
          )
        else
          GridView.builder(
            shrinkWrap: true,
            physics: const NeverScrollableScrollPhysics(),
            itemCount: photos.length + (canAddMore ? 1 : 0),
            gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
              crossAxisCount: 3,
              crossAxisSpacing: 8,
              mainAxisSpacing: 8,
              childAspectRatio: 1,
            ),
            itemBuilder: (context, index) {
              if (canAddMore && index == 0) {
                return _AddPhotoTile(uploading: _uploading, onTap: _add);
              }
              final i = canAddMore ? index - 1 : index;
              final photo = photos[i];
              final isMine = myId != null && myId.isNotEmpty && photo.uploadedBy == myId;
              return GestureDetector(
                onTap: () => showDialog<void>(
                  context: context,
                  builder: (_) => _PhotoViewerDialog(urls: urls, initialIndex: i),
                ),
                child: Stack(
                  children: [
                    Positioned.fill(
                      child: ProgressiveImage(
                        url: thumbUrls[i],
                        microThumb: photo.microThumb,
                        fit: BoxFit.cover,
                        borderRadius: BorderRadius.circular(14),
                        errorBuilder: (_) => Container(color: SpottoColors.sand),
                      ),
                    ),
                    if (_timeAgo(photo.createdAt).isNotEmpty)
                      Positioned(
                        left: 4,
                        bottom: 4,
                        child: Container(
                          padding: const EdgeInsets.symmetric(horizontal: 5, vertical: 2),
                          decoration: BoxDecoration(
                            color: Colors.black54,
                            borderRadius: BorderRadius.circular(6),
                          ),
                          child: Text(_timeAgo(photo.createdAt),
                              style: const TextStyle(
                                  color: Colors.white, fontSize: 9, fontWeight: FontWeight.w700)),
                        ),
                      ),
                    if (isMine)
                      Positioned(
                        right: 4,
                        top: 4,
                        child: GestureDetector(
                          onTap: () => _delete(photo),
                          child: Container(
                            padding: const EdgeInsets.all(3),
                            decoration: const BoxDecoration(
                              color: Colors.black54,
                              shape: BoxShape.circle,
                            ),
                            child: _deletingId == photo.id
                                ? const SizedBox(
                                    width: 12,
                                    height: 12,
                                    child: CircularProgressIndicator(
                                        strokeWidth: 1.5, color: Colors.white),
                                  )
                                : const Icon(Icons.close, size: 14, color: Colors.white),
                          ),
                        ),
                      ),
                  ],
                ),
              );
            },
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

/// Full-screen swipeable photo viewer — opened by tapping any thumbnail in
/// a review's or a spot's traveller-contributed photo strip. Same full-
/// screen Dialog shape as _PromoVideoDialog above (black background, close
/// button top-right), with a PageView of photos in place of a single video
/// plus a page counter since there's more than one photo to swipe through.
class _PhotoViewerDialog extends StatefulWidget {
  final List<String> urls;
  final int initialIndex;
  const _PhotoViewerDialog({required this.urls, required this.initialIndex});

  @override
  State<_PhotoViewerDialog> createState() => _PhotoViewerDialogState();
}

class _PhotoViewerDialogState extends State<_PhotoViewerDialog> {
  late final PageController _controller;
  late int _index;
  double _dragOffset = 0;

  @override
  void initState() {
    super.initState();
    _index = widget.initialIndex;
    _controller = PageController(initialPage: widget.initialIndex);
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  void _onDragEnd(DragEndDetails details) {
    final flungFast = (details.primaryVelocity ?? 0).abs() > 800;
    if (_dragOffset.abs() > 120 || flungFast) {
      Navigator.of(context).pop();
    } else {
      setState(() => _dragOffset = 0);
    }
  }

  @override
  Widget build(BuildContext context) {
    final size = MediaQuery.of(context).size;
    // Fades the backdrop out as the photo is dragged down, so the dismiss
    // gesture reads as "pulling the photo away" rather than a hard cut.
    final dragProgress = (_dragOffset.abs() / 400).clamp(0.0, 1.0);
    return GestureDetector(
      onVerticalDragUpdate: (details) => setState(() => _dragOffset += details.delta.dy),
      onVerticalDragEnd: _onDragEnd,
      child: Dialog(
        backgroundColor: Colors.black.withValues(alpha: 1 - dragProgress * 0.7),
        insetPadding: EdgeInsets.zero,
        child: Transform.translate(
          offset: Offset(0, _dragOffset),
          child: SizedBox(
            width: size.width,
            height: size.height,
            child: Stack(
              alignment: Alignment.center,
              children: [
                PageView.builder(
                  controller: _controller,
                  itemCount: widget.urls.length,
                  onPageChanged: (i) => setState(() => _index = i),
                  itemBuilder: (_, i) => InteractiveViewer(
                    child: Center(
                      child: Image.network(
                        widget.urls[i],
                        fit: BoxFit.contain,
                        errorBuilder: (_, __, ___) => const Text("Couldn't load this photo.",
                            style: TextStyle(color: Colors.white)),
                        loadingBuilder: (_, child, progress) => progress == null
                            ? child
                            : const CircularProgressIndicator(color: Colors.white),
                      ),
                    ),
                  ),
                ),
                if (widget.urls.length > 1)
                  Positioned(
                    bottom: 24,
                    child: Container(
                      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
                      decoration: BoxDecoration(
                        color: Colors.black54,
                        borderRadius: BorderRadius.circular(20),
                      ),
                      child: Text('${_index + 1} / ${widget.urls.length}',
                          style: const TextStyle(
                              color: Colors.white, fontSize: 12, fontWeight: FontWeight.w700)),
                    ),
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
          ),
        ),
      ),
    );
  }
}
