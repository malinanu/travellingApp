// Paying for a booking. The traveller pays on the gateway's own hosted
// checkout page (opened in the in-app browser); this app never sees card
// details. Only the gateway's signed notification to the server confirms a
// payment, so the sheet just polls the booking's status until it's
// confirmed or its seat hold runs out.

import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:qr_flutter/qr_flutter.dart';
import 'package:url_launcher/url_launcher.dart';

import '../../../core/api/api_client.dart';
import '../../../core/l10n/app_strings.dart';
import '../../../core/l10n/l10n_provider.dart';
import '../../../core/theme/palette.dart';
import '../../../core/utils/money.dart';
import '../../../data/models/models.dart';
import '../../../data/repositories/repositories.dart';

/// Opens a checkout URL; overridden in tests.
final checkoutLauncherProvider = Provider<Future<bool> Function(Uri)>(
  (ref) => (uri) => launchUrl(uri, mode: LaunchMode.inAppBrowserView),
);

/// How often the sheet re-reads the booking while waiting for payment.
final paymentPollIntervalProvider = Provider<Duration>((ref) => const Duration(seconds: 2));

/// Shows the payment sheet for [booking]. [checkout] is the one returned
/// when the booking was created; without it (paying later from My
/// Bookings) a fresh one is requested.
Future<void> showBookingPaymentSheet(BuildContext context, Booking booking, {CheckoutInfo? checkout}) {
  return showModalBottomSheet<void>(
    context: context,
    isScrollControlled: true,
    backgroundColor: Colors.transparent,
    builder: (_) => BookingPaymentSheet(booking: booking, checkout: checkout),
  );
}

enum _Phase { ready, waiting, confirmed, expired }

class BookingPaymentSheet extends ConsumerStatefulWidget {
  final Booking booking;
  final CheckoutInfo? checkout;
  const BookingPaymentSheet({super.key, required this.booking, this.checkout});

  @override
  ConsumerState<BookingPaymentSheet> createState() => _BookingPaymentSheetState();
}

class _BookingPaymentSheetState extends ConsumerState<BookingPaymentSheet> with WidgetsBindingObserver {
  late Booking _booking = widget.booking;
  CheckoutInfo? _checkout;
  _Phase _phase = _Phase.ready;
  bool _opening = false;
  Timer? _poll;

  @override
  void initState() {
    super.initState();
    _checkout = widget.checkout;
    WidgetsBinding.instance.addObserver(this);
    if (_booking.status == 'confirmed') _phase = _Phase.confirmed;
  }

  @override
  void dispose() {
    _poll?.cancel();
    WidgetsBinding.instance.removeObserver(this);
    super.dispose();
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    // Back from the browser: check straight away rather than on the next tick.
    if (state == AppLifecycleState.resumed && _phase == _Phase.waiting) _check();
  }

  Future<void> _pay() async {
    final messenger = ScaffoldMessenger.maybeOf(context);
    final l10n = ref.read(l10nProvider);
    setState(() => _opening = true);
    try {
      final co = _checkout ?? await ref.read(bookingsProvider.notifier).checkout(_booking.id);
      _checkout = co;
      final opened = await ref.read(checkoutLauncherProvider)(Uri.parse(co.url));
      if (!opened) throw StateError('not opened');
      if (!mounted) return;
      setState(() => _phase = _Phase.waiting);
      _poll?.cancel();
      _poll = Timer.periodic(ref.read(paymentPollIntervalProvider), (_) => _check());
    } catch (e) {
      messenger?.showSnackBar(
        SnackBar(content: Text(e is StateError ? l10n.couldNotOpenPayment : ApiClient.errorMessage(e))),
      );
    } finally {
      if (mounted) setState(() => _opening = false);
    }
  }

  Future<void> _check() async {
    try {
      final b = await ref.read(bookingsProvider.notifier).refresh(_booking.id);
      if (b == null || !mounted) return;
      setState(() {
        _booking = b;
        if (b.status == 'confirmed' || b.status == 'completed') {
          _phase = _Phase.confirmed;
        } else if (b.status == 'expired' || b.status == 'cancelled' || (b.awaitingPayment && !b.holdLive)) {
          _phase = _Phase.expired;
        }
      });
      if (_phase == _Phase.confirmed || _phase == _Phase.expired) _poll?.cancel();
    } catch (_) {
      // Offline for a moment: keep polling.
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = ref.watch(l10nProvider);
    final text = Theme.of(context).textTheme;
    return Container(
      padding: EdgeInsets.fromLTRB(20, 16, 20, 20 + MediaQuery.of(context).padding.bottom),
      decoration: const BoxDecoration(
        color: SpottoColors.sand,
        borderRadius: BorderRadius.vertical(top: Radius.circular(24)),
      ),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Center(
            child: Container(
              width: 40,
              height: 4,
              decoration: BoxDecoration(color: SpottoColors.line, borderRadius: BorderRadius.circular(2)),
            ),
          ),
          const SizedBox(height: 14),
          if (_phase == _Phase.confirmed) ...[
            BookingQrCard(booking: _booking),
            const SizedBox(height: 14),
            FilledButton(
              key: const Key('payment_done_button'),
              onPressed: () => Navigator.of(context).pop(),
              child: Text(l10n.done),
            ),
          ] else ...[
            Text(l10n.payTitle, style: text.titleLarge?.copyWith(fontWeight: FontWeight.w800)),
            const SizedBox(height: 4),
            Text(_booking.serviceName, style: text.bodyMedium?.copyWith(fontWeight: FontWeight.w700)),
            const SizedBox(height: 12),
            Row(
              children: [
                Text(l10n.totalLabel, style: text.bodyMedium?.copyWith(color: SpottoColors.muted)),
                const Spacer(),
                Text(formatLkr(_booking.amountMinor), style: text.titleMedium?.copyWith(fontWeight: FontWeight.w800)),
              ],
            ),
            if (_checkout case final co? when co.chargeCurrency != 'LKR' && co.chargeAmountMinor > 0) ...[
              const SizedBox(height: 4),
              Text(
                l10n.chargedAs(formatMoney(co.chargeAmountMinor, co.chargeCurrency)),
                key: const Key('payment_charge_note'),
                style: text.bodySmall?.copyWith(color: SpottoColors.muted),
              ),
            ],
            if (_holdText(l10n) case final hold?) ...[
              const SizedBox(height: 6),
              Text(hold, style: text.bodySmall?.copyWith(color: SpottoColors.muted)),
            ],
            const SizedBox(height: 14),
            if (_phase == _Phase.expired)
              Text(l10n.payExpired, key: const Key('payment_expired_text'), style: text.bodyMedium)
            else if (_phase == _Phase.waiting) ...[
              Row(
                children: [
                  const SizedBox(width: 18, height: 18, child: CircularProgressIndicator(strokeWidth: 2)),
                  const SizedBox(width: 10),
                  Expanded(child: Text(l10n.payWaiting, style: text.bodyMedium)),
                ],
              ),
              const SizedBox(height: 12),
              OutlinedButton(onPressed: _opening ? null : _pay, child: Text(l10n.payAgain)),
            ] else
              FilledButton.icon(
                key: const Key('payment_pay_button'),
                onPressed: _opening ? null : _pay,
                icon: const Icon(Icons.lock_outline, size: 18),
                label: Text(l10n.paySecurely),
              ),
            const SizedBox(height: 10),
            Text(
              l10n.payCardNote,
              textAlign: TextAlign.center,
              style: text.bodySmall?.copyWith(color: SpottoColors.muted),
            ),
          ],
        ],
      ),
    );
  }

  String? _holdText(AppStrings l10n) {
    final until = DateTime.tryParse(_booking.holdExpiresAt)?.toLocal();
    if (until == null || _phase == _Phase.expired) return null;
    final t = '${until.hour.toString().padLeft(2, '0')}:${until.minute.toString().padLeft(2, '0')}';
    return l10n.payHeldUntil(t);
  }
}

/// The confirmed booking's QR code and BOOK- code, shown at check-in.
class BookingQrCard extends ConsumerWidget {
  final Booking booking;
  final VoidCallback? onDone;
  const BookingQrCard({super.key, required this.booking, this.onDone});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = ref.watch(l10nProvider);
    return Container(
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(color: SpottoColors.ink, borderRadius: BorderRadius.circular(16)),
      child: Row(
        children: [
          Container(
            padding: const EdgeInsets.all(6),
            decoration: BoxDecoration(color: Colors.white, borderRadius: BorderRadius.circular(8)),
            child: QrImageView(data: booking.code, size: 72, gapless: false),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  l10n.bookedShowAtCheckIn,
                  style: const TextStyle(fontSize: 11.5, fontWeight: FontWeight.w700, color: Colors.white70),
                ),
                const SizedBox(height: 3),
                Text(
                  booking.code,
                  style: const TextStyle(fontSize: 17, fontWeight: FontWeight.w800, color: Colors.white, letterSpacing: 0.5),
                ),
                const SizedBox(height: 3),
                Text(
                  [
                    l10n.partyOf(booking.partySize),
                    if (booking.amountMinor > 0) l10n.paidAmount(formatLkr(booking.amountMinor)),
                  ].join(' · '),
                  style: const TextStyle(fontSize: 12.5, fontWeight: FontWeight.w600, color: Colors.white70),
                ),
              ],
            ),
          ),
          if (onDone != null)
            IconButton(
              key: const Key('booking_qr_done'),
              onPressed: onDone,
              icon: const Icon(Icons.close, color: Colors.white70, size: 20),
              tooltip: l10n.done,
            ),
        ],
      ),
    );
  }
}
