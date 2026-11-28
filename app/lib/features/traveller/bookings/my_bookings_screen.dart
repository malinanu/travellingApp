// My Bookings — every booking the traveller has made: upcoming first
// (soonest at the top, including ones still waiting for payment), then past
// ones (newest first). Top-level route (like Weather/Profile), reached from
// Profile's Account section — see app/router.dart and profile_screen.dart.

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:intl/intl.dart';
import 'package:qr_flutter/qr_flutter.dart';

import '../../../core/api/api_client.dart';
import '../../../core/l10n/app_strings.dart';
import '../../../core/l10n/l10n_provider.dart';
import '../../../core/theme/palette.dart';
import '../../../core/utils/money.dart';
import '../../../data/models/models.dart';
import '../../../data/repositories/repositories.dart';
import '../profile/profile_screen.dart' show circleButton;
import 'booking_payment.dart';

// Matches profile_screen.dart's private _danger convention (not exported).
const _danger = Color(0xFFE11D48);

/// Splits bookings into upcoming (soonest first) and past (newest first).
({List<Booking> upcoming, List<Booking> past}) splitBookings(List<Booking> all) {
  DateTime at(Booking b) => DateTime.tryParse(b.startAt) ?? DateTime(0);
  final upcoming = [for (final b in all) if (b.isUpcoming) b]..sort((a, b) => at(a).compareTo(at(b)));
  final past = [for (final b in all) if (!b.isUpcoming) b]..sort((a, b) => at(b).compareTo(at(a)));
  return (upcoming: upcoming, past: past);
}

class MyBookingsScreen extends ConsumerWidget {
  const MyBookingsScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = ref.watch(l10nProvider);
    final text = Theme.of(context).textTheme;
    final split = splitBookings(ref.watch(bookingsProvider));

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
                    child: Center(child: Text(l10n.myBookings, style: text.titleMedium?.copyWith(fontSize: 17))),
                  ),
                  const SizedBox(width: 40), // balances the back button
                ],
              ),
            ),
            Expanded(
              child: split.upcoming.isEmpty && split.past.isEmpty
                  ? Center(
                      child: Padding(
                        padding: const EdgeInsets.all(32),
                        child: Column(
                          mainAxisSize: MainAxisSize.min,
                          children: [
                            const Icon(Icons.event_busy_outlined, size: 40, color: SpottoColors.muted),
                            const SizedBox(height: 10),
                            Text(l10n.noBookingsYet, style: text.titleSmall?.copyWith(color: SpottoColors.muted)),
                            const SizedBox(height: 4),
                            Text(
                              l10n.noBookingsHint,
                              textAlign: TextAlign.center,
                              style: const TextStyle(fontSize: 13, fontWeight: FontWeight.w600, color: SpottoColors.muted),
                            ),
                          ],
                        ),
                      ),
                    )
                  : RefreshIndicator(
                      onRefresh: () => ref.read(bookingsProvider.notifier).reload(),
                      child: ListView(
                        padding: const EdgeInsets.fromLTRB(16, 4, 16, 32),
                        children: [
                          if (split.upcoming.isNotEmpty) ...[
                            _SectionHeader(l10n.upcomingSection),
                            for (final b in split.upcoming) _BookingTile(key: ValueKey(b.id), booking: b),
                          ],
                          if (split.past.isNotEmpty) ...[
                            _SectionHeader(l10n.pastSection),
                            for (final b in split.past) _BookingTile(key: ValueKey(b.id), booking: b),
                          ],
                        ],
                      ),
                    ),
            ),
          ],
        ),
      ),
    );
  }
}

class _SectionHeader extends StatelessWidget {
  final String label;
  const _SectionHeader(this.label);

  @override
  Widget build(BuildContext context) => Padding(
        padding: const EdgeInsets.fromLTRB(4, 12, 4, 8),
        child: Text(
          label.toUpperCase(),
          style: const TextStyle(fontSize: 11.5, fontWeight: FontWeight.w800, color: SpottoColors.muted, letterSpacing: 0.6),
        ),
      );
}

String bookingStatusLabel(AppStrings l10n, String status) => switch (status) {
      'awaiting_payment' => l10n.statusAwaitingPayment,
      'confirmed' => l10n.statusConfirmed,
      'completed' => l10n.statusCompleted,
      'no_show' => l10n.statusNoShow,
      'expired' => l10n.statusExpired,
      _ => l10n.statusCancelled,
    };

class _BookingTile extends ConsumerStatefulWidget {
  final Booking booking;
  const _BookingTile({super.key, required this.booking});

  @override
  ConsumerState<_BookingTile> createState() => _BookingTileState();
}

class _BookingTileState extends ConsumerState<_BookingTile> {
  bool _cancelling = false;

  @override
  Widget build(BuildContext context) {
    final l10n = ref.watch(l10nProvider);
    final b = widget.booking;
    final start = DateTime.tryParse(b.startAt)?.toLocal();
    final end = DateTime.tryParse(b.endAt)?.toLocal();
    final when = b.type == 'stay' && start != null && end != null
        ? '${DateFormat.yMMMd().format(start)} → ${DateFormat.yMMMd().format(end)}'
        : start != null
        ? DateFormat('EEE d MMM · HH:mm').format(start)
        : '';
    final canCancel = (b.status == 'confirmed' || b.holdLive) && start != null && start.isAfter(DateTime.now());
    final tone = _statusTone(b.status);
    final money = [
      if (b.amountMinor > 0 && b.paymentStatus != 'pending' && b.paymentStatus != 'failed')
        l10n.paidAmount(b.moneyLabel(b.amountMinor)),
      if (b.refundedMinor > 0) l10n.refundedAmount(b.moneyLabel(b.refundedMinor)),
    ].join(' · ');

    return Container(
      margin: const EdgeInsets.only(bottom: 10),
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(color: Colors.white, borderRadius: BorderRadius.circular(16)),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Expanded(
                child: Text(
                  b.serviceName.isNotEmpty ? b.serviceName : l10n.bookHeading,
                  style: const TextStyle(fontSize: 15, fontWeight: FontWeight.w800),
                  overflow: TextOverflow.ellipsis,
                ),
              ),
              Container(
                padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
                decoration: BoxDecoration(color: tone.$1, borderRadius: BorderRadius.circular(999)),
                child: Text(
                  bookingStatusLabel(l10n, b.status),
                  style: TextStyle(fontSize: 10.5, fontWeight: FontWeight.w800, color: tone.$2),
                ),
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
          if (money.isNotEmpty) ...[
            const SizedBox(height: 4),
            Text(money, style: const TextStyle(fontSize: 12, fontWeight: FontWeight.w600, color: SpottoColors.muted)),
          ],
          if (b.status == 'confirmed') ...[
            const SizedBox(height: 10),
            Row(
              children: [
                Container(
                  padding: const EdgeInsets.all(4),
                  decoration: BoxDecoration(
                    color: Colors.white,
                    border: Border.all(color: SpottoColors.line),
                    borderRadius: BorderRadius.circular(6),
                  ),
                  child: QrImageView(data: b.code, size: 36, gapless: false),
                ),
                const SizedBox(width: 8),
                Text(b.code, style: const TextStyle(fontSize: 13, fontWeight: FontWeight.w800, letterSpacing: 0.4)),
                const Spacer(),
                if (canCancel) _cancelButton(l10n),
              ],
            ),
          ] else if (b.holdLive) ...[
            const SizedBox(height: 10),
            Row(
              children: [
                FilledButton(
                  key: Key('complete_payment_${b.id}'),
                  onPressed: () => showBookingPaymentSheet(context, b),
                  child: Text(l10n.completePayment),
                ),
                const Spacer(),
                if (canCancel) _cancelButton(l10n),
              ],
            ),
          ],
        ],
      ),
    );
  }

  Widget _cancelButton(AppStrings l10n) => TextButton(
        key: Key('cancel_booking_${widget.booking.id}'),
        onPressed: _cancelling ? null : _cancel,
        child: Text(
          _cancelling ? l10n.cancellingBooking : l10n.cancelBookingAction,
          style: const TextStyle(fontSize: 12.5, fontWeight: FontWeight.w800, color: _danger),
        ),
      );

  Future<void> _cancel() async {
    final messenger = ScaffoldMessenger.of(context);
    final l10n = ref.read(l10nProvider);
    final b = widget.booking;
    // Paid bookings show what comes back before the traveller commits.
    String message = '';
    if (b.isPaid && b.paymentStatus == 'paid') {
      try {
        final p = await ref.read(cancelPreviewProvider(b.id).future);
        message = p.refundMinor <= 0
            ? l10n.refundNone
            : p.fullRefund
            ? l10n.refundFullBack(formatMoney(p.gatewayRefundMinor, p.gatewayCurrency))
            : l10n.refundPartBack(formatMoney(p.gatewayRefundMinor, p.gatewayCurrency), formatMoney(p.gatewayPaidMinor, p.gatewayCurrency));
      } catch (e) {
        messenger.showSnackBar(SnackBar(content: Text(ApiClient.errorMessage(e))));
        return;
      }
    }
    if (!mounted) return;
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(l10n.cancelBookingTitle),
        content: message.isEmpty ? null : Text(message, key: const Key('cancel_refund_message')),
        actions: [
          TextButton(onPressed: () => Navigator.of(ctx).pop(false), child: Text(l10n.keepBooking)),
          FilledButton(
            key: const Key('confirm_cancel_booking'),
            style: FilledButton.styleFrom(backgroundColor: _danger),
            onPressed: () => Navigator.of(ctx).pop(true),
            child: Text(l10n.confirmCancelBooking),
          ),
        ],
      ),
    );
    if (ok != true || !mounted) return;
    setState(() => _cancelling = true);
    try {
      final res = await ref.read(bookingsProvider.notifier).cancel(b.id);
      messenger.showSnackBar(
        SnackBar(
          content: Text(
            res.refundMinor > 0
                ? l10n.cancelledWithRefund(formatMoney(res.gatewayRefundMinor, res.gatewayCurrency))
                : l10n.bookingCancelled,
          ),
        ),
      );
    } catch (e) {
      messenger.showSnackBar(SnackBar(content: Text(ApiClient.errorMessage(e))));
    } finally {
      if (mounted) setState(() => _cancelling = false);
    }
  }
}

(Color, Color) _statusTone(String status) => switch (status) {
      'confirmed' => (SpottoColors.accent.withValues(alpha: 0.12), SpottoColors.accent),
      'awaiting_payment' => (SpottoColors.sun.withValues(alpha: 0.18), const Color(0xFF8A5A00)),
      'completed' => (const Color(0xFF16A34A).withValues(alpha: 0.14), const Color(0xFF15803D)),
      'no_show' => (_danger.withValues(alpha: 0.14), _danger),
      _ => (SpottoColors.line.withValues(alpha: 0.5), SpottoColors.muted), // cancelled, expired
    };
