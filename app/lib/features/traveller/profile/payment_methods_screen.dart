// Payment Methods Management Screen
// Allows travellers and vendors to manage saved payment cards (Visa, Mastercard, AMEX),
// set default card, and add new payment cards cleanly.

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/api/api_client.dart';
import '../../../core/theme/palette.dart';
import '../../../core/widgets/spotto_card.dart';
import '../../../data/models/models.dart';
import '../../../data/repositories/repositories.dart';

class PaymentMethodsScreen extends ConsumerWidget {
  const PaymentMethodsScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final text = Theme.of(context).textTheme;
    final asyncCards = ref.watch(paymentMethodsProvider);

    return Scaffold(
      backgroundColor: SpottoColors.sand,
      appBar: AppBar(
        title: const Text('Payment Methods', style: TextStyle(fontWeight: FontWeight.bold)),
        backgroundColor: Colors.transparent,
        elevation: 0,
      ),
      floatingActionButton: FloatingActionButton.extended(
        backgroundColor: SpottoColors.accent,
        icon: const Icon(Icons.add_card, color: Colors.white),
        label: const Text('Add Card', style: TextStyle(color: Colors.white, fontWeight: FontWeight.bold)),
        onPressed: () => _showAddCardSheet(context, ref),
      ),
      body: SafeArea(
        child: asyncCards.when(
          data: (cards) {
            if (cards.isEmpty) {
              return Center(
                child: Padding(
                  padding: const EdgeInsets.all(32),
                  child: Column(
                    mainAxisAlignment: MainAxisAlignment.center,
                    children: [
                      const Icon(Icons.credit_card_off_outlined, size: 64, color: SpottoColors.muted),
                      const SizedBox(height: 16),
                      Text('No Saved Payment Cards', style: text.titleMedium?.copyWith(fontWeight: FontWeight.bold)),
                      const SizedBox(height: 8),
                      Text('Add a Visa, Mastercard, or AMEX to easily pay for vendor bookings & experiences.',
                          textAlign: TextAlign.center, style: text.bodySmall?.copyWith(color: SpottoColors.muted)),
                      const SizedBox(height: 24),
                      FilledButton.icon(
                        icon: const Icon(Icons.add_card, size: 18),
                        label: const Text('Add Payment Card'),
                        style: FilledButton.styleFrom(backgroundColor: SpottoColors.accent),
                        onPressed: () => _showAddCardSheet(context, ref),
                      ),
                    ],
                  ),
                ),
              );
            }

            return ListView.separated(
              padding: const EdgeInsets.fromLTRB(16, 16, 16, 80),
              itemCount: cards.length,
              separatorBuilder: (_, __) => const SizedBox(height: 12),
              itemBuilder: (context, i) {
                final card = cards[i];
                return SpottoCard(
                  child: ListTile(
                    contentPadding: const EdgeInsets.all(12),
                    leading: Container(
                      padding: const EdgeInsets.all(10),
                      decoration: BoxDecoration(
                        color: SpottoColors.accentTint,
                        borderRadius: BorderRadius.circular(12),
                      ),
                      child: Icon(_cardBrandIcon(card.brand), color: SpottoColors.accent),
                    ),
                    title: Row(
                      children: [
                        Text('${card.brand} •••• ${card.last4}',
                            style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 16)),
                        if (card.isDefault) ...[
                          const SizedBox(width: 8),
                          Container(
                            padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
                            decoration: BoxDecoration(
                              color: SpottoColors.sun.withValues(alpha: 0.2),
                              borderRadius: BorderRadius.circular(8),
                            ),
                            child: const Text('DEFAULT',
                                style: TextStyle(fontSize: 10, fontWeight: FontWeight.bold, color: SpottoColors.ink)),
                          ),
                        ],
                      ],
                    ),
                    subtitle: Padding(
                      padding: const EdgeInsets.only(top: 4),
                      child: Text('Expires ${card.expMonth.toString().padLeft(2, '0')}/${card.expYear} • ${card.holderName}',
                          style: const TextStyle(fontSize: 12, color: SpottoColors.muted)),
                    ),
                    trailing: PopupMenuButton<String>(
                      onSelected: (val) async {
                        if (val == 'default') {
                          await ref.read(apiProvider).dio.post('/me/payment-methods/${card.id}/default');
                          ref.invalidate(paymentMethodsProvider);
                        } else if (val == 'delete') {
                          await ref.read(apiProvider).dio.delete('/me/payment-methods/${card.id}');
                          ref.invalidate(paymentMethodsProvider);
                        }
                      },
                      itemBuilder: (context) => [
                        if (!card.isDefault)
                          const PopupMenuItem(value: 'default', child: Text('Set as Default')),
                        const PopupMenuItem(value: 'delete', child: Text('Remove Card', style: TextStyle(color: Colors.red))),
                      ],
                    ),
                  ),
                );
              },
            );
          },
          loading: () => const Center(child: CircularProgressIndicator(color: SpottoColors.accent)),
          error: (err, _) => Center(child: Text('Error loading payment methods: $err')),
        ),
      ),
    );
  }

  static IconData _cardBrandIcon(String brand) {
    switch (brand.toLowerCase()) {
      case 'visa':
        return Icons.credit_card;
      case 'mastercard':
        return Icons.payment;
      case 'american express':
        return Icons.credit_score;
      default:
        return Icons.credit_card_outlined;
    }
  }

  static void _showAddCardSheet(BuildContext context, WidgetRef ref) {
    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      backgroundColor: Colors.white,
      shape: const RoundedRectangleBorder(borderRadius: BorderRadius.vertical(top: Radius.circular(20))),
      builder: (context) => _AddCardSheet(ref: ref),
    );
  }
}

class _AddCardSheet extends StatefulWidget {
  const _AddCardSheet({required this.ref});
  final WidgetRef ref;

  @override
  State<_AddCardSheet> createState() => _AddCardSheetState();
}

class _AddCardSheetState extends State<_AddCardSheet> {
  final _numberCtrl = TextEditingController();
  final _holderCtrl = TextEditingController();
  final _expCtrl = TextEditingController();
  final _cvvCtrl = TextEditingController();
  bool _isDefault = false;
  bool _isSaving = false;

  @override
  void dispose() {
    _numberCtrl.dispose();
    _holderCtrl.dispose();
    _expCtrl.dispose();
    _cvvCtrl.dispose();
    super.dispose();
  }

  Future<void> _saveCard() async {
    final numStr = _numberCtrl.text.replaceAll(' ', '');
    final expStr = _expCtrl.text.trim();
    if (numStr.length < 13 || !expStr.contains('/')) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Please enter a valid card number and expiration (MM/YY)')),
      );
      return;
    }
    final parts = expStr.split('/');
    final m = int.tryParse(parts[0]) ?? 1;
    var y = int.tryParse(parts[1]) ?? 26;
    if (y < 100) y += 2000;

    setState(() => _isSaving = true);
    try {
      await widget.ref.read(apiProvider).dio.post('/me/payment-methods', data: {
        'cardNumber': numStr,
        'expMonth': m,
        'expYear': y,
        'holderName': _holderCtrl.text.trim().isEmpty ? 'Card Holder' : _holderCtrl.text.trim(),
        'isDefault': _isDefault,
      });
      widget.ref.invalidate(paymentMethodsProvider);
      if (mounted) Navigator.pop(context);
    } catch (err) {
      if (mounted) {
        setState(() => _isSaving = false);
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Error saving card: $err')),
        );
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: EdgeInsets.only(
        top: 20,
        left: 20,
        right: 20,
        bottom: MediaQuery.of(context).viewInsets.bottom + 20,
      ),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              const Text('Add Payment Card', style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold)),
              IconButton(onPressed: () => Navigator.pop(context), icon: const Icon(Icons.close)),
            ],
          ),
          const SizedBox(height: 16),
          TextField(
            controller: _numberCtrl,
            keyboardType: TextInputType.number,
            decoration: const InputDecoration(
              labelText: 'Card Number',
              hintText: '4532 •••• •••• 8910',
              prefixIcon: Icon(Icons.credit_card),
            ),
          ),
          const SizedBox(height: 12),
          Row(
            children: [
              Expanded(
                child: TextField(
                  controller: _expCtrl,
                  keyboardType: TextInputType.datetime,
                  decoration: const InputDecoration(
                    labelText: 'Expiry',
                    hintText: 'MM/YY',
                  ),
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: TextField(
                  controller: _cvvCtrl,
                  keyboardType: TextInputType.number,
                  obscureText: true,
                  decoration: const InputDecoration(
                    labelText: 'CVV',
                    hintText: '123',
                  ),
                ),
              ),
            ],
          ),
          const SizedBox(height: 12),
          TextField(
            controller: _holderCtrl,
            decoration: const InputDecoration(
              labelText: 'Cardholder Name',
              hintText: 'e.g. Alex Morgan',
            ),
          ),
          const SizedBox(height: 12),
          SwitchListTile(
            title: const Text('Set as Default Payment Method'),
            value: _isDefault,
            activeColor: SpottoColors.accent,
            onChanged: (v) => setState(() => _isDefault = v),
          ),
          const SizedBox(height: 16),
          SizedBox(
            width: double.infinity,
            height: 48,
            child: FilledButton(
              onPressed: _isSaving ? null : _saveCard,
              style: FilledButton.styleFrom(backgroundColor: SpottoColors.accent),
              child: _isSaving
                  ? const SizedBox(width: 20, height: 20, child: CircularProgressIndicator(color: Colors.white, strokeWidth: 2))
                  : const Text('Save Card', style: TextStyle(fontWeight: FontWeight.bold, fontSize: 16)),
            ),
          ),
        ],
      ),
    );
  }
}
