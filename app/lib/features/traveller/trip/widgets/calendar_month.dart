// Month grid — used by the trip calendar view and the new-trip date picker.

import 'package:flutter/material.dart';
import 'package:intl/intl.dart';

import '../../../../core/theme/palette.dart';
import '../../../../core/widgets/press_scale.dart';

class PlannedDay {
  final int index;
  final int stops;
  const PlannedDay({required this.index, required this.stops});
}

/// A month of days. In trip mode pass [planned] + [onPickDay]; in
/// date-picker mode pass [selected] + [onPickDate] (past days disabled).
class CalendarMonth extends StatelessWidget {
  final DateTime month; // first of month
  final VoidCallback onPrevMonth;
  final VoidCallback onNextMonth;
  final Map<String, PlannedDay> planned;
  final ValueChanged<int>? onPickDay;
  final DateTime? selected;
  final ValueChanged<DateTime>? onPickDate;
  final bool compact;

  const CalendarMonth({
    super.key,
    required this.month,
    required this.onPrevMonth,
    required this.onNextMonth,
    this.planned = const {},
    this.onPickDay,
    this.selected,
    this.onPickDate,
    this.compact = false,
  });

  static String keyOf(DateTime d) => '${d.year}-${d.month}-${d.day}';

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    final now = DateTime.now();
    final today = DateTime(now.year, now.month, now.day);
    final startDow = DateTime(month.year, month.month, 1).weekday % 7; // Sun = 0
    final daysInMonth = DateTime(month.year, month.month + 1, 0).day;
    final cellHeight = compact ? 36.0 : 46.0;
    const weekdays = ['S', 'M', 'T', 'W', 'T', 'F', 'S'];

    Widget navButton(IconData icon, VoidCallback onTap) => PressScale(
          scale: 0.9,
          onTap: onTap,
          child: Container(
            width: 32,
            height: 32,
            decoration: BoxDecoration(
              color: Colors.white,
              borderRadius: BorderRadius.circular(10),
              boxShadow: const [
                BoxShadow(color: Color(0x14103D3A), blurRadius: 8, offset: Offset(0, 2))
              ],
            ),
            child: Icon(icon, size: 18, color: SpottoColors.ink),
          ),
        );

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            Expanded(
              child: Text(DateFormat('MMMM yyyy').format(month),
                  style: text.titleMedium?.copyWith(fontSize: 17)),
            ),
            navButton(Icons.chevron_left, onPrevMonth),
            const SizedBox(width: 6),
            navButton(Icons.chevron_right, onNextMonth),
          ],
        ),
        const SizedBox(height: 8),
        Row(
          children: [
            for (final w in weekdays)
              Expanded(
                child: Center(
                  child: Text(w,
                      style: text.labelSmall?.copyWith(color: SpottoColors.muted, fontSize: 11)),
                ),
              ),
          ],
        ),
        const SizedBox(height: 6),
        for (var row = 0; row * 7 < startDow + daysInMonth; row++)
          Padding(
            padding: const EdgeInsets.only(bottom: 6),
            child: Row(
              children: [
                for (var col = 0; col < 7; col++)
                  Expanded(child: _cell(context, row * 7 + col - startDow + 1, daysInMonth, today, cellHeight)),
              ],
            ),
          ),
      ],
    );
  }

  Widget _cell(BuildContext context, int day, int daysInMonth, DateTime today, double height) {
    if (day < 1 || day > daysInMonth) return SizedBox(height: height);
    final date = DateTime(month.year, month.month, day);
    final key = keyOf(date);
    final isToday = key == keyOf(today);
    final plan = planned[key];

    if (plan != null && onPickDay != null) {
      final hasStops = plan.stops > 0;
      final bgColor = hasStops ? SpottoColors.accent : const Color(0xFFFF9800);
      final shadowColor = hasStops ? const Color(0x4D0B6F86) : const Color(0x4DFF9800);

      return Padding(
        padding: const EdgeInsets.symmetric(horizontal: 3),
        child: PressScale(
          scale: 0.95,
          onTap: () => onPickDay!(plan.index),
          child: Container(
            height: height,
            decoration: BoxDecoration(
              color: bgColor,
              borderRadius: BorderRadius.circular(12),
              boxShadow: [
                BoxShadow(color: shadowColor, blurRadius: 10, offset: const Offset(0, 4))
              ],
            ),
            child: Column(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                Text('$day',
                    style: const TextStyle(
                        color: Colors.white, fontSize: 15, fontWeight: FontWeight.w800)),
                Text(
                  hasStops
                      ? '${plan.stops} stop${plan.stops > 1 ? 's' : ''}'
                      : 'Unplanned',
                  style: TextStyle(
                      color: Colors.white.withValues(alpha: 0.9), fontSize: 8.5, fontWeight: FontWeight.w800),
                ),
              ],
            ),
          ),
        ),
      );
    }

    if (onPickDate != null) {
      final isPast = date.isBefore(today);
      final isSel = selected != null && key == keyOf(selected!);
      return Padding(
        padding: const EdgeInsets.symmetric(horizontal: 3),
        child: GestureDetector(
          onTap: isPast ? null : () => onPickDate!(date),
          child: Container(
            height: height,
            alignment: Alignment.center,
            decoration: BoxDecoration(
              color: isSel ? SpottoColors.accent : Colors.transparent,
              borderRadius: BorderRadius.circular(10),
              border: Border.all(
                  color: isToday && !isSel ? SpottoColors.accent : Colors.transparent,
                  width: 1.5),
            ),
            child: Text(
              '$day',
              style: TextStyle(
                fontSize: 13,
                fontWeight: FontWeight.w700,
                color: isSel
                    ? Colors.white
                    : isPast
                        ? SpottoColors.ink.withValues(alpha: 0.22)
                        : SpottoColors.ink,
              ),
            ),
          ),
        ),
      );
    }

    return Container(
      height: height,
      alignment: Alignment.center,
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(12),
        border: Border.all(
            color: isToday ? SpottoColors.accent : Colors.transparent, width: 1.5),
      ),
      child: Text(
        '$day',
        style: TextStyle(
          fontSize: 14,
          fontWeight: FontWeight.w700,
          color: isToday ? SpottoColors.accent : SpottoColors.ink.withValues(alpha: 0.45),
        ),
      ),
    );
  }
}
