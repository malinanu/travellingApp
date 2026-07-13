// Spots tab: scrollable business cards — port of SpotsList.jsx.

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/theme/palette.dart';
import '../../../core/widgets/illustration.dart';
import '../../../core/widgets/spotto_card.dart';
import '../../../data/repositories/repositories.dart';
import '../place_detail/business_drawer.dart';

class SpotsScreen extends ConsumerWidget {
  const SpotsScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final text = Theme.of(context).textTheme;
    final places = ref.watch(placesProvider);
    return SafeArea(
      child: ListView(
        padding: const EdgeInsets.fromLTRB(16, 12, 16, 90),
        children: [
          Text('Spots in Weligama', style: text.headlineSmall),
          const SizedBox(height: 4),
          Text('${places.length} curated places nearby', style: text.bodyMedium),
          const SizedBox(height: 16),
          for (final p in places) ...[
            SpottoCard(
              padding: EdgeInsets.zero,
              onTap: () => showBusinessDrawer(context, p),
              child: ClipRRect(
                borderRadius: BorderRadius.circular(18),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    SizedBox(
                      height: 130,
                      width: double.infinity,
                      child: Stack(
                        fit: StackFit.expand,
                        children: [
                          Illustration(kind: p.photo),
                          if (p.deal.active)
                            Positioned(
                              left: 12,
                              top: 12,
                              child: Container(
                                padding:
                                    const EdgeInsets.symmetric(horizontal: 10, vertical: 5),
                                decoration: BoxDecoration(
                                  color: SpottoColors.sun,
                                  borderRadius: BorderRadius.circular(999),
                                ),
                                child: Text(p.deal.tag ?? 'Deal',
                                    style: text.labelSmall
                                        ?.copyWith(color: SpottoColors.ink)),
                              ),
                            ),
                          Positioned(
                            left: 14,
                            right: 14,
                            bottom: 10,
                            child: Text(p.name,
                                style: text.titleMedium?.copyWith(color: Colors.white)),
                          ),
                        ],
                      ),
                    ),
                    Padding(
                      padding: const EdgeInsets.all(12),
                      child: Row(
                        children: [
                          const Icon(Icons.star, size: 16, color: SpottoColors.sun),
                          const SizedBox(width: 4),
                          Text('${p.rating}', style: text.labelLarge),
                          Text('  ·  ${p.type}  ·  ${p.distance}', style: text.bodySmall),
                          const Spacer(),
                          if (p.wifi.score >= 8) ...[
                            const Icon(Icons.wifi, size: 15, color: SpottoColors.accent),
                            const SizedBox(width: 3),
                            Text('${p.wifi.score}', style: text.labelMedium),
                          ],
                        ],
                      ),
                    ),
                  ],
                ),
              ),
            ),
            const SizedBox(height: 14),
          ],
        ],
      ),
    );
  }
}
