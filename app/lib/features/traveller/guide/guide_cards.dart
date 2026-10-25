// Local Guide cards: trip questions with tappable answers, the visual trip
// plan, and "add to Trip calendar". Payload models are in guide_payload.dart.

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:intl/intl.dart';
import 'package:url_launcher/url_launcher.dart';

import '../../../core/theme/palette.dart';
import '../../../core/widgets/press_scale.dart';
import '../../../data/catalog/trip_catalog.dart';
import '../../../data/models/inspiration.dart' show kindLabel;
import '../../../data/repositories/repositories.dart';
import '../inspiration/board_widgets.dart';
import 'guide_payload.dart';

BoxDecoration _cardBox(BuildContext context) => BoxDecoration(
      color: Theme.of(context).colorScheme.surface,
      borderRadius: BorderRadius.circular(20),
      boxShadow: const [BoxShadow(color: Color(0x1F103D3A), blurRadius: 12, offset: Offset(0, 2))],
    );

String _fmtDate(String iso) {
  final d = DateTime.tryParse(iso);
  return d == null ? iso : DateFormat('EEE d MMM').format(d);
}

String _fmtClock(String hhmm) {
  final m = parseClock(hhmm);
  return m == null ? '' : tripFmtTime(m);
}

// ── questions ──────────────────────────────────────────────────────

/// Questions the guide asks before planning. Answers are sent back as one
/// message; once the conversation moves on, the card shows them read-only.
class GuideQuestionsCard extends StatefulWidget {
  final List<GuideQuestion> questions;
  final bool answered;
  final ValueChanged<String> onSubmit;
  const GuideQuestionsCard({super.key, required this.questions, required this.answered, required this.onSubmit});

  @override
  State<GuideQuestionsCard> createState() => _GuideQuestionsCardState();
}

class _GuideQuestionsCardState extends State<GuideQuestionsCard> {
  final _answers = <String, List<String>>{};
  final _text = <String, TextEditingController>{};

  @override
  void dispose() {
    for (final c in _text.values) {
      c.dispose();
    }
    super.dispose();
  }

  TextEditingController _controller(String id) => _text.putIfAbsent(id, TextEditingController.new);

  Map<String, List<String>> get _collected => {
        for (final q in widget.questions)
          q.id: [
            ...?_answers[q.id],
            if (q.kind == 'text' && _controller(q.id).text.trim().isNotEmpty) _controller(q.id).text.trim(),
          ],
      };

  bool get _ready => _collected.values.any((a) => a.isNotEmpty);

  void _toggle(GuideQuestion q, String option) {
    setState(() {
      final cur = [...?_answers[q.id]];
      if (q.kind == 'multi') {
        cur.contains(option) ? cur.remove(option) : cur.add(option);
      } else {
        if (cur.contains(option)) {
          cur.clear();
        } else {
          cur
            ..clear()
            ..add(option);
        }
        if (q.kind == 'text') _controller(q.id).clear();
      }
      _answers[q.id] = cur;
    });
  }

  Future<void> _pickDates(GuideQuestion q) async {
    final now = DateTime.now();
    final range = await showDateRangePicker(
      context: context,
      firstDate: DateTime(now.year, now.month, now.day),
      lastDate: now.add(const Duration(days: 730)),
      helpText: q.text,
    );
    if (range == null) return;
    final nights = range.end.difference(range.start).inDays;
    final f = DateFormat('d MMM yyyy');
    final iso = DateFormat('yyyy-MM-dd');
    setState(() => _answers[q.id] = [
          '${f.format(range.start)} to ${f.format(range.end)} ($nights ${nights == 1 ? 'night' : 'nights'}, '
              'starting ${iso.format(range.start)})',
        ]);
  }

  Widget _chip(String label, bool selected, VoidCallback? onTap) => PressScale(
        scale: 0.95,
        onTap: onTap,
        child: AnimatedContainer(
          duration: const Duration(milliseconds: 150),
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
          decoration: BoxDecoration(
            color: selected ? SpottoColors.accent : Theme.of(context).colorScheme.surface,
            borderRadius: BorderRadius.circular(999),
            border: Border.all(color: selected ? SpottoColors.accent : SpottoColors.line, width: 1.5),
          ),
          child: Text(label,
              style: TextStyle(
                fontSize: 12.5,
                fontWeight: FontWeight.w700,
                color: selected ? Colors.white : Theme.of(context).colorScheme.onSurface,
              )),
        ),
      );

  @override
  Widget build(BuildContext context) {
    final locked = widget.answered;
    return Container(
      padding: const EdgeInsets.fromLTRB(14, 14, 14, 12),
      decoration: _cardBox(context),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          for (final q in widget.questions) ...[
            Text(q.text, style: const TextStyle(fontSize: 13.5, fontWeight: FontWeight.w800)),
            const SizedBox(height: 8),
            if (q.kind == 'dates')
              _chip(
                (_answers[q.id] ?? const []).isEmpty ? 'Pick dates' : _answers[q.id]!.first.split(' (').first,
                (_answers[q.id] ?? const []).isNotEmpty,
                locked ? null : () => _pickDates(q),
              ),
            if (q.options.isNotEmpty)
              Padding(
                padding: EdgeInsets.only(top: q.kind == 'dates' ? 8 : 0),
                child: Wrap(spacing: 8, runSpacing: 8, children: [
                  for (final o in q.options)
                    _chip(o, _answers[q.id]?.contains(o) ?? false, locked ? null : () => _toggle(q, o)),
                ]),
              ),
            if (q.kind == 'text' && !locked)
              Padding(
                padding: const EdgeInsets.only(top: 8),
                child: TextField(
                  controller: _controller(q.id),
                  onChanged: (v) => setState(() {
                    if (v.trim().isNotEmpty) _answers[q.id] = [];
                  }),
                  decoration: InputDecoration(
                    isDense: true,
                    hintText: 'Type your answer…',
                    contentPadding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
                    border: OutlineInputBorder(borderRadius: BorderRadius.circular(12)),
                  ),
                  style: const TextStyle(fontSize: 13.5),
                ),
              ),
            const SizedBox(height: 14),
          ],
          if (!locked)
            SizedBox(
              width: double.infinity,
              child: FilledButton.icon(
                onPressed: _ready ? () => widget.onSubmit(composeGuideAnswers(widget.questions, _collected)) : null,
                icon: const Icon(Icons.auto_awesome, size: 18),
                label: const Text('Plan my trip'),
                style: FilledButton.styleFrom(
                  backgroundColor: SpottoColors.accent,
                  padding: const EdgeInsets.symmetric(vertical: 12),
                  shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(14)),
                ),
              ),
            )
          else
            const Text('Answered', style: TextStyle(fontSize: 12, fontWeight: FontWeight.w700, color: SpottoColors.muted)),
        ],
      ),
    );
  }
}

// ── plan ───────────────────────────────────────────────────────────

BoardPlaceRef _boardRef(GuidePlanPlace p) => BoardPlaceRef(
      name: p.name,
      kind: p.simpleKind,
      subtitle: p.address,
      lat: p.lat,
      lng: p.lng,
      geoPlace: p.asGeoPlace,
      placeId: p.placeId,
    );

/// A day-by-day plan: one day at a time, with its map and stops.
class GuidePlanCard extends ConsumerStatefulWidget {
  final GuidePlan plan;
  final ValueChanged<GuideCalendarAction> onAdd;
  const GuidePlanCard({super.key, required this.plan, required this.onAdd});

  @override
  ConsumerState<GuidePlanCard> createState() => _GuidePlanCardState();
}

class _GuidePlanCardState extends ConsumerState<GuidePlanCard> {
  int _day = 0;

  @override
  Widget build(BuildContext context) {
    final plan = widget.plan;
    final day = plan.days[_day.clamp(0, plan.days.length - 1)];
    final meta = [
      if (plan.startDate != null) _fmtDate(plan.startDate!),
      if (plan.nights > 0) '${plan.nights} ${plan.nights == 1 ? 'night' : 'nights'}',
      if (plan.stay.isNotEmpty) 'Staying at ${plan.stay}',
    ].join(' · ');

    return Container(
      decoration: _cardBox(context),
      clipBehavior: Clip.antiAlias,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(14, 14, 14, 10),
            child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
              Text(plan.title, style: const TextStyle(fontSize: 17, fontWeight: FontWeight.w800, height: 1.2)),
              if (meta.isNotEmpty) ...[
                const SizedBox(height: 3),
                Text(meta, style: const TextStyle(fontSize: 12.5, color: SpottoColors.muted, fontWeight: FontWeight.w600)),
              ],
            ]),
          ),
          if (plan.days.length > 1)
            SizedBox(
              height: 40,
              child: ListView.separated(
                scrollDirection: Axis.horizontal,
                padding: const EdgeInsets.symmetric(horizontal: 14),
                itemCount: plan.days.length,
                separatorBuilder: (_, _) => const SizedBox(width: 8),
                itemBuilder: (_, i) {
                  final selected = i == _day;
                  final d = plan.days[i];
                  return PressScale(
                    scale: 0.95,
                    onTap: () => setState(() => _day = i),
                    child: Container(
                      padding: const EdgeInsets.symmetric(horizontal: 14),
                      alignment: Alignment.center,
                      decoration: BoxDecoration(
                        color: selected ? SpottoColors.ink : SpottoColors.accentTint,
                        borderRadius: BorderRadius.circular(999),
                      ),
                      child: Text(
                        d.date != null ? 'Day ${i + 1} · ${_fmtDate(d.date!)}' : 'Day ${i + 1}',
                        style: TextStyle(
                          fontSize: 12.5,
                          fontWeight: FontWeight.w800,
                          color: selected ? Colors.white : SpottoColors.accent,
                        ),
                      ),
                    ),
                  );
                },
              ),
            ),
          if (day.title.isNotEmpty)
            Padding(
              padding: const EdgeInsets.fromLTRB(14, 10, 14, 0),
              child: Text(day.title, style: const TextStyle(fontSize: 14, fontWeight: FontWeight.w800, color: SpottoColors.accent)),
            ),
          Padding(
            padding: const EdgeInsets.fromLTRB(14, 10, 14, 4),
            child: ClipRRect(
              borderRadius: BorderRadius.circular(14),
              child: BoardPlacesMap(
                key: ValueKey('${plan.id}-$_day'),
                height: 150,
                places: [for (final s in day.stops) _boardRef(s.place)],
                onTapPlace: (i) => openBoardPlace(context, ref, _boardRef(day.stops[i].place)),
              ),
            ),
          ),
          for (var i = 0; i < day.stops.length; i++)
            _StopRow(
              index: i,
              stop: day.stops[i],
              onOpen: () => openBoardPlace(context, ref, _boardRef(day.stops[i].place)),
              onAdd: () => widget.onAdd(GuideCalendarAction.fromPlan(plan, only: [
                GuideCalendarStop(dayIndex: _day, time: day.stops[i].time, place: day.stops[i].place),
              ])),
            ),
          if (plan.tips.isNotEmpty)
            Padding(
              padding: const EdgeInsets.fromLTRB(14, 6, 14, 0),
              child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                for (final t in plan.tips)
                  Padding(
                    padding: const EdgeInsets.only(bottom: 4),
                    child: Row(crossAxisAlignment: CrossAxisAlignment.start, children: [
                      const Padding(
                        padding: EdgeInsets.only(top: 2),
                        child: Icon(Icons.lightbulb_outline, size: 15, color: SpottoColors.sun),
                      ),
                      const SizedBox(width: 6),
                      Expanded(child: Text(t, style: const TextStyle(fontSize: 12.5, height: 1.4))),
                    ]),
                  ),
              ]),
            ),
          Padding(
            padding: const EdgeInsets.fromLTRB(14, 8, 14, 14),
            child: SizedBox(
              width: double.infinity,
              child: FilledButton.icon(
                onPressed: () => widget.onAdd(GuideCalendarAction.fromPlan(plan)),
                icon: const Icon(Icons.event_available, size: 18),
                label: const Text('Add whole trip to calendar'),
                style: FilledButton.styleFrom(
                  backgroundColor: SpottoColors.accent,
                  padding: const EdgeInsets.symmetric(vertical: 12),
                  shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(14)),
                ),
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class _StopRow extends StatelessWidget {
  final int index;
  final GuidePlanStop stop;
  final VoidCallback onOpen;
  final VoidCallback onAdd;
  const _StopRow({required this.index, required this.stop, required this.onOpen, required this.onAdd});

  @override
  Widget build(BuildContext context) {
    final p = stop.place;
    final rating = p.rating == null
        ? ''
        : '★ ${p.rating!.toStringAsFixed(1)}${p.ratingCount > 0 ? ' (${NumberFormat.compact().format(p.ratingCount)})' : ''}';
    final clock = _fmtClock(stop.time);
    return InkWell(
      onTap: onOpen,
      child: Padding(
        padding: const EdgeInsets.fromLTRB(14, 8, 6, 8),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Stack(clipBehavior: Clip.none, children: [
              BoardPlacePhoto(photoUrl: p.photoUrl, kind: p.simpleKind, size: 58),
              Positioned(
                left: -4,
                top: -4,
                child: CircleAvatar(
                  radius: 10,
                  backgroundColor: SpottoColors.ink,
                  child: Text('${index + 1}', style: const TextStyle(fontSize: 10.5, color: Colors.white, fontWeight: FontWeight.w800)),
                ),
              ),
            ]),
            const SizedBox(width: 12),
            Expanded(
              child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                if (clock.isNotEmpty)
                  Text(
                    stop.durationMin > 0 ? '$clock · ${_duration(stop.durationMin)}' : clock,
                    style: const TextStyle(fontSize: 11.5, fontWeight: FontWeight.w800, color: SpottoColors.accent),
                  ),
                Text(p.name, style: const TextStyle(fontSize: 14.5, fontWeight: FontWeight.w800, height: 1.25)),
                if (rating.isNotEmpty || p.kind.isNotEmpty)
                  Padding(
                    padding: const EdgeInsets.only(top: 2),
                    child: Text(
                      [if (rating.isNotEmpty) rating, if (p.kind.isNotEmpty) kindLabelFor(p)].join(' · '),
                      style: const TextStyle(fontSize: 12, color: SpottoColors.muted, fontWeight: FontWeight.w600),
                    ),
                  ),
                if (stop.note.isNotEmpty)
                  Padding(
                    padding: const EdgeInsets.only(top: 4),
                    child: Text(stop.note, style: const TextStyle(fontSize: 12.8, height: 1.4)),
                  ),
              ]),
            ),
            Column(children: [
              IconButton(
                tooltip: 'Add to calendar',
                visualDensity: VisualDensity.compact,
                onPressed: onAdd,
                icon: const Icon(Icons.add_circle_outline, color: SpottoColors.accent),
              ),
              IconButton(
                tooltip: 'Directions',
                visualDensity: VisualDensity.compact,
                onPressed: () => launchUrl(
                  Uri.parse('https://www.google.com/maps/dir/?api=1&destination=${p.lat},${p.lng}'),
                  mode: LaunchMode.externalApplication,
                ),
                icon: const Icon(Icons.directions_outlined, color: SpottoColors.muted),
              ),
            ]),
          ],
        ),
      ),
    );
  }

  static String _duration(int mins) => mins < 60
      ? '$mins min'
      : (mins % 60 == 0 ? '${mins ~/ 60} h' : '${mins ~/ 60} h ${mins % 60} min');
}

/// Category label: catalog places send their category, Google places a kind.
String kindLabelFor(GuidePlanPlace p) => p.placeId != null ? p.kind : kindLabel(p.simpleKind);

// ── calendar ───────────────────────────────────────────────────────

/// "Add to calendar" from a chat message: shows what will be added and,
/// unless the guide was told, lets the traveller pick new vs current trip.
class GuideCalendarCard extends ConsumerWidget {
  final GuideCalendarAction action;
  final String? addedTo; // trip name once added
  final void Function(String mode) onChoose;
  const GuideCalendarCard({super.key, required this.action, required this.addedTo, required this.onChoose});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final current = ref.watch(tripProvider);
    final names = action.stops.map((s) => s.place.name).toList();
    return Container(
      padding: const EdgeInsets.fromLTRB(14, 14, 14, 12),
      decoration: _cardBox(context),
      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Row(children: [
          const Icon(Icons.event_note, size: 18, color: SpottoColors.accent),
          const SizedBox(width: 8),
          Expanded(
            child: Text(
              names.length == 1 ? names.first : '${names.length} places · ${action.title}',
              style: const TextStyle(fontSize: 14, fontWeight: FontWeight.w800),
            ),
          ),
        ]),
        if (names.length > 1) ...[
          const SizedBox(height: 6),
          Text(names.take(6).join(' · ') + (names.length > 6 ? ' …' : ''),
              style: const TextStyle(fontSize: 12.5, color: SpottoColors.muted, height: 1.4)),
        ],
        const SizedBox(height: 12),
        if (addedTo != null)
          Row(children: [
            const Icon(Icons.check_circle, size: 18, color: Color(0xFF16A34A)),
            const SizedBox(width: 6),
            Expanded(child: Text('Added to $addedTo', style: const TextStyle(fontSize: 13, fontWeight: FontWeight.w700))),
            TextButton(onPressed: () => context.go('/t/trip'), child: const Text('Open Trip calendar')),
          ])
        else
          Row(children: [
            Expanded(
              child: OutlinedButton(
                onPressed: () => onChoose('new'),
                style: OutlinedButton.styleFrom(shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(14))),
                child: const Text('New trip'),
              ),
            ),
            const SizedBox(width: 8),
            Expanded(
              child: FilledButton(
                onPressed: () => onChoose('current'),
                style: FilledButton.styleFrom(
                  backgroundColor: SpottoColors.accent,
                  shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(14)),
                ),
                child: Text('Add to ${current.name}', maxLines: 1, overflow: TextOverflow.ellipsis),
              ),
            ),
          ]),
      ]),
    );
  }
}

/// Adds [action]'s stops to the Trip calendar. [mode] "ask" shows a sheet
/// to pick a new trip or the current one. Returns the trip's name, or null
/// if the traveller cancelled.
Future<String?> applyGuideCalendar(BuildContext context, WidgetRef ref, GuideCalendarAction action, String mode) async {
  if (mode == 'ask') {
    final picked = await _askNewOrCurrent(context, ref.read(tripProvider).name);
    if (picked == null) return null;
    mode = picked;
  }
  if (!context.mounted) return null;
  final notifier = ref.read(tripProvider.notifier);
  if (mode == 'new') {
    var start = action.startDate;
    if (start == null) {
      final now = DateTime.now();
      final d = await showDatePicker(
        context: context,
        helpText: 'When does this trip start?',
        firstDate: DateTime(now.year, now.month, now.day),
        lastDate: now.add(const Duration(days: 730)),
        initialDate: now,
      );
      if (d == null) return null;
      start = DateFormat('yyyy-MM-dd').format(d);
    }
    // Keep the trip being replaced in "My trips".
    ref.read(savedTripsProvider.notifier).saveTrip(ref.read(tripProvider));
    final trip = guideNewTrip(action, start);
    notifier.update(trip);
    return trip.name;
  }
  final trip = guideAddStops(ref.read(tripProvider), action);
  notifier.update(trip);
  return trip.name;
}

Future<String?> _askNewOrCurrent(BuildContext context, String currentName) => showModalBottomSheet<String>(
      context: context,
      backgroundColor: Colors.white,
      shape: const RoundedRectangleBorder(borderRadius: BorderRadius.vertical(top: Radius.circular(24))),
      builder: (sheet) => SafeArea(
        child: Padding(
          padding: const EdgeInsets.fromLTRB(20, 18, 20, 12),
          child: Column(mainAxisSize: MainAxisSize.min, crossAxisAlignment: CrossAxisAlignment.start, children: [
            Text('Add to your Trip calendar',
                style: Theme.of(sheet).textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w800)),
            const SizedBox(height: 8),
            ListTile(
              contentPadding: EdgeInsets.zero,
              leading: const CircleAvatar(
                  backgroundColor: SpottoColors.accentTint, child: Icon(Icons.add_rounded, color: SpottoColors.accent)),
              title: const Text('New trip', style: TextStyle(fontWeight: FontWeight.w700)),
              subtitle: const Text('Keeps your current trip in My trips'),
              onTap: () => Navigator.pop(sheet, 'new'),
            ),
            ListTile(
              contentPadding: EdgeInsets.zero,
              leading: const CircleAvatar(
                  backgroundColor: SpottoColors.sand, child: Icon(Icons.luggage_outlined, color: SpottoColors.accent)),
              title: Text('Add to $currentName', style: const TextStyle(fontWeight: FontWeight.w700)),
              subtitle: const Text('Your current trip'),
              onTap: () => Navigator.pop(sheet, 'current'),
            ),
          ]),
        ),
      ),
    );
