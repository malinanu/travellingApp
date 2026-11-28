// Saved payment: there are no saved cards. Paid bookings are paid on the
// payment gateway's own secure checkout page, so card details never reach
// Spotto's servers or this app (see bookings/booking_payment.dart). Cards
// may come back as gateway tokens once the gateway supports it.

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/l10n/l10n_provider.dart';
import '../../../core/theme/palette.dart';

class PaymentMethodsScreen extends ConsumerWidget {
  const PaymentMethodsScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = ref.watch(l10nProvider);
    final text = Theme.of(context).textTheme;
    return Scaffold(
      backgroundColor: SpottoColors.sand,
      appBar: AppBar(
        title: Text(l10n.savedPayment),
        backgroundColor: SpottoColors.sand,
        elevation: 0,
      ),
      body: ListView(
        padding: const EdgeInsets.all(20),
        children: [
          Container(
            padding: const EdgeInsets.all(18),
            decoration: BoxDecoration(color: Colors.white, borderRadius: BorderRadius.circular(16)),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                const Icon(Icons.lock_outline, color: SpottoColors.accent, size: 28),
                const SizedBox(height: 10),
                Text(l10n.payAtCheckoutTitle, style: text.titleMedium?.copyWith(fontWeight: FontWeight.w800)),
                const SizedBox(height: 6),
                Text(l10n.payAtCheckoutBody, style: text.bodyMedium?.copyWith(color: SpottoColors.muted)),
              ],
            ),
          ),
        ],
      ),
    );
  }
}
