// "More" tab: entry points to Menu (food only), Reports, Business profile,
// Account, plus role switch and logout.

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/theme/palette.dart';
import '../../../data/mock/vendor_seed.dart';
import '../../../data/repositories/repositories.dart';
import '../widgets/vendor_common.dart';

class VendorMoreScreen extends ConsumerWidget {
  const VendorMoreScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final text = Theme.of(context).textTheme;
    final business = ref.watch(vendorAuthProvider);
    final isFood = businessTypes
            .where((t) => t.id == business?.typeId)
            .firstOrNull
            ?.food ??
        false;

    Widget row(IconData icon, String title, String sub, VoidCallback onTap,
            {Color? color}) =>
        ListTile(
          leading: Container(
            width: 40,
            height: 40,
            decoration: BoxDecoration(
              color: (color ?? SpottoColors.accent).withValues(alpha: 0.12),
              borderRadius: BorderRadius.circular(12),
            ),
            child: Icon(icon, size: 20, color: color ?? SpottoColors.accent),
          ),
          title: Text(title, style: text.titleSmall),
          subtitle: Text(sub, style: text.bodySmall?.copyWith(color: SpottoColors.muted)),
          trailing: const Icon(Icons.chevron_right, size: 18, color: SpottoColors.muted),
          onTap: onTap,
        );

    return ListView(
      padding: const EdgeInsets.fromLTRB(20, 8, 20, 24),
      children: [
        VendorCard(
          padding: EdgeInsets.zero,
          child: Column(
            children: [
              if (isFood) ...[
                row(Icons.restaurant_menu, 'Menu',
                    'Sections, items and prices', () => context.go('/v/more/menu')),
                const Divider(height: 1),
              ],
              row(Icons.bar_chart, 'Reports', 'Traffic, footfall and trends',
                  () => context.go('/v/more/reports')),
              const Divider(height: 1),
              row(Icons.storefront, 'Business profile',
                  'Type, Wi-Fi, amenities and hours',
                  () => context.go('/v/more/profile')),
              const Divider(height: 1),
              row(Icons.manage_accounts_outlined, 'Account',
                  'Owner, team and subscription', () => context.go('/v/more/account')),
            ],
          ),
        ),
        const SizedBox(height: 16),
        VendorCard(
          padding: EdgeInsets.zero,
          child: Column(
            children: [
              row(Icons.swap_horiz, 'Switch to traveller app',
                  'Back to the role gate', () => context.go('/')),
              const Divider(height: 1),
              row(Icons.logout, 'Log out', 'Sign out of the vendor portal', () {
                ref.read(vendorAuthProvider.notifier).signOut();
                context.go('/v/auth');
              }, color: vendorRed),
            ],
          ),
        ),
      ],
    );
  }
}
