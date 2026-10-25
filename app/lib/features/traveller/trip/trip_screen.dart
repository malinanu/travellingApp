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
                  if (_dayView) _daySubHeader(text, trip),
                ],
              ),
            ),
            Expanded(child: _dayView ? _dayBody(stops) : _calendarBody(trip)),
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

  Widget _dayBody(List<TripStop> stops) {
    final text = Theme.of(context).textTheme;
    final legs = <Leg?>[
      for (var i = 1; i < stops.length; i++)
        () {
          final from = tripPlace(stops[i - 1]);
          final to = tripPlace(stops[i]);
          return from != null && to != null ? tripLeg(from, to) : null;
        }(),
    ];

    return GestureDetector(
      onHorizontalDragEnd: (d) {
        final v = d.primaryVelocity ?? 0;
        if (v.abs() > 250) _goDay(v < 0 ? 1 : -1);
      },
      child: ListView(
        padding: const EdgeInsets.fromLTRB(20, 12, 20, 16),
        children: [
          if (stops.isEmpty)
            Padding(
              padding: const EdgeInsets.fromLTRB(24, 56, 24, 12),
              child: Column(
                children: [
                  Container(
                    width: 64,
                    height: 64,
                    decoration: const BoxDecoration(
                        color: SpottoColors.accentTint, shape: BoxShape.circle),
                    child: const Icon(Icons.route, size: 28, color: SpottoColors.accent),
                  ),
                  const SizedBox(height: 16),
                  Text('Nothing planned yet', style: text.titleMedium?.copyWith(fontSize: 18)),
                  const SizedBox(height: 6),
                  Text(
                      'Add your first stop — search any place or pick a suggestion for your style.',
                      textAlign: TextAlign.center,
                      style: text.bodyMedium?.copyWith(color: SpottoColors.muted)),
                ],
              ),
            )
          else
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
        ],
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
