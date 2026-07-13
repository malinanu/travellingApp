// Vendor dashboard — port of VendorDashboard.jsx: KPIs, weekly traffic,
// live redemptions, active offers.

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/theme/palette.dart';
import '../../../core/widgets/charts.dart';
import '../../../core/widgets/spotto_icons.dart';
import '../../../data/mock/vendor_seed.dart';
import '../../../data/repositories/repositories.dart';
import '../widgets/vendor_common.dart';

class VendorDashboardScreen extends ConsumerWidget {
  const VendorDashboardScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final text = Theme.of(context).textTheme;
    final business = ref.watch(vendorAuthProvider);
    final offers = ref.watch(offersProvider);
    final active =
        offers.where((o) => o.status == 'live' || o.status == 'ending').toList();

    return ListView(
      padding: const EdgeInsets.fromLTRB(20, 8, 20, 24),
      children: [
        Row(
          crossAxisAlignment: CrossAxisAlignment.end,
          children: [
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text('Good morning, ${business?.owner.split(' ').first ?? 'there'} 👋',
                      style: text.headlineSmall?.copyWith(fontSize: 22)),
                  const SizedBox(height: 4),
                  Text(
                      "Here's how ${business?.name.split(' ').first ?? 'your spot'} is doing in ${business?.area ?? 'Weligama'} today.",
                      style: text.bodyMedium?.copyWith(color: SpottoColors.muted)),
                ],
              ),
            ),
            const SizedBox(width: 10),
            FilledButton.icon(
              style: FilledButton.styleFrom(
                padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
              ),
              onPressed: () => context.go('/v/offers'),
              icon: const Icon(Icons.add, size: 18),
              label: const Text('New offer'),
            ),
          ],
        ),
        const SizedBox(height: 18),

        // KPIs
        for (final k in kpis) ...[
          _KpiCard(k: k),
          const SizedBox(height: 12),
        ],
        const SizedBox(height: 6),

        // traffic
        VendorCard(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text('Traffic this week', style: text.titleMedium?.copyWith(fontSize: 16)),
              Text('Profile views · redemptions',
                  style: text.bodySmall?.copyWith(color: SpottoColors.muted)),
              const SizedBox(height: 8),
              Row(
                children: [
                  _legendSwatch(SpottoColors.accent.withValues(alpha: 0.25), 'Views', text),
                  const SizedBox(width: 12),
                  _legendSwatch(SpottoColors.accent, 'Redeemed', text),
                ],
              ),
              const SizedBox(height: 10),
              const TrafficChart(),
            ],
          ),
        ),
        const SizedBox(height: 16),

        // live redemptions
        VendorCard(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                children: [
                  Text('Live redemptions',
                      style: text.titleMedium?.copyWith(fontSize: 16)),
                  const Spacer(),
                  Container(
                      width: 8,
                      height: 8,
                      decoration: const BoxDecoration(
                          color: vendorGreen, shape: BoxShape.circle)),
                  const SizedBox(width: 5),
                  Text('Live', style: text.labelSmall?.copyWith(color: vendorGreen)),
                ],
              ),
              const SizedBox(height: 4),
              for (final (i, r) in redemptionsSeed.indexed) ...[
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
                        child: Text(r.name[0],
                            style: const TextStyle(
                                fontSize: 12,
                                fontWeight: FontWeight.w800,
                                color: SpottoColors.ink)),
                      ),
                      const SizedBox(width: 12),
                      Expanded(
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text(r.name,
                                maxLines: 1,
                                overflow: TextOverflow.ellipsis,
                                style: text.titleSmall?.copyWith(fontSize: 13.5)),
                            Text(r.offer,
                                maxLines: 1,
                                overflow: TextOverflow.ellipsis,
                                style: text.bodySmall
                                    ?.copyWith(color: SpottoColors.muted)),
                          ],
                        ),
                      ),
                      Column(
                        crossAxisAlignment: CrossAxisAlignment.end,
                        children: [
                          Row(
                            mainAxisSize: MainAxisSize.min,
                            children: [
                              Container(
                                width: 6,
                                height: 6,
                                decoration: BoxDecoration(
                                    color: styleDot(r.style), shape: BoxShape.circle),
                              ),
                              const SizedBox(width: 4),
                              Text(r.style,
                                  style: TextStyle(
                                      fontSize: 11,
                                      fontWeight: FontWeight.w700,
                                      color: styleDot(r.style))),
                            ],
                          ),
                          Text(r.time,
                              style: text.labelSmall
                                  ?.copyWith(color: SpottoColors.muted)),
                        ],
                      ),
                    ],
                  ),
                ),
              ],
            ],
          ),
        ),
        const SizedBox(height: 16),

        // active offers
        VendorCard(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                children: [
                  Text('Active offers', style: text.titleMedium?.copyWith(fontSize: 16)),
                  const Spacer(),
                  TextButton(
                    onPressed: () => context.go('/v/offers'),
                    child: const Text('Manage →'),
                  ),
                ],
              ),
              for (final (i, o) in active.indexed) ...[
                if (i > 0) const Divider(height: 1),
                Padding(
                  padding: const EdgeInsets.symmetric(vertical: 12),
                  child: Row(
                    children: [
                      Donut(
                        value: o.claimed,
                        max: o.limit,
                        size: 50,
                        stroke: 7,
                        color: o.status == 'ending'
                            ? const Color(0xFFF0682F)
                            : SpottoColors.accent,
                      ),
                      const SizedBox(width: 14),
                      Expanded(
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Row(
                              children: [
                                Flexible(
                                  child: Text(o.title,
                                      maxLines: 1,
                                      overflow: TextOverflow.ellipsis,
                                      style:
                                          text.titleSmall?.copyWith(fontSize: 14)),
                                ),
                                const SizedBox(width: 8),
                                StatusChip(status: o.status),
                              ],
                            ),
                            const SizedBox(height: 2),
                            Text(
                                '${o.claimed}/${o.limit} claimed · ${o.redeemed} redeemed · ends ${o.end}',
                                style: text.bodySmall
                                    ?.copyWith(color: SpottoColors.muted)),
                          ],
                        ),
                      ),
                    ],
                  ),
                ),
              ],
            ],
          ),
        ),
      ],
    );
  }

  Widget _legendSwatch(Color color, String label, TextTheme text) => Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Container(
            width: 10,
            height: 10,
            decoration:
                BoxDecoration(color: color, borderRadius: BorderRadius.circular(3)),
          ),
          const SizedBox(width: 5),
          Text(label, style: text.labelSmall?.copyWith(color: SpottoColors.muted)),
        ],
      );
}

class _KpiCard extends StatelessWidget {
  final Kpi k;
  const _KpiCard({required this.k});

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    final tone = kpiTones[k.tone] ?? kpiTones['teal']!;
    final delta = k.delta;

    return VendorCard(
      child: Row(
        children: [
          Container(
            width: 44,
            height: 44,
            decoration: BoxDecoration(
              color: tone.bg,
              borderRadius: BorderRadius.circular(12),
            ),
            child: Icon(spottoIcon(k.icon), size: 22, color: tone.fg),
          ),
          const SizedBox(width: 14),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(_fmt(k.value),
                    style: text.headlineSmall?.copyWith(fontSize: 26)),
                Text('${k.label} · ${k.hint}',
                    style: text.bodySmall?.copyWith(color: SpottoColors.muted)),
              ],
            ),
          ),
          if (delta != null)
            Container(
              padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
              decoration: BoxDecoration(
                color: delta >= 0 ? vendorGreenBg : vendorRedBg,
                borderRadius: BorderRadius.circular(999),
              ),
              child: Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Icon(delta >= 0 ? Icons.trending_up : Icons.trending_down,
                      size: 13, color: delta >= 0 ? vendorGreen : vendorRed),
                  const SizedBox(width: 3),
                  Text('${delta >= 0 ? '+' : ''}$delta%',
                      style: TextStyle(
                          fontSize: 12.5,
                          fontWeight: FontWeight.w800,
                          color: delta >= 0 ? vendorGreen : vendorRed)),
                ],
              ),
            ),
        ],
      ),
    );
  }

  String _fmt(int v) => v.toString().replaceAllMapped(
      RegExp(r'(\d)(?=(\d{3})+$)'), (m) => '${m[1]},');
}
