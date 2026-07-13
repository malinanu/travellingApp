// New-trip sheet: name, start date (inline calendar), length.

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';

import '../../../../core/theme/palette.dart';
import '../../../../core/widgets/press_scale.dart';
import '../../../../core/widgets/sheet_scaffold.dart';
import '../../../../data/mock/trip_catalog.dart';
import '../../../../data/repositories/repositories.dart';
import 'calendar_month.dart';

/// Returns the picked start date (so the caller can jump the calendar there).
Future<String?> showNewTripSheet(BuildContext context, WidgetRef ref) {
  return showModalBottomSheet<String>(
    context: context,
    isScrollControlled: true,
    builder: (_) => const _NewTripSheet(),
  );
}

class _NewTripSheet extends ConsumerStatefulWidget {
  const _NewTripSheet();

  @override
  ConsumerState<_NewTripSheet> createState() => _NewTripSheetState();
}

class _NewTripSheetState extends ConsumerState<_NewTripSheet> {
  final _name = TextEditingController(text: 'My Weligama Trip');
  int _days = 2;
  late String _date = todayIso();
  bool _pickerOpen = false;
  late DateTime _month;

  @override
  void initState() {
    super.initState();
    final d = DateTime.parse(_date);
    _month = DateTime(d.year, d.month, 1);
  }

  @override
  void dispose() {
    _name.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    final sel = DateTime.parse(_date);
    const opts = [
      (n: 1, label: 'One day', sub: 'A single timeline', icon: Icons.wb_sunny),
      (n: 2, label: 'Weekend', sub: '2 days', icon: Icons.calendar_month),
      (n: 3, label: 'Long stay', sub: '3 days', icon: Icons.date_range),
    ];

    return SheetScaffold(
      padding: EdgeInsets.fromLTRB(20, 4, 20, 24 + MediaQuery.of(context).viewInsets.bottom),
      child: SingleChildScrollView(
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          mainAxisSize: MainAxisSize.min,
          children: [
            Text('Plan a new trip', style: text.headlineSmall?.copyWith(fontSize: 19)),
            const SizedBox(height: 16),
            Text('Trip name', style: text.labelMedium?.copyWith(color: SpottoColors.inkSoft)),
            const SizedBox(height: 6),
            TextField(
              controller: _name,
              decoration: const InputDecoration(
                prefixIcon: Icon(Icons.route, size: 18, color: SpottoColors.muted),
              ),
            ),
            const SizedBox(height: 16),
            Text('Start date', style: text.labelMedium?.copyWith(color: SpottoColors.inkSoft)),
            const SizedBox(height: 6),
            PressScale(
              onTap: () => setState(() => _pickerOpen = !_pickerOpen),
              child: Container(
                height: 50,
                padding: const EdgeInsets.symmetric(horizontal: 14),
                decoration: BoxDecoration(
                  color: Colors.white,
                  borderRadius: BorderRadius.circular(16),
                  border: Border.all(
                      color: _pickerOpen ? SpottoColors.accent : SpottoColors.line, width: 1.5),
                ),
                child: Row(
                  children: [
                    Icon(Icons.calendar_month,
                        size: 17,
                        color: _pickerOpen ? SpottoColors.accent : SpottoColors.muted),
                    const SizedBox(width: 10),
                    Expanded(
                      child: Text(DateFormat('EEE, MMM d, yyyy').format(sel),
                          style: text.titleSmall),
                    ),
                    AnimatedRotation(
                      turns: _pickerOpen ? 0.5 : 0,
                      duration: const Duration(milliseconds: 200),
                      child: const Icon(Icons.keyboard_arrow_down,
                          size: 18, color: SpottoColors.muted),
                    ),
                  ],
                ),
              ),
            ),
            if (_pickerOpen) ...[
              const SizedBox(height: 8),
              Container(
                padding: const EdgeInsets.all(12),
                decoration: BoxDecoration(
                  color: Colors.white,
                  borderRadius: BorderRadius.circular(16),
                  border: Border.all(color: SpottoColors.line, width: 1.5),
                  boxShadow: const [
                    BoxShadow(color: Color(0x33103D3A), blurRadius: 30, offset: Offset(0, 12), spreadRadius: -14)
                  ],
                ),
                child: CalendarMonth(
                  month: _month,
                  compact: true,
                  selected: sel,
                  onPickDate: (d) => setState(() {
                    _date =
                        '${d.year}-${d.month.toString().padLeft(2, '0')}-${d.day.toString().padLeft(2, '0')}';
                    _pickerOpen = false;
                  }),
                  onPrevMonth: () =>
                      setState(() => _month = DateTime(_month.year, _month.month - 1, 1)),
                  onNextMonth: () =>
                      setState(() => _month = DateTime(_month.year, _month.month + 1, 1)),
                ),
              ),
            ],
            const SizedBox(height: 16),
            Text('How long?', style: text.labelMedium?.copyWith(color: SpottoColors.inkSoft)),
            const SizedBox(height: 8),
            Row(
              children: [
                for (final (i, o) in opts.indexed) ...[
                  if (i > 0) const SizedBox(width: 10),
                  Expanded(
                    child: PressScale(
                      scale: 0.97,
                      onTap: () => setState(() => _days = o.n),
                      child: Container(
                        padding: const EdgeInsets.symmetric(vertical: 12),
                        decoration: BoxDecoration(
                          color: _days == o.n ? SpottoColors.accent : Colors.white,
                          borderRadius: BorderRadius.circular(16),
                          border: Border.all(
                              color: _days == o.n ? SpottoColors.accent : SpottoColors.line,
                              width: 1.5),
                        ),
                        child: Column(
                          children: [
                            Icon(o.icon,
                                size: 20,
                                color: _days == o.n ? Colors.white : SpottoColors.ink),
                            const SizedBox(height: 6),
                            Text(o.label,
                                style: TextStyle(
                                    fontSize: 13,
                                    fontWeight: FontWeight.w800,
                                    color: _days == o.n ? Colors.white : SpottoColors.ink)),
                            Text(o.sub,
                                style: TextStyle(
                                    fontSize: 11,
                                    fontWeight: FontWeight.w700,
                                    color: _days == o.n
                                        ? Colors.white70
                                        : SpottoColors.muted)),
                          ],
                        ),
                      ),
                    ),
                  ),
                ],
              ],
            ),
            const SizedBox(height: 20),
            SizedBox(
              width: double.infinity,
              child: FilledButton.icon(
                onPressed: () {
                  ref.read(tripProvider.notifier).createTrip(_name.text, _days, _date);
                  Navigator.of(context).pop(_date);
                },
                icon: const Text('Start planning'),
                label: const Icon(Icons.arrow_forward, size: 18),
              ),
            ),
          ],
        ),
      ),
    );
  }
}
