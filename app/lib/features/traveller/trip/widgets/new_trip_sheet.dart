// New-trip sheet: name, start & end date pickers with automatic duration calculation.

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';

import '../../../../core/theme/palette.dart';
import '../../../../core/widgets/press_scale.dart';
import '../../../../core/widgets/sheet_scaffold.dart';
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
  final _name = TextEditingController(text: 'Spotto Trip');
  late DateTime _startDate;
  late DateTime _endDate;
  int _activePicker = 0; // 0 = closed, 1 = start date, 2 = end date
  late DateTime _month;

  @override
  void initState() {
    super.initState();
    final now = DateTime.now();
    _startDate = DateTime(now.year, now.month, now.day);
    _endDate = _startDate.add(const Duration(days: 6)); // Default 7-day trip
    _month = DateTime(_startDate.year, _startDate.month, 1);
  }

  @override
  void dispose() {
    _name.dispose();
    super.dispose();
  }

  int get _totalDays {
    final diff = _endDate.difference(_startDate).inDays + 1;
    return diff > 0 ? diff : 1;
  }


  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;

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
            Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                Text('Trip dates', style: text.labelMedium?.copyWith(color: SpottoColors.inkSoft)),
                Container(
                  padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
                  decoration: BoxDecoration(
                    color: SpottoColors.accentTint,
                    borderRadius: BorderRadius.circular(10),
                  ),
                  child: Text(
                    '$_totalDays ${_totalDays == 1 ? 'day' : 'days'} total',
                    style: const TextStyle(fontSize: 11.5, fontWeight: FontWeight.bold, color: SpottoColors.accent),
                  ),
                ),
              ],
            ),
            const SizedBox(height: 8),

            // Start & End Date Card Selectors
            Row(
              children: [
                Expanded(
                  child: PressScale(
                    onTap: () => setState(() => _activePicker = _activePicker == 1 ? 0 : 1),
                    child: Container(
                      padding: const EdgeInsets.all(12),
                      decoration: BoxDecoration(
                        color: Colors.white,
                        borderRadius: BorderRadius.circular(16),
                        border: Border.all(
                            color: _activePicker == 1 ? SpottoColors.accent : SpottoColors.line, width: 1.5),
                      ),
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          const Text('START DATE', style: TextStyle(fontSize: 9.5, fontWeight: FontWeight.bold, color: SpottoColors.muted, letterSpacing: 0.5)),
                          const SizedBox(height: 4),
                          Row(
                            children: [
                              const Icon(Icons.calendar_today_rounded, size: 15, color: SpottoColors.accent),
                              const SizedBox(width: 6),
                              Expanded(
                                child: Text(
                                  DateFormat('MMM d, yyyy').format(_startDate),
                                  style: text.titleSmall?.copyWith(fontSize: 13.5),
                                  maxLines: 1,
                                  overflow: TextOverflow.ellipsis,
                                ),
                              ),
                            ],
                          ),
                        ],
                      ),
                    ),
                  ),
                ),
                const SizedBox(width: 10),
                Expanded(
                  child: PressScale(
                    onTap: () => setState(() => _activePicker = _activePicker == 2 ? 0 : 2),
                    child: Container(
                      padding: const EdgeInsets.all(12),
                      decoration: BoxDecoration(
                        color: Colors.white,
                        borderRadius: BorderRadius.circular(16),
                        border: Border.all(
                            color: _activePicker == 2 ? SpottoColors.accent : SpottoColors.line, width: 1.5),
                      ),
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          const Text('END DATE', style: TextStyle(fontSize: 9.5, fontWeight: FontWeight.bold, color: SpottoColors.muted, letterSpacing: 0.5)),
                          const SizedBox(height: 4),
                          Row(
                            children: [
                              const Icon(Icons.event_available, size: 15, color: SpottoColors.accent),
                              const SizedBox(width: 6),
                              Expanded(
                                child: Text(
                                  DateFormat('MMM d, yyyy').format(_endDate),
                                  style: text.titleSmall?.copyWith(fontSize: 13.5),
                                  maxLines: 1,
                                  overflow: TextOverflow.ellipsis,
                                ),
                              ),
                            ],
                          ),
                        ],
                      ),
                    ),
                  ),
                ),
              ],
            ),

            // Inline Calendar Picker for active date selection
            if (_activePicker != 0) ...[
              const SizedBox(height: 10),
              Container(
                padding: const EdgeInsets.all(12),
                decoration: BoxDecoration(
                  color: Colors.white,
                  borderRadius: BorderRadius.circular(16),
                  border: Border.all(color: SpottoColors.accent, width: 1.5),
                  boxShadow: const [
                    BoxShadow(color: Color(0x33103D3A), blurRadius: 24, offset: Offset(0, 10), spreadRadius: -10)
                  ],
                ),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Padding(
                      padding: const EdgeInsets.only(bottom: 8),
                      child: Text(
                        _activePicker == 1 ? 'Select Start Date' : 'Select End Date',
                        style: const TextStyle(fontSize: 12, fontWeight: FontWeight.bold, color: SpottoColors.accent),
                      ),
                    ),
                    CalendarMonth(
                      month: _month,
                      compact: true,
                      selected: _activePicker == 1 ? _startDate : _endDate,
                      onPickDate: (d) => setState(() {
                        final picked = DateTime(d.year, d.month, d.day);
                        if (_activePicker == 1) {
                          _startDate = picked;
                          if (_endDate.isBefore(_startDate)) {
                            _endDate = _startDate;
                          }
                        } else {
                          _endDate = picked;
                          if (_endDate.isBefore(_startDate)) {
                            _startDate = _endDate;
                          }
                        }
                        _activePicker = 0;
                      }),
                      onPrevMonth: () =>
                          setState(() => _month = DateTime(_month.year, _month.month - 1, 1)),
                      onNextMonth: () =>
                          setState(() => _month = DateTime(_month.year, _month.month + 1, 1)),
                    ),
                  ],
                ),
              ),
            ],

            const SizedBox(height: 24),
            SizedBox(
              width: double.infinity,
              child: FilledButton.icon(
                onPressed: () {
                  final startDateIso =
                      '${_startDate.year}-${_startDate.month.toString().padLeft(2, '0')}-${_startDate.day.toString().padLeft(2, '0')}';
                  ref.read(tripProvider.notifier).createTrip(_name.text, _totalDays, startDateIso);
                  Navigator.of(context).pop(startDateIso);
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
