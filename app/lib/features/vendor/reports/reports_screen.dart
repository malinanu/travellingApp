// Reports — port of Reports.jsx: stat pills computed from the seed data,
// trend vs category, weekly traffic and popular times.

import 'dart:math' as math;

import 'package:flutter/material.dart';

import '../../../core/theme/palette.dart';
import '../../../core/widgets/charts.dart';
import '../../../data/mock/vendor_seed.dart';
import '../widgets/vendor_common.dart';

class VendorReportsScreen extends StatelessWidget {
  const VendorReportsScreen({super.key});

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;

    final totalViews = weekTraffic.fold(0, (a, d) => a + d.views);
    final totalRedeems = weekTraffic.fold(0, (a, d) => a + d.redemptions);
    final peakHour = popularTimes.reduce((a, b) => a.v >= b.v ? a : b).h;
    final youNow = trend.last.you;
    final catNow = trend.last.cat;
    final lead = ((youNow - catNow) / catNow * 100).round();
    final bestDay =
        weekTraffic.reduce((a, b) => a.views >= b.views ? a : b).day;

    return Scaffold(
      appBar:
          AppBar(title: const Text('Reports'), backgroundColor: SpottoColors.sand),
      body: ListView(
      padding: const EdgeInsets.fromLTRB(20, 8, 20, 24),
      children: [
        // stat pills
        Row(
          children: [
            Expanded(
                child: _statPill(text, '$totalViews', 'views this week',
                    [for (final d in weekTraffic) d.views])),
            const SizedBox(width: 10),
            Expanded(
                child: _statPill(text, '$totalRedeems', 'redemptions',
                    [for (final d in weekTraffic) d.redemptions])),
          ],
        ),
        const SizedBox(height: 10),
        Row(
          children: [
            Expanded(
                child: _statPill(
                    text,
                    '${peakHour > 12 ? peakHour - 12 : peakHour} ${peakHour >= 12 ? 'PM' : 'AM'}',
                    'busiest hour',
                    null)),
            const SizedBox(width: 10),
            Expanded(child: _statPill(text, bestDay, 'best day', null)),
          ],
        ),
        const SizedBox(height: 16),

        // trend
        VendorCard(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text('You vs category', style: text.titleMedium?.copyWith(fontSize: 16)),
              Text('Weekly redemptions, last 6 weeks',
                  style: text.bodySmall?.copyWith(color: SpottoColors.muted)),
              const SizedBox(height: 12),
              const TrendChart(),
              const SizedBox(height: 12),
              Container(
                padding: const EdgeInsets.all(12),
                decoration: BoxDecoration(
                  color: SpottoColors.accentTint,
                  borderRadius: BorderRadius.circular(14),
                ),
                child: Row(
                  children: [
                    const Icon(Icons.insights, size: 18, color: SpottoColors.accent),
                    const SizedBox(width: 10),
                    Expanded(
                      child: Text(
                          "You're ${math.max(0, lead)}% ahead of the average "
                          'Weligama café — the growth started when you added the '
                          'pour-over offer in W4.',
                          style:
                              text.bodySmall?.copyWith(color: SpottoColors.ink)),
                    ),
                  ],
                ),
              ),
            ],
          ),
        ),
        const SizedBox(height: 16),

        VendorCard(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text('Traffic this week', style: text.titleMedium?.copyWith(fontSize: 16)),
              Text('Profile views with redemptions overlaid',
                  style: text.bodySmall?.copyWith(color: SpottoColors.muted)),
              const SizedBox(height: 12),
              const TrafficChart(),
            ],
          ),
        ),
        const SizedBox(height: 16),

        VendorCard(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text('Popular times', style: text.titleMedium?.copyWith(fontSize: 16)),
              Text('Relative footfall by hour, peak highlighted',
                  style: text.bodySmall?.copyWith(color: SpottoColors.muted)),
              const SizedBox(height: 12),
              const PopularTimesChart(),
            ],
          ),
        ),
      ],
      ),
    );
  }

  Widget _statPill(TextTheme text, String value, String label, List<num>? spark) =>
      VendorCard(
        padding: const EdgeInsets.all(14),
        child: Row(
          children: [
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(value, style: text.headlineSmall?.copyWith(fontSize: 22)),
                  Text(label,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: text.bodySmall?.copyWith(color: SpottoColors.muted)),
                ],
              ),
            ),
            if (spark != null) Sparkline(values: spark, width: 56, height: 26),
          ],
        ),
      );
}
