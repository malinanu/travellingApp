// Trip tab — port of TripPlanner.jsx: month calendar → single-day timeline
// with times, budgets, walk/tuk-tuk legs and Google Maps deep-links.

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:intl/intl.dart';
import 'package:url_launcher/url_launcher.dart';

import '../../../core/theme/palette.dart';
import '../../../core/widgets/press_scale.dart';
import '../../../data/catalog/travellers.dart' as data;
import '../../../data/catalog/trip_catalog.dart';
import '../../../data/repositories/inspiration_repository.dart';
import '../../../data/repositories/repositories.dart';
import '../place_detail/business_drawer.dart';
import 'widgets/add_stop_sheet.dart';
import 'widgets/calendar_month.dart';
import 'widgets/confirm_delete_dialog.dart';
import '../../../core/api/api_client.dart';
import '../../../core/widgets/progressive_image.dart';
import '../../../data/models/models.dart';
import 'widgets/location_picker_sheet.dart';
import 'widgets/my_trips_sheet.dart';
import 'widgets/stop_card.dart';
import 'widgets/trip_common.dart';
import 'widgets/trip_share_sheet.dart';

class TripScreen extends ConsumerStatefulWidget {
  const TripScreen({super.key});

  @override
  ConsumerState<TripScreen> createState() => _TripScreenState();
}

class _TripScreenState extends ConsumerState<TripScreen> {
  int _dayIx = 0;
  bool _dayView = false;
  late DateTime _calMonth;

  @override
  void initState() {
    super.initState();
    final start = DateTime.parse(ref.read(tripProvider).startDate);
    _calMonth = DateTime(start.year, start.month, 1);
  }

  void _goDay(int dir) {
    final days = ref.read(tripProvider).days.length;
    setState(() => _dayIx = (_dayIx + dir).clamp(0, days - 1));
  }



  Future<void> _removeStop(TripStop stop, String name) async {
    final ok = await showConfirmDeleteDialog(context, name, tripWho(stop.by));
    if (ok) ref.read(tripProvider.notifier).removeStop(_dayIx, stop.id);
  }

  void _showMoveToDaySheet(int stopIx) {
    final trip = ref.read(tripProvider);
    showModalBottomSheet(
      context: context,
      backgroundColor: Colors.transparent,
      builder: (_) => Container(
        decoration: const BoxDecoration(
          color: Colors.white,
          borderRadius: BorderRadius.vertical(top: Radius.circular(24)),
        ),
        padding: const EdgeInsets.fromLTRB(20, 20, 20, 32),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text('Move to day',
                style: TextStyle(fontSize: 16, fontWeight: FontWeight.w800, color: SpottoColors.ink)),
            const SizedBox(height: 8),
            for (int d = 0; d < trip.days.length; d++)
              if (d != _dayIx)
                PressScale(
                  scale: 0.98,
                  onTap: () {
                    Navigator.pop(context);
                    ref.read(tripProvider.notifier).moveStopBetweenDays(
                        _dayIx, stopIx, d, trip.days[d].stops.length);
                  },
                  child: Padding(
                    padding: const EdgeInsets.symmetric(vertical: 12),
                    child: Row(
                      children: [
                        const Icon(Icons.calendar_today_rounded, size: 16, color: SpottoColors.accent),
                        const SizedBox(width: 10),
                        Text('Day ${d + 1} — ${DateFormat('MMM d').format(tripDayDate(trip.startDate, d))}',
                            style: const TextStyle(fontSize: 14.5, fontWeight: FontWeight.w700)),
                      ],
                    ),
                  ),
                ),
          ],
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    final trip = ref.watch(tripProvider);
    if (_dayIx >= trip.days.length) _dayIx = 0;
    final stops = trip.days[_dayIx].stops;

    ref.listen<Trip>(tripProvider, (prev, next) {
      if (prev?.startDate != next.startDate || prev?.code != next.code) {
        try {
          final start = DateTime.parse(next.startDate);
          setState(() {
            _dayIx = 0;
            _dayView = false;
            _calMonth = DateTime(start.year, start.month, 1);
          });
        } catch (_) {}
      }
    });

    return PopScope(
      canPop: !_dayView,
      onPopInvokedWithResult: (didPop, result) {
        if (didPop) return;
        if (_dayView) {
          setState(() => _dayView = false);
        }
      },
      child: SafeArea(
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(20, 12, 20, 0),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    children: [
                      Expanded(
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text('TRIP PLANNER',
                                style: text.labelSmall
                                    ?.copyWith(color: SpottoColors.accent, letterSpacing: 1)),
                            Text(trip.name,
                                maxLines: 1,
                                overflow: TextOverflow.ellipsis,
                                style: text.headlineSmall?.copyWith(fontSize: 21)),
                          ],
                        ),
                      ),
                      _myTripsButton(ref),
                      const SizedBox(width: 8),
                      _inspirationButton(trip),
                      const SizedBox(width: 8),
                      _collabButton(trip),
                    ],
                  ),
                  if (_dayView) ...[
                    _daySubHeader(text, trip),
                    _dayLocationChip(text, trip.days[_dayIx]),
                  ],
                ],
              ),
            ),
            Expanded(child: _dayView ? _dayBody(stops, trip.days[_dayIx]) : _calendarBody(trip)),
            if (_dayView && stops.isNotEmpty) _daySummary(text, stops),
          ],
        ),
      ),
    );
  }

  Widget _myTripsButton(WidgetRef ref) {
    final allTrips = ref.watch(savedTripsProvider);
    return PressScale(
      scale: 0.95,
      onTap: () => showMyTripsSheet(
        context,
        ref,
        onTripOpened: () => setState(() {
          _dayIx = 0;
          _dayView = true;
        }),
      ),
      child: Container(
        height: 40,
        padding: const EdgeInsets.symmetric(horizontal: 10),
        decoration: BoxDecoration(
          color: Colors.white,
          borderRadius: BorderRadius.circular(16),
          boxShadow: const [
            BoxShadow(color: Color(0x1A103D3A), blurRadius: 10, offset: Offset(0, 2))
          ],
        ),
        child: Row(
          children: [
            const Icon(Icons.flight_takeoff_rounded, size: 16, color: SpottoColors.accent),
            const SizedBox(width: 6),
            Text(
              'My Trips (${allTrips.length})',
              style: const TextStyle(fontSize: 12, fontWeight: FontWeight.w800, color: SpottoColors.ink),
            ),
          ],
        ),
      ),
    );
  }

  Widget _inspirationButton(Trip trip) {
    final count = ref.watch(boardLinksProvider).length;
    return PressScale(
      scale: 0.92,
      onTap: () => context.push('/t/inspiration'),
      child: Stack(
        clipBehavior: Clip.none,
        children: [
          Container(
            width: 40,
            height: 40,
            decoration: BoxDecoration(
              color: Colors.white,
              borderRadius: BorderRadius.circular(16),
              boxShadow: const [
                BoxShadow(color: Color(0x1A103D3A), blurRadius: 10, offset: Offset(0, 2))
              ],
            ),
            child: const Icon(Icons.bookmark_add_rounded, size: 18, color: SpottoColors.accent),
          ),
          if (count > 0)
            Positioned(
              right: -3,
              top: -3,
              child: Container(
                width: 17,
                height: 17,
                decoration: const BoxDecoration(
                  color: SpottoColors.accent,
                  shape: BoxShape.circle,
                ),
                child: Center(
                  child: Text(
                    count > 9 ? '9+' : '$count',
                    style: const TextStyle(
                        fontSize: 9, color: Colors.white, fontWeight: FontWeight.w900),
                  ),
                ),
              ),
            ),
        ],
      ),
    );
  }

  Widget _collabButton(Trip trip) {
    final members = [data.me, ...trip.collaborators.map(tripWho).nonNulls].take(3).toList();
    return PressScale(
      scale: 0.95,
      onTap: () => showTripShareSheet(context),
      child: Container(
        height: 40,
        padding: const EdgeInsets.fromLTRB(6, 0, 10, 0),
        decoration: BoxDecoration(
          color: Colors.white,
          borderRadius: BorderRadius.circular(16),
          boxShadow: const [
            BoxShadow(color: Color(0x1A103D3A), blurRadius: 10, offset: Offset(0, 2))
          ],
        ),
        child: Row(
          children: [
            SizedBox(
              width: 24.0 + (members.length - 1) * 16.0,
              height: 24,
              child: Stack(
                children: [
                  for (final (i, m) in members.indexed)
                    Positioned(
                      left: i * 16.0,
                      child: AvatarBubble(
                          initial: m.initial, color: Color(m.color), size: 24, borderWidth: 2),
                    ),
                ],
              ),
            ),
            const SizedBox(width: 6),
            const Icon(Icons.person_add_outlined, size: 15, color: SpottoColors.accent),
          ],
        ),
      ),
    );
  }



  Widget _daySubHeader(TextTheme text, Trip trip) {
    final dt = tripDayDate(trip.startDate, _dayIx);
    Widget nav(IconData icon, bool disabled, VoidCallback onTap) => PressScale(
          scale: 0.9,
          onTap: disabled ? null : onTap,
          child: Container(
            width: 36,
            height: 36,
            decoration: BoxDecoration(
              color: Colors.white,
              borderRadius: BorderRadius.circular(12),
              boxShadow: const [
                BoxShadow(color: Color(0x1A103D3A), blurRadius: 8, offset: Offset(0, 2))
              ],
            ),
            child: Icon(icon,
                size: 18,
                color: disabled ? SpottoColors.ink.withValues(alpha: 0.25) : SpottoColors.ink),
          ),
        );

    return Padding(
      padding: const EdgeInsets.only(top: 12),
      child: Row(
        children: [
          nav(Icons.chevron_left, false, () => setState(() => _dayView = false)),
          const SizedBox(width: 10),
          Expanded(
            child: Row(
              crossAxisAlignment: CrossAxisAlignment.baseline,
              textBaseline: TextBaseline.alphabetic,
              children: [
                Flexible(
                  child: Text(DateFormat('EEEE, MMM d').format(dt),
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: text.titleMedium?.copyWith(fontSize: 16)),
                ),
                const SizedBox(width: 8),
                Text('Day ${_dayIx + 1}',
                    style: text.labelMedium?.copyWith(color: SpottoColors.accent)),
              ],
            ),
          ),
          nav(Icons.chevron_left, _dayIx == 0, () => _goDay(-1)),
          const SizedBox(width: 6),
          nav(Icons.chevron_right, _dayIx == ref.read(tripProvider).days.length - 1,
              () => _goDay(1)),
        ],
      ),
    );
  }

  Widget _dayLocationChip(TextTheme text, TripDay day) {
    final hasLoc = day.location != null && day.location!.isNotEmpty;
    return Padding(
      padding: const EdgeInsets.only(top: 10),
      child: PressScale(
        scale: 0.98,
        onTap: () async {
          final choice = await showLocationPickerSheet(context, currentLocation: day.location);
          if (choice != null) {
            final isCleared = choice.name.isEmpty;
            ref.read(tripProvider.notifier).setDayLocation(
                  _dayIx,
                  location: isCleared ? null : choice.name,
                  regionId: isCleared ? null : choice.regionId,
                );
          }
        },
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
          decoration: BoxDecoration(
            color: hasLoc ? SpottoColors.accentTint : Colors.white,
            borderRadius: BorderRadius.circular(14),
            border: Border.all(
              color: hasLoc ? SpottoColors.accent.withValues(alpha: 0.35) : SpottoColors.line,
              width: 1.2,
            ),
            boxShadow: const [
              BoxShadow(color: Color(0x0D103D3A), blurRadius: 6, offset: Offset(0, 2))
            ],
          ),
          child: Row(
            children: [
              Icon(
                hasLoc ? Icons.location_on : Icons.add_location_alt_outlined,
                size: 16,
                color: SpottoColors.accent,
              ),
              const SizedBox(width: 8),
              Expanded(
                child: Text(
                  hasLoc ? 'Visiting ${day.location}' : 'Set location (e.g. Ella, Mirissa) to get suggestions…',
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: TextStyle(
                    fontSize: 13,
                    fontWeight: hasLoc ? FontWeight.w800 : FontWeight.w600,
                    color: hasLoc ? SpottoColors.ink : SpottoColors.muted,
                  ),
                ),
              ),
              const SizedBox(width: 6),
              Container(
                padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3.5),
                decoration: BoxDecoration(
                  color: hasLoc ? SpottoColors.accent.withValues(alpha: 0.12) : SpottoColors.sand,
                  borderRadius: BorderRadius.circular(8),
                ),
                child: Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    Text(
                      hasLoc ? 'Change' : 'Pick town',
                      style: const TextStyle(
                        fontSize: 11,
                        fontWeight: FontWeight.w700,
                        color: SpottoColors.accent,
                      ),
                    ),
                    const Icon(Icons.arrow_drop_down, size: 14, color: SpottoColors.accent),
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _calendarBody(Trip trip) {
    final text = Theme.of(context).textTheme;
    final planned = <String, PlannedDay>{};
    for (final (i, d) in trip.days.indexed) {
      planned[CalendarMonth.keyOf(tripDayDate(trip.startDate, i))] =
          PlannedDay(index: i, stops: d.stops.length);
    }
    final first = tripDayDate(trip.startDate, 0);
    final last = tripDayDate(trip.startDate, trip.days.length - 1);
    final rangeLabel =
        '${DateFormat('MMM d').format(first)} – ${DateFormat('MMM d').format(last)}';

    return ListView(
      padding: const EdgeInsets.fromLTRB(20, 16, 20, 16),
      children: [
        Row(
          children: [
            Container(
              padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 5),
              decoration: BoxDecoration(
                color: SpottoColors.accentTint,
                borderRadius: BorderRadius.circular(999),
              ),
              child: Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  const Icon(Icons.calendar_month, size: 13, color: SpottoColors.accent),
                  const SizedBox(width: 5),
                  Text('${trip.days.length}-day trip · $rangeLabel',
                      style: const TextStyle(
                          fontSize: 11.5,
                          fontWeight: FontWeight.w800,
                          color: SpottoColors.accent)),
                ],
              ),
            ),
          ],
        ),
        const SizedBox(height: 12),
        CalendarMonth(
          month: _calMonth,
          planned: planned,
          onPickDay: (i) => setState(() {
            _dayIx = i;
            _dayView = true;
          }),
          onPrevMonth: () =>
              setState(() => _calMonth = DateTime(_calMonth.year, _calMonth.month - 1, 1)),
          onNextMonth: () =>
              setState(() => _calMonth = DateTime(_calMonth.year, _calMonth.month + 1, 1)),
        ),
        const SizedBox(height: 10),
        Row(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Row(
              children: [
                Container(
                  width: 10,
                  height: 10,
                  decoration: const BoxDecoration(color: SpottoColors.accent, shape: BoxShape.circle),
                ),
                const SizedBox(width: 5),
                const Text('Planned Day', style: TextStyle(fontSize: 11.5, color: SpottoColors.ink, fontWeight: FontWeight.w700)),
              ],
            ),
            const SizedBox(width: 18),
            Row(
              children: [
                Container(
                  width: 10,
                  height: 10,
                  decoration: const BoxDecoration(color: Color(0xFFFF9800), shape: BoxShape.circle),
                ),
                const SizedBox(width: 5),
                const Text('Unplanned Day', style: TextStyle(fontSize: 11.5, color: SpottoColors.ink, fontWeight: FontWeight.w700)),
              ],
            ),
          ],
        ),
        const SizedBox(height: 8),
        Text('Tap a highlighted day to open its plan.',
            textAlign: TextAlign.center,
            style: text.bodySmall?.copyWith(color: SpottoColors.muted)),
        ...[
          const SizedBox(height: 12),
          PressScale(
            scale: 0.98,
            onTap: () => ref.read(tripProvider.notifier).addDay(),
            child: Container(
              padding: const EdgeInsets.symmetric(vertical: 12),
              decoration: BoxDecoration(
                borderRadius: BorderRadius.circular(16),
                border: Border.all(color: SpottoColors.accent, width: 1.5),
              ),
              child: const Row(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  Icon(Icons.add, size: 15, color: SpottoColors.accent),
                  SizedBox(width: 6),
                  Text('Add another day',
                      style: TextStyle(
                          fontSize: 13.5,
                          fontWeight: FontWeight.w800,
                          color: SpottoColors.accent)),
                ],
              ),
            ),
          ),
        ],
      ],
    );
  }

  Widget _dayBody(List<TripStop> stops, TripDay currentDay) {
    final text = Theme.of(context).textTheme;
    final legs = <Leg?>[
      for (var i = 1; i < stops.length; i++)
        () {
          final from = tripPlace(stops[i - 1]);
          final to = tripPlace(stops[i]);
          return from != null && to != null ? tripLeg(from, to) : null;
        }(),
    ];

    final allPlaces = ref.watch(placesProvider);
    final prefs = ref.watch(prefsProvider);
    final userStyles = prefs?.styles ?? const <String>[];
    final usedPlaceIds = stops.map((s) => s.placeId).toSet();

    final dayLoc = (currentDay.location ?? '').trim().toLowerCase();
    final dayReg = (currentDay.regionId ?? '').trim().toLowerCase();

    final suggestedSpots = allPlaces.where((p) {
      if (usedPlaceIds.contains(p.id)) return false;
      final pr = p.region.toLowerCase();
      final prid = p.regionId.toLowerCase();
      if (dayReg.isNotEmpty && prid == dayReg) return true;
      if (dayLoc.isNotEmpty) return pr.contains(dayLoc) || prid.contains(dayLoc);
      return false;
    }).toList();

    suggestedSpots.sort((a, b) {
      final aStyle = a.styles.any(userStyles.contains) ? 1 : 0;
      final bStyle = b.styles.any(userStyles.contains) ? 1 : 0;
      if (aStyle != bStyle) return bStyle.compareTo(aStyle);
      return b.vibe.sampleSize.compareTo(a.vibe.sampleSize);
    });

    final hasLocation = currentDay.location != null && currentDay.location!.isNotEmpty;

    return GestureDetector(
      onHorizontalDragEnd: (d) {
        final v = d.primaryVelocity ?? 0;
        if (v.abs() > 250) _goDay(v < 0 ? 1 : -1);
      },
      child: ListView(
        padding: const EdgeInsets.fromLTRB(20, 12, 20, 16),
        children: [
          if (stops.isEmpty) ...[
            if (hasLocation) ...[
              // Destination chosen: show smart suggestions for that town
              Padding(
                padding: const EdgeInsets.fromLTRB(4, 12, 4, 14),
                child: Row(
                  children: [
                    Container(
                      padding: const EdgeInsets.all(6),
                      decoration: BoxDecoration(
                        color: SpottoColors.accentTint,
                        borderRadius: BorderRadius.circular(8),
                      ),
                      child: const Icon(Icons.auto_awesome, size: 16, color: SpottoColors.accent),
                    ),
                    const SizedBox(width: 8),
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            'Suggested for you in ${currentDay.location}',
                            style: const TextStyle(fontSize: 15, fontWeight: FontWeight.w800, color: SpottoColors.ink),
                          ),
                          Text(
                            'Based on your travel style — tap to add or inspect details',
                            style: text.bodySmall?.copyWith(color: SpottoColors.muted),
                          ),
                        ],
                      ),
                    ),
                  ],
                ),
              ),
              if (suggestedSpots.isNotEmpty) ...[
                for (final place in suggestedSpots.take(5))
                  _spotSuggestionCard(place, currentDay),
              ] else ...[
                Container(
                  padding: const EdgeInsets.all(16),
                  margin: const EdgeInsets.only(bottom: 12),
                  decoration: BoxDecoration(
                    color: Colors.white,
                    borderRadius: BorderRadius.circular(16),
                    border: Border.all(color: SpottoColors.line),
                  ),
                  child: Column(
                    children: [
                      const Icon(Icons.explore_outlined, size: 32, color: SpottoColors.muted),
                      const SizedBox(height: 8),
                      Text(
                        'No local spots indexed for ${currentDay.location} yet',
                        style: const TextStyle(fontSize: 14, fontWeight: FontWeight.bold, color: SpottoColors.ink),
                      ),
                      const SizedBox(height: 4),
                      Text(
                        'Tap "+ Add a stop" below to search any hotel, beach, cafe, or Google Maps place in ${currentDay.location}.',
                        textAlign: TextAlign.center,
                        style: text.bodySmall?.copyWith(color: SpottoColors.muted),
                      ),
                    ],
                  ),
                ),
              ],
            ] else ...[
              // No destination chosen yet: show default empty + quick town picker
              Padding(
                padding: const EdgeInsets.fromLTRB(24, 28, 24, 16),
                child: Column(
                  children: [
                    Container(
                      width: 58,
                      height: 58,
                      decoration: const BoxDecoration(
                        color: SpottoColors.accentTint,
                        shape: BoxShape.circle,
                      ),
                      child: const Icon(Icons.route, size: 26, color: SpottoColors.accent),
                    ),
                    const SizedBox(height: 14),
                    Text('Nothing planned yet', style: text.titleMedium?.copyWith(fontSize: 18)),
                    const SizedBox(height: 4),
                    Text(
                      'Select where you are visiting on Day ${_dayIx + 1} to get personalized spot recommendations:',
                      textAlign: TextAlign.center,
                      style: text.bodySmall?.copyWith(color: SpottoColors.muted),
                    ),
                    const SizedBox(height: 14),
                    Wrap(
                      spacing: 8,
                      runSpacing: 8,
                      alignment: WrapAlignment.center,
                      children: [
                        _quickTownPill('Ella', 'ella'),
                        _quickTownPill('Galle', 'galle'),
                        _quickTownPill('Mirissa', 'mirissa'),
                        _quickTownPill('Weligama', 'weligama'),
                        _quickTownPill('Kandy', 'kandy'),
                        _quickTownPill('Colombo', 'colombo'),
                        _quickTownPill('Sigiriya', 'sigiriya'),
                        PressScale(
                          scale: 0.94,
                          onTap: () async {
                            final choice = await showLocationPickerSheet(context, currentLocation: null);
                            if (choice != null && choice.name.isNotEmpty) {
                              ref.read(tripProvider.notifier).setDayLocation(
                                    _dayIx,
                                    location: choice.name,
                                    regionId: choice.regionId,
                                  );
                            }
                          },
                          child: Container(
                            padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 5),
                            decoration: BoxDecoration(
                              color: SpottoColors.sand,
                              borderRadius: BorderRadius.circular(16),
                              border: Border.all(color: SpottoColors.line),
                            ),
                            child: const Row(
                              mainAxisSize: MainAxisSize.min,
                              children: [
                                Text('More towns…', style: TextStyle(fontSize: 11.5, fontWeight: FontWeight.bold, color: SpottoColors.accent)),
                                Icon(Icons.arrow_drop_down, size: 14, color: SpottoColors.accent),
                              ],
                            ),
                          ),
                        ),
                      ],
                    ),
                  ],
                ),
              ),
            ],
          ] else ...[
            for (final (i, s) in stops.indexed) ...[
              if (i > 0 && legs[i - 1] != null) LegRow(leg: legs[i - 1]!),
              if (i > 0 && legs[i - 1] == null) const SizedBox(height: 10),
              Builder(builder: (context) {
                final place = tripPlace(s);
                if (place == null) return const SizedBox.shrink();
                return StopCard(
                  stop: s,
                  place: place,
                  first: i == 0,
                  last: i == stops.length - 1,
                  by: tripWho(s.by),
                  onTime: (t) => ref.read(tripProvider.notifier).setStopTime(_dayIx, s.id, t),
                  onUp: () => ref.read(tripProvider.notifier).moveStop(_dayIx, i, -1),
                  onDown: () => ref.read(tripProvider.notifier).moveStop(_dayIx, i, 1),
                  onRemove: () => _removeStop(s, place.name),
                  onMoveToDay: ref.read(tripProvider).days.length > 1
                      ? () => _showMoveToDaySheet(i)
                      : null,
                  onOpen: place.spottoId == null
                      ? null
                      : () {
                          final p = ref
                              .read(placesProvider)
                              .where((x) => x.id == place.spottoId)
                              .firstOrNull;
                          if (p != null) showBusinessDrawer(context, p);
                        },
                );
              }),
            ],
          ],

          const SizedBox(height: 12),
          PressScale(
            scale: 0.98,
            onTap: () => showAddStopSheet(context, ref, _dayIx),
            child: Container(
              padding: const EdgeInsets.symmetric(vertical: 14),
              decoration: BoxDecoration(
                borderRadius: BorderRadius.circular(24),
                border: Border.all(color: SpottoColors.accent, width: 2),
              ),
              child: const Row(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  Icon(Icons.add, size: 17, color: SpottoColors.accent),
                  SizedBox(width: 6),
                  Text('Add a stop',
                      style: TextStyle(
                          fontSize: 14.5,
                          fontWeight: FontWeight.w800,
                          color: SpottoColors.accent)),
                ],
              ),
            ),
          ),

          // If stops already exist and there are more spots in this town:
          if (stops.isNotEmpty && hasLocation && suggestedSpots.isNotEmpty) ...[
            const SizedBox(height: 20),
            Row(
              children: [
                const Icon(Icons.explore_outlined, size: 15, color: SpottoColors.accent),
                const SizedBox(width: 6),
                Text(
                  'More in ${currentDay.location} you might like',
                  style: const TextStyle(fontSize: 13.5, fontWeight: FontWeight.w800, color: SpottoColors.ink),
                ),
              ],
            ),
            const SizedBox(height: 8),
            for (final place in suggestedSpots.take(3))
              _spotSuggestionCard(place, currentDay),
          ],
        ],
      ),
    );
  }

  Widget _quickTownPill(String name, String regionId) {
    return PressScale(
      scale: 0.94,
      onTap: () => ref.read(tripProvider.notifier).setDayLocation(_dayIx, location: name, regionId: regionId),
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 11, vertical: 5),
        decoration: BoxDecoration(
          color: Colors.white,
          borderRadius: BorderRadius.circular(16),
          border: Border.all(color: SpottoColors.line),
          boxShadow: const [
            BoxShadow(color: Color(0x0A103D3A), blurRadius: 4, offset: Offset(0, 1))
          ],
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            const Icon(Icons.place, size: 12, color: SpottoColors.accent),
            const SizedBox(width: 4),
            Text(name, style: const TextStyle(fontSize: 12, fontWeight: FontWeight.w700, color: SpottoColors.ink)),
          ],
        ),
      ),
    );
  }

  Widget _spotSuggestionCard(Place place, TripDay currentDay) {
    final text = Theme.of(context).textTheme;
    final imgUrl = place.coverThumb.isNotEmpty ? place.coverThumb : place.cover;

    return Container(
      margin: const EdgeInsets.only(bottom: 8),
      decoration: BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.circular(16),
        border: Border.all(color: SpottoColors.line, width: 1.2),
        boxShadow: const [
          BoxShadow(color: Color(0x0D103D3A), blurRadius: 8, offset: Offset(0, 2))
        ],
      ),
      child: Material(
        color: Colors.transparent,
        child: InkWell(
          borderRadius: BorderRadius.circular(16),
          onTap: () => showBusinessDrawer(context, place),
          child: Padding(
            padding: const EdgeInsets.all(10),
            child: Row(
              children: [
                ClipRRect(
                  borderRadius: BorderRadius.circular(12),
                  child: SizedBox(
                    width: 58,
                    height: 58,
                    child: imgUrl.isNotEmpty
                        ? ProgressiveImage(
                            url: resolveMediaUrl(imgUrl),
                            width: 58,
                            height: 58,
                            fit: BoxFit.cover,
                          )
                        : Container(
                            color: SpottoColors.accentTint,
                            child: const Icon(Icons.place, color: SpottoColors.accent),
                          ),
                  ),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        place.name,
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                        style: const TextStyle(fontSize: 14, fontWeight: FontWeight.w800, color: SpottoColors.ink),
                      ),
                      const SizedBox(height: 2),
                      Row(
                        children: [
                          Flexible(
                            child: Text(
                              [place.type, if (place.region.isNotEmpty) place.region].join(' · '),
                              maxLines: 1,
                              overflow: TextOverflow.ellipsis,
                              style: text.bodySmall?.copyWith(color: SpottoColors.muted, fontSize: 11.5),
                            ),
                          ),
                          if (place.vibe.score > 0) ...[
                            const SizedBox(width: 6),
                            Container(
                              padding: const EdgeInsets.symmetric(horizontal: 5, vertical: 1),
                              decoration: BoxDecoration(
                                color: SpottoColors.accentTint,
                                borderRadius: BorderRadius.circular(4),
                              ),
                              child: Text(
                                '${(place.vibe.score * 20).round()}% Vibe',
                                style: const TextStyle(fontSize: 9.5, fontWeight: FontWeight.bold, color: SpottoColors.accent),
                              ),
                            ),
                          ],
                        ],
                      ),
                    ],
                  ),
                ),
                const SizedBox(width: 8),
                PressScale(
                  scale: 0.94,
                  onTap: () {
                    ref.read(tripProvider.notifier).addPlaceStop(_dayIx, place);
                  },
                  child: Container(
                    padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 7),
                    decoration: BoxDecoration(
                      color: SpottoColors.accent,
                      borderRadius: BorderRadius.circular(10),
                    ),
                    child: const Row(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        Icon(Icons.add, size: 14, color: Colors.white),
                        SizedBox(width: 3),
                        Text(
                          'Add',
                          style: TextStyle(fontSize: 12, fontWeight: FontWeight.w800, color: Colors.white),
                        ),
                      ],
                    ),
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }

  Widget _daySummary(TextTheme text, List<TripStop> stops) {
    final places = stops.map(tripPlace).nonNulls.toList();
    final legs = <Leg>[
      for (var i = 1; i < places.length; i++) tripLeg(places[i - 1], places[i]),
    ];
    final totalCost = places.fold(0, (a, p) => a + p.cost);
    final totalTravel = legs.fold(0, (a, l) => a + l.mins);
    final dayRoute = tripDayGmaps(stops);

    Widget stat(IconData icon, String label) => Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(icon, size: 14, color: SpottoColors.accent),
            const SizedBox(width: 5),
            Text(label,
                style: const TextStyle(
                    fontSize: 12.5, fontWeight: FontWeight.w800, color: SpottoColors.ink)),
          ],
        );

    return Container(
      width: double.infinity,
      padding: const EdgeInsets.fromLTRB(20, 12, 20, 12),
      decoration: const BoxDecoration(
        color: Colors.white,
        boxShadow: [
          BoxShadow(color: Color(0x4D103D3A), blurRadius: 20, offset: Offset(0, -4), spreadRadius: -12)
        ],
      ),
      child: Column(
        children: [
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              stat(Icons.place, '${stops.length} stops'),
              stat(Icons.timer_outlined, '$totalTravel min travel'),
              stat(Icons.account_balance_wallet_outlined, tripFmtRs(totalCost)),
            ],
          ),
          if (dayRoute != null) ...[
            const SizedBox(height: 10),
            SizedBox(
              width: double.infinity,
              child: FilledButton.icon(
                onPressed: () =>
                    launchUrl(Uri.parse(dayRoute), mode: LaunchMode.externalApplication),
                icon: const Icon(Icons.near_me, size: 16),
                label: const Text('Open day route in Google Maps'),
              ),
            ),
          ],
        ],
      ),
    );
  }
}
