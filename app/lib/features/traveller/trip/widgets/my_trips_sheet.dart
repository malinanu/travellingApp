// My-trips sheet: lists all saved trips, switch active trip, edit or delete trip.

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';

import '../../../../core/theme/palette.dart';
import '../../../../core/widgets/press_scale.dart';
import '../../../../core/widgets/sheet_scaffold.dart';
import '../../../../data/mock/trip_catalog.dart';
import '../../../../data/repositories/repositories.dart';
import 'new_trip_sheet.dart';

Future<void> showMyTripsSheet(BuildContext context, WidgetRef ref, {VoidCallback? onTripOpened}) {
  return showModalBottomSheet(
    context: context,
    isScrollControlled: true,
    builder: (_) => _MyTripsSheet(onTripOpened: onTripOpened),
  );
}

class _MyTripsSheet extends ConsumerWidget {
  final VoidCallback? onTripOpened;
  const _MyTripsSheet({this.onTripOpened});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final text = Theme.of(context).textTheme;
    final activeTrip = ref.watch(tripProvider);
    final allTrips = ref.watch(savedTripsProvider);

    return SheetScaffold(
      padding: const EdgeInsets.fromLTRB(20, 4, 20, 24),
      child: SizedBox(
        height: MediaQuery.of(context).size.height * 0.7,
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text('MY PLANNED TRIPS', style: text.labelSmall?.copyWith(color: SpottoColors.accent, letterSpacing: 1)),
                      Text('${allTrips.length} ${allTrips.length == 1 ? 'Trip' : 'Trips'} Saved', style: text.headlineSmall?.copyWith(fontSize: 19)),
                    ],
                  ),
                ),
                FilledButton.icon(
                  style: FilledButton.styleFrom(
                    backgroundColor: SpottoColors.accent,
                    shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(14)),
                  ),
                  onPressed: () async {
                    Navigator.pop(context);
                    await showNewTripSheet(context, ref);
                  },
                  icon: const Icon(Icons.add, size: 16),
                  label: const Text('New Trip'),
                ),
              ],
            ),
            const SizedBox(height: 16),
            Expanded(
              child: ListView.builder(
                itemCount: allTrips.length,
                itemBuilder: (context, index) {
                  final trip = allTrips[index];
                  final isActive = trip.code == activeTrip.code;
                  final totalStops = trip.days.fold<int>(0, (sum, d) => sum + d.stops.length);
                  final firstDate = tripDayDate(trip.startDate, 0);
                  final lastDate = tripDayDate(trip.startDate, trip.days.length - 1);
                  final dateRangeStr = '${DateFormat('MMM d').format(firstDate)} – ${DateFormat('MMM d, yyyy').format(lastDate)}';

                  return Padding(
                    padding: const EdgeInsets.only(bottom: 12),
                    child: PressScale(
                      scale: 0.98,
                      onTap: () {
                        if (!isActive) {
                          ref.read(tripProvider.notifier).switchActiveTrip(trip.code);
                        }
                        Navigator.pop(context);
                        onTripOpened?.call();
                      },
                      child: Container(
                        padding: const EdgeInsets.all(16),
                        decoration: BoxDecoration(
                          color: Colors.white,
                          borderRadius: BorderRadius.circular(20),
                          border: Border.all(
                            color: isActive ? SpottoColors.accent : SpottoColors.line,
                            width: isActive ? 2.0 : 1.2,
                          ),
                          boxShadow: [
                            BoxShadow(
                              color: isActive ? const Color(0x2E0B6F86) : const Color(0x10103D3A),
                              blurRadius: isActive ? 12 : 6,
                              offset: const Offset(0, 4),
                            ),
                          ],
                        ),
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Row(
                              children: [
                                Expanded(
                                  child: Text(
                                    trip.name,
                                    style: text.titleMedium?.copyWith(
                                      fontWeight: FontWeight.w800,
                                      color: isActive ? SpottoColors.accent : SpottoColors.ink,
                                    ),
                                    maxLines: 1,
                                    overflow: TextOverflow.ellipsis,
                                  ),
                                ),
                                if (isActive)
                                  Container(
                                    padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
                                    decoration: BoxDecoration(
                                      color: SpottoColors.accent,
                                      borderRadius: BorderRadius.circular(8),
                                    ),
                                    child: const Text('ACTIVE', style: TextStyle(fontSize: 9.5, color: Colors.white, fontWeight: FontWeight.bold, letterSpacing: 0.5)),
                                  ),
                              ],
                            ),
                            const SizedBox(height: 6),
                            Row(
                              children: [
                                const Icon(Icons.calendar_today_rounded, size: 13, color: SpottoColors.muted),
                                const SizedBox(width: 6),
                                Text(dateRangeStr, style: text.bodySmall?.copyWith(color: SpottoColors.inkSoft, fontWeight: FontWeight.w600)),
                              ],
                            ),
                            const SizedBox(height: 10),
                            Row(
                              children: [
                                _badge('${trip.days.length} Days', Icons.date_range),
                                const SizedBox(width: 8),
                                _badge('$totalStops ${totalStops == 1 ? 'Stop' : 'Stops'}', Icons.place_outlined),
                                const Spacer(),
                                IconButton(
                                  icon: const Icon(Icons.edit_outlined, size: 18, color: SpottoColors.accent),
                                  onPressed: () {
                                    _showEditTripDialog(context, ref, trip);
                                  },
                                ),
                                IconButton(
                                  icon: const Icon(Icons.delete_outline, size: 18, color: Colors.redAccent),
                                  onPressed: () {
                                    _confirmDelete(context, ref, trip);
                                  },
                                ),
                              ],
                            ),
                          ],
                        ),
                      ),
                    ),
                  );
                },
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _badge(String label, IconData icon) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      decoration: BoxDecoration(
        color: SpottoColors.sand,
        borderRadius: BorderRadius.circular(8),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(icon, size: 12, color: SpottoColors.inkSoft),
          const SizedBox(width: 4),
          Text(label, style: const TextStyle(fontSize: 11, color: SpottoColors.ink, fontWeight: FontWeight.w700)),
        ],
      ),
    );
  }

  void _confirmDelete(BuildContext context, WidgetRef ref, Trip trip) {
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(20)),
        title: Text('Delete "${trip.name}"?'),
        content: const Text('Are you sure you want to remove this trip itinerary?'),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx),
            child: const Text('Cancel'),
          ),
          FilledButton(
            style: FilledButton.styleFrom(backgroundColor: Colors.redAccent),
            onPressed: () {
              ref.read(tripProvider.notifier).deleteTrip(trip.code);
              Navigator.pop(ctx);
            },
            child: const Text('Delete'),
          ),
        ],
      ),
    );
  }

  void _showEditTripDialog(BuildContext context, WidgetRef ref, Trip trip) {
    final nameCtrl = TextEditingController(text: trip.name);
    DateTime startDate = DateTime.parse(trip.startDate);
    int days = trip.days.length;

    showDialog(
      context: context,
      builder: (ctx) => StatefulBuilder(
        builder: (context, setState) {
          final endDate = startDate.add(Duration(days: days - 1));
          return AlertDialog(
            shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(20)),
            title: const Text('Edit Trip Details'),
            content: SingleChildScrollView(
              child: Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  const Text('TRIP NAME', style: TextStyle(fontSize: 10, fontWeight: FontWeight.bold, color: SpottoColors.muted, letterSpacing: 0.5)),
                  const SizedBox(height: 4),
                  TextField(
                    controller: nameCtrl,
                    decoration: const InputDecoration(
                      prefixIcon: Icon(Icons.route, size: 18, color: SpottoColors.muted),
                    ),
                  ),
                  const SizedBox(height: 14),
                  const Text('START DATE', style: TextStyle(fontSize: 10, fontWeight: FontWeight.bold, color: SpottoColors.muted, letterSpacing: 0.5)),
                  const SizedBox(height: 4),
                  OutlinedButton.icon(
                    style: OutlinedButton.styleFrom(
                      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
                      side: const BorderSide(color: SpottoColors.line),
                    ),
                    onPressed: () async {
                      final picked = await showDatePicker(
                        context: context,
                        initialDate: startDate,
                        firstDate: DateTime(2024),
                        lastDate: DateTime(2030),
                      );
                      if (picked != null) {
                        setState(() => startDate = picked);
                      }
                    },
                    icon: const Icon(Icons.calendar_today_rounded, size: 15, color: SpottoColors.accent),
                    label: Text(
                      DateFormat('EEE, MMM d, yyyy').format(startDate),
                      style: const TextStyle(fontSize: 13, color: SpottoColors.ink),
                    ),
                  ),
                  const SizedBox(height: 14),
                  Row(
                    mainAxisAlignment: MainAxisAlignment.spaceBetween,
                    children: [
                      const Text('TOTAL DURATION', style: TextStyle(fontSize: 10, fontWeight: FontWeight.bold, color: SpottoColors.muted, letterSpacing: 0.5)),
                      Text('$days ${days == 1 ? 'day' : 'days'}', style: const TextStyle(fontSize: 12, fontWeight: FontWeight.bold, color: SpottoColors.accent)),
                    ],
                  ),
                  Row(
                    children: [
                      IconButton(
                        icon: const Icon(Icons.remove_circle_outline, color: SpottoColors.accent),
                        onPressed: days > 1 ? () => setState(() => days--) : null,
                      ),
                      Text('$days', style: const TextStyle(fontSize: 16, fontWeight: FontWeight.bold)),
                      IconButton(
                        icon: const Icon(Icons.add_circle_outline, color: SpottoColors.accent),
                        onPressed: () => setState(() => days++),
                      ),
                    ],
                  ),
                  const SizedBox(height: 4),
                  Text('End Date: ${DateFormat('EEE, MMM d, yyyy').format(endDate)}', style: const TextStyle(fontSize: 11, color: SpottoColors.muted, fontWeight: FontWeight.w600)),
                ],
              ),
            ),
            actions: [
              TextButton(
                onPressed: () => Navigator.pop(ctx),
                child: const Text('Cancel'),
              ),
              FilledButton(
                style: FilledButton.styleFrom(backgroundColor: SpottoColors.accent),
                onPressed: () {
                  final startDateIso =
                      '${startDate.year}-${startDate.month.toString().padLeft(2, '0')}-${startDate.day.toString().padLeft(2, '0')}';
                  ref.read(tripProvider.notifier).updateTripDetails(trip.code, nameCtrl.text, startDateIso, days);
                  Navigator.pop(ctx);
                },
                child: const Text('Save'),
              ),
            ],
          );
        },
      ),
    );
  }
}
