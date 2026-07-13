// Account — port of AccountProfile.jsx: owner details, team, plan, logout.

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/theme/palette.dart';
import '../../../data/repositories/repositories.dart';
import '../widgets/vendor_common.dart';

class VendorAccountScreen extends ConsumerWidget {
  const VendorAccountScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final text = Theme.of(context).textTheme;
    final business = ref.watch(vendorAuthProvider);

    return Scaffold(
      appBar:
          AppBar(title: const Text('Account'), backgroundColor: SpottoColors.sand),
      body: ListView(
        padding: const EdgeInsets.fromLTRB(20, 8, 20, 24),
        children: [
          // owner card
          VendorCard(
            child: Row(
              children: [
                Container(
                  width: 56,
                  height: 56,
                  alignment: Alignment.center,
                  decoration: BoxDecoration(
                    gradient: const LinearGradient(
                      begin: Alignment.topLeft,
                      end: Alignment.bottomRight,
                      colors: [SpottoColors.accent, SpottoColors.accent2],
                    ),
                    borderRadius: BorderRadius.circular(18),
                  ),
                  child: Text(business?.initials ?? 'DC',
                      style: const TextStyle(
                          fontSize: 18,
                          fontWeight: FontWeight.w800,
                          color: Colors.white)),
                ),
                const SizedBox(width: 14),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(business?.owner ?? 'Owner', style: text.titleMedium),
                      Text('${business?.name ?? ''} · ${business?.area ?? ''}',
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: text.bodySmall?.copyWith(color: SpottoColors.muted)),
                      const SizedBox(height: 4),
                      Container(
                        padding: const EdgeInsets.symmetric(
                            horizontal: 8, vertical: 3),
                        decoration: BoxDecoration(
                          color: vendorGreenBg,
                          borderRadius: BorderRadius.circular(999),
                        ),
                        child: const Text('VERIFIED BUSINESS',
                            style: TextStyle(
                                fontSize: 9.5,
                                fontWeight: FontWeight.w800,
                                letterSpacing: 0.4,
                                color: vendorGreen)),
                      ),
                    ],
                  ),
                ),
              ],
            ),
          ),
          const SizedBox(height: 16),

          // team
          VendorCard(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  children: [
                    Text('Team', style: text.titleMedium?.copyWith(fontSize: 16)),
                    const Spacer(),
                    Text('2 of 3 seats',
                        style: text.labelSmall?.copyWith(color: SpottoColors.muted)),
                  ],
                ),
                const SizedBox(height: 8),
                for (final (i, m) in const [
                  (name: 'Tharindu Perera', role: 'Owner', initial: 'T'),
                  (name: 'Ishara Silva', role: 'Counter staff', initial: 'I'),
                ].indexed) ...[
                  if (i > 0) const Divider(height: 1),
                  Padding(
                    padding: const EdgeInsets.symmetric(vertical: 10),
                    child: Row(
                      children: [
                        Container(
                          width: 36,
                          height: 36,
                          alignment: Alignment.center,
                          decoration: const BoxDecoration(
                              color: SpottoColors.sand, shape: BoxShape.circle),
                          child: Text(m.initial,
                              style: const TextStyle(
                                  fontSize: 13,
                                  fontWeight: FontWeight.w800,
                                  color: SpottoColors.ink)),
                        ),
                        const SizedBox(width: 12),
                        Expanded(child: Text(m.name, style: text.titleSmall)),
                        Container(
                          padding: const EdgeInsets.symmetric(
                              horizontal: 8, vertical: 3),
                          decoration: BoxDecoration(
                            color: m.role == 'Owner'
                                ? SpottoColors.accentTint
                                : const Color(0x0F103D3A),
                            borderRadius: BorderRadius.circular(999),
                          ),
                          child: Text(m.role.toUpperCase(),
                              style: TextStyle(
                                  fontSize: 9.5,
                                  fontWeight: FontWeight.w800,
                                  color: m.role == 'Owner'
                                      ? SpottoColors.accent
                                      : SpottoColors.muted)),
                        ),
                      ],
                    ),
                  ),
                ],
                const SizedBox(height: 4),
                SizedBox(
                  width: double.infinity,
                  child: OutlinedButton.icon(
                    onPressed: () => ScaffoldMessenger.of(context).showSnackBar(
                        const SnackBar(content: Text('Invite sent (demo)'))),
                    icon: const Icon(Icons.person_add_outlined, size: 17),
                    label: const Text('Invite a teammate'),
                  ),
                ),
              ],
            ),
          ),
          const SizedBox(height: 16),

          // plan
          Container(
            padding: const EdgeInsets.all(18),
            decoration: BoxDecoration(
              gradient: const LinearGradient(
                begin: Alignment.topLeft,
                end: Alignment.bottomRight,
                colors: [SpottoColors.deepTeal, SpottoColors.accent],
              ),
              borderRadius: BorderRadius.circular(20),
            ),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  children: [
                    const Icon(Icons.workspace_premium,
                        size: 20, color: SpottoColors.sun),
                    const SizedBox(width: 8),
                    Text('Premium plan',
                        style: text.titleMedium?.copyWith(color: Colors.white)),
                    const Spacer(),
                    Text('\$24/mo',
                        style: text.titleMedium?.copyWith(color: Colors.white)),
                  ],
                ),
                const SizedBox(height: 6),
                Text(
                    'Unlimited offers, growth tools, reports and priority placement. Renews Aug 1, 2026.',
                    style: text.bodySmall?.copyWith(color: Colors.white70)),
              ],
            ),
          ),
          const SizedBox(height: 16),

          VendorCard(
            padding: EdgeInsets.zero,
            child: Column(
              children: [
                ListTile(
                  leading: const Icon(Icons.swap_horiz, color: SpottoColors.accent),
                  title: Text('Switch to traveller app', style: text.titleSmall),
                  onTap: () => context.go('/'),
                ),
                const Divider(height: 1),
                ListTile(
                  leading: const Icon(Icons.logout, color: vendorRed),
                  title: Text('Log out',
                      style: text.titleSmall?.copyWith(color: vendorRed)),
                  onTap: () {
                    ref.read(vendorAuthProvider.notifier).signOut();
                    context.go('/v/auth');
                  },
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}
