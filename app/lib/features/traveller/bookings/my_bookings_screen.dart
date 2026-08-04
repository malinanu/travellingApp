// My Bookings — every real booking the traveller has made (session/table/
// stay), newest-upcoming-first, with cancel for still-confirmed future
// bookings. Top-level route (like Weather/Profile), reached from Profile's
// Account section — see app/router.dart and profile_screen.dart's profRow.

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:qr_flutter/qr_flutter.dart';

import '../../../core/api/api_client.dart';
import '../../../core/theme/palette.dart';
import '../../../data/models/models.dart';
import '../../../data/repositories/repositories.dart';
import '../profile/profile_screen.dart' show circleButton;

// Matches profile_screen.dart's private _danger convention (not exported).
const _danger = Color(0xFFE11D48);

class MyBookingsScreen extends ConsumerWidget {
  const MyBookingsScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final text = Theme.of(context).textTheme;
    final bookings = [...ref.watch(bookingsProvider)]
      ..sort((a, b) {
        final aAt = DateTime.tryParse(a.startAt) ?? DateTime(0);
        final bAt = DateTime.tryParse(b.startAt) ?? DateTime(0);
        return aAt.compareTo(bAt);
      });

    return Scaffold(
      backgroundColor: SpottoColors.sand,
      body: SafeArea(
        child: Column(
          children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 8, 16, 8),
              child: Row(
                children: [
                  circleButton(Icons.chevron_left, () => context.pop()),
                  Expanded(
                    child: Center(
                      child: Text('My Bookings', style: text.titleMedium?.copyWith(fontSize: 17)),
                    ),
                  ),
                  const SizedBox(width: 40), // balances the back button
                ],
              ),
            ),
            Expanded(
              child: bookings.isEmpty
                  ? Center(
                      child: Padding(
                        padding: const EdgeInsets.all(32),
                        child: Column(
                          mainAxisSize: MainAxisSize.min,
                          children: [
                            const Icon(Icons.event_busy_outlined, size: 40, color: SpottoColors.muted),
                            const SizedBox(height: 10),
                            Text('No bookings yet',
                                style: text.titleSmall?.copyWith(color: SpottoColors.muted)),
                            const SizedBox(height: 4),
                            const Text(
                              'Book a surf lesson, a table, or a room from any business page.',
                              textAlign: TextAlign.center,
                              style: TextStyle(fontSize: 13, fontWeight: FontWeight.w600, color: SpottoColors.muted),
                            ),
                          ],
                        ),
                      ),
                    )
                  : ListView.separated(
                      padding: const EdgeInsets.fromLTRB(16, 4, 16, 32),
                      itemCount: bookings.length,
                      separatorBuilder: (_, _) => const SizedBox(height: 10),
                      itemBuilder: (context, i) => _BookingTile(booking: bookings[i]),
                    ),
            ),
          ],
        ),
      ),
    );
  }
}

class _BookingTile extends ConsumerStatefulWidget {
  final Booking booking;
  const _BookingTile({required this.booking});

  @override
  ConsumerState<_BookingTile> createState() => _BookingTileState();
}

class _BookingTileState extends ConsumerState<_BookingTile> {
  bool _cancelling = false;

  @override
  Widget build(BuildContext context) {
    final b = widget.booking;
    final start = DateTime.tryParse(b.startAt);
    final end = DateTime.tryParse(b.endAt);
    final when = b.type == 'stay' && start != null && end != null
        ? '${_fmtDate(start)} → ${_fmtDate(end)}'
        : start != null
            ? _fmtDateTime(start)
            : '';
    final canCancel = b.status == 'confirmed' && start != null && start.isAfter(DateTime.now());
    final tone = _statusTone(b.status);

    return Container(
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(color: Colors.white, borderRadius: BorderRadius.circular(16)),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Expanded(
                child: Text(b.serviceName.isNotEmpty ? b.serviceName : 'Booking',
                    style: const TextStyle(fontSize: 15, fontWeight: FontWeight.w800),
                    overflow: TextOverflow.ellipsis),
              ),
              Container(
                padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
                decoration: BoxDecoration(color: tone.$1, borderRadius: BorderRadius.circular(999)),
                child: Text(b.status.replaceAll('_', ' '),
                    style: TextStyle(fontSize: 10.5, fontWeight: FontWeight.w800, color: tone.$2)),
              ),
            ],
          ),
          if (b.vendorName.isNotEmpty)
            Text(b.vendorName, style: const TextStyle(fontSize: 12.5, fontWeight: FontWeight.w600, color: SpottoColors.muted)),
          const SizedBox(height: 8),
          Row(
            children: [
              const Icon(Icons.calendar_today_outlined, size: 13, color: SpottoColors.muted),
              const SizedBox(width: 5),
              Expanded(child: Text(when, style: const TextStyle(fontSize: 12.5, fontWeight: FontWeight.w600))),
              const Icon(Icons.people_outline, size: 13, color: SpottoColors.muted),
              const SizedBox(width: 3),
              Text('${b.partySize}', style: const TextStyle(fontSize: 12.5, fontWeight: FontWeight.w700)),
            ],
          ),
          if (b.status == 'confirmed') ...[
            const SizedBox(height: 10),
            Row(
              children: [
                Container(
                  padding: const EdgeInsets.all(4),
                  decoration: BoxDecoration(color: Colors.white, border: Border.all(color: SpottoColors.line), borderRadius: BorderRadius.circular(6)),
                  child: QrImageView(data: b.code, size: 36, gapless: false),
                ),
                const SizedBox(width: 8),
                Text(b.code, style: const TextStyle(fontSize: 13, fontWeight: FontWeight.w800, letterSpacing: 0.4)),
                const Spacer(),
                if (canCancel)
                  TextButton(
                    onPressed: _cancelling ? null : _cancel,
                    child: Text(_cancelling ? 'Cancelling…' : 'Cancel',
                        style: const TextStyle(fontSize: 12.5, fontWeight: FontWeight.w800, color: _danger)),
                  ),
              ],
            ),
          ],
        ],
      ),
    );
  }

  Future<void> _cancel() async {
    final messenger = ScaffoldMessenger.of(context);
    setState(() => _cancelling = true);
    try {
      await ref.read(bookingsProvider.notifier).cancel(widget.booking.id);
    } catch (e) {
      messenger.showSnackBar(SnackBar(content: Text(ApiClient.errorMessage(e))));
    } finally {
      if (mounted) setState(() => _cancelling = false);
    }
  }
}

(Color, Color) _statusTone(String status) => switch (status) {
      'confirmed' => (SpottoColors.accent.withValues(alpha: 0.12), SpottoColors.accent),
      'completed' => (const Color(0xFF16A34A).withValues(alpha: 0.14), const Color(0xFF15803D)),
      'no_show' => (_danger.withValues(alpha: 0.14), _danger),
      _ => (SpottoColors.line.withValues(alpha: 0.5), SpottoColors.muted), // cancelled
    };

String _fmtDate(DateTime d) => '${d.month}/${d.day}/${d.year}';
String _fmtDateTime(DateTime d) {
  final hour12 = d.hour % 12 == 0 ? 12 : d.hour % 12;
  final ampm = d.hour >= 12 ? 'PM' : 'AM';
  return '${d.month}/${d.day} · $hour12:${d.minute.toString().padLeft(2, '0')} $ampm';
}
