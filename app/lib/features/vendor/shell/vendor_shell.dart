// Vendor shell — mobile adaptation of VendorApp.jsx's sidebar: bottom tabs
// (Dashboard / Offers / Validate / Growth / More) + notification bell.

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/theme/palette.dart';
import '../../../core/widgets/sheet_scaffold.dart';
import '../../../core/widgets/spotto_icons.dart';
import '../../../data/repositories/repositories.dart';
import '../widgets/vendor_common.dart';

class VendorShell extends ConsumerWidget {
  final StatefulNavigationShell shell;
  const VendorShell({super.key, required this.shell});

  static const _titles = ['Dashboard', 'Offers', 'Validate', 'Growth', 'More'];

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final text = Theme.of(context).textTheme;
    final business = ref.watch(vendorAuthProvider);
    final unread = ref.watch(vendorNotifsProvider).where((n) => n.unread).length;

    return Scaffold(
      appBar: AppBar(
        backgroundColor: SpottoColors.sand,
        surfaceTintColor: Colors.transparent,
        automaticallyImplyLeading: false,
        titleSpacing: 20,
        title: Row(
          children: [
            Container(
              width: 34,
              height: 34,
              alignment: Alignment.center,
              decoration: BoxDecoration(
                color: SpottoColors.accent,
                borderRadius: BorderRadius.circular(10),
              ),
              child: Text(business?.initials ?? 'DC',
                  style: const TextStyle(
                      fontSize: 13, fontWeight: FontWeight.w800, color: Colors.white)),
            ),
            const SizedBox(width: 10),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(_titles[shell.currentIndex],
                      style: text.titleMedium?.copyWith(fontSize: 17)),
                  Text(business?.name ?? '',
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: text.bodySmall?.copyWith(color: SpottoColors.muted)),
                ],
              ),
            ),
          ],
        ),
        actions: [
          Padding(
            padding: const EdgeInsets.only(right: 12),
            child: Stack(
              children: [
                IconButton(
                  icon: const Icon(Icons.notifications_none, color: SpottoColors.ink),
                  onPressed: () => _showNotifications(context),
                ),
                if (unread > 0)
                  Positioned(
                    right: 8,
                    top: 8,
                    child: Container(
                      width: 16,
                      height: 16,
                      alignment: Alignment.center,
                      decoration: const BoxDecoration(
                          color: Color(0xFFF0682F), shape: BoxShape.circle),
                      child: Text('$unread',
                          style: const TextStyle(
                              fontSize: 9.5,
                              fontWeight: FontWeight.w800,
                              color: Colors.white)),
                    ),
                  ),
              ],
            ),
          ),
        ],
      ),
      body: shell,
      bottomNavigationBar: NavigationBar(
        selectedIndex: shell.currentIndex,
        onDestinationSelected: (i) =>
            shell.goBranch(i, initialLocation: i == shell.currentIndex),
        backgroundColor: Colors.white,
        indicatorColor: SpottoColors.accentTint,
        destinations: const [
          NavigationDestination(
              icon: Icon(Icons.dashboard_outlined),
              selectedIcon: Icon(Icons.dashboard),
              label: 'Dashboard'),
          NavigationDestination(
              icon: Icon(Icons.local_offer_outlined),
              selectedIcon: Icon(Icons.local_offer),
              label: 'Offers'),
          NavigationDestination(
              icon: Icon(Icons.qr_code_scanner), label: 'Validate'),
          NavigationDestination(
              icon: Icon(Icons.trending_up), label: 'Growth'),
          NavigationDestination(
              icon: Icon(Icons.menu), label: 'More'),
        ],
      ),
    );
  }

  void _showNotifications(BuildContext context) {
    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      builder: (_) => Consumer(
        builder: (context, ref, _) {
          final text = Theme.of(context).textTheme;
          final notifs = ref.watch(vendorNotifsProvider);
          return SheetScaffold(
            padding: const EdgeInsets.fromLTRB(20, 4, 20, 24),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  children: [
                    Text('Notifications',
                        style: text.headlineSmall?.copyWith(fontSize: 19)),
                    const Spacer(),
                    TextButton(
                      onPressed: () =>
                          ref.read(vendorNotifsProvider.notifier).markAll(),
                      child: const Text('Mark all read'),
                    ),
                  ],
                ),
                const SizedBox(height: 4),
                Flexible(
                  child: ListView(
                    shrinkWrap: true,
                    children: [
                      for (final n in notifs)
                        InkWell(
                          onTap: () =>
                              ref.read(vendorNotifsProvider.notifier).markOne(n.id),
                          child: Padding(
                            padding: const EdgeInsets.symmetric(vertical: 10),
                            child: Row(
                              crossAxisAlignment: CrossAxisAlignment.start,
                              children: [
                                Container(
                                  width: 36,
                                  height: 36,
                                  decoration: BoxDecoration(
                                    color: n.tone == 'sun'
                                        ? kpiTones['sun']!.bg
                                        : n.tone == 'muted'
                                            ? const Color(0x0F103D3A)
                                            : SpottoColors.accentTint,
                                    borderRadius: BorderRadius.circular(12),
                                  ),
                                  child: Icon(spottoIcon(n.icon),
                                      size: 17,
                                      color: n.tone == 'sun'
                                          ? kpiTones['sun']!.fg
                                          : n.tone == 'muted'
                                              ? SpottoColors.muted
                                              : SpottoColors.accent),
                                ),
                                const SizedBox(width: 12),
                                Expanded(
                                  child: Column(
                                    crossAxisAlignment: CrossAxisAlignment.start,
                                    children: [
                                      Text(n.title,
                                          style: text.titleSmall?.copyWith(
                                              fontWeight: n.unread
                                                  ? FontWeight.w800
                                                  : FontWeight.w600)),
                                      Text(n.sub,
                                          style: text.bodySmall
                                              ?.copyWith(color: SpottoColors.muted)),
                                      Text(n.time,
                                          style: text.labelSmall?.copyWith(
                                              color: SpottoColors.muted,
                                              fontSize: 10.5)),
                                    ],
                                  ),
                                ),
                                if (n.unread)
                                  Container(
                                    width: 8,
                                    height: 8,
                                    margin: const EdgeInsets.only(top: 6),
                                    decoration: const BoxDecoration(
                                        color: SpottoColors.accent,
                                        shape: BoxShape.circle),
                                  ),
                              ],
                            ),
                          ),
                        ),
                    ],
                  ),
                ),
              ],
            ),
          );
        },
      ),
    );
  }
}
