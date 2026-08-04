import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/theme/palette.dart';
import '../../../data/models/models.dart';
import '../../../data/repositories/repositories.dart';

/// Shows traveler-facing "What's happening nearby" Events discovery sheet.
class EventsSheet extends ConsumerWidget {
  const EventsSheet({super.key, required this.onSelectPlace});

  final ValueChanged<Place> onSelectPlace;

  static void show(BuildContext context, ValueChanged<Place> onSelectPlace) {
    showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      backgroundColor: Colors.transparent,
      builder: (ctx) => EventsSheet(onSelectPlace: onSelectPlace),
    );
  }

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final places = ref.watch(placesProvider);
    // Filter places that have event/party/music/festival in their type or tags
    final events = places.where((p) {
      final type = p.type.toLowerCase();
      final tags = p.tags.join(' ').toLowerCase();
      final deal = (p.deal.label ?? '').toLowerCase();
      return type.contains('event') ||
          type.contains('party') ||
          type.contains('music') ||
          type.contains('festival') ||
          tags.contains('event') ||
          tags.contains('party') ||
          tags.contains('music') ||
          deal.contains('dj') ||
          deal.contains('music') ||
          deal.contains('event') ||
          deal.contains('party');
    }).toList();

    final text = Theme.of(context).textTheme;

    return Container(
      height: MediaQuery.of(context).size.height * 0.7,
      decoration: const BoxDecoration(
        color: SpottoColors.sand,
        borderRadius: BorderRadius.vertical(top: Radius.circular(24)),
      ),
      child: Column(
        children: [
          const SizedBox(height: 12),
          Container(
            width: 40,
            height: 4,
            decoration: BoxDecoration(
              color: SpottoColors.line,
              borderRadius: BorderRadius.circular(2),
            ),
          ),
          const SizedBox(height: 16),
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 20),
            child: Row(
              children: [
                const Text('🎉', style: TextStyle(fontSize: 22)),
                const SizedBox(width: 10),
                Text("What's Happening Nearby", style: text.headlineSmall),
              ],
            ),
          ),
          const SizedBox(height: 12),
          Expanded(
            child: events.isEmpty
                ? Center(
                    child: Padding(
                      padding: const EdgeInsets.all(24),
                      child: Column(
                        mainAxisAlignment: MainAxisAlignment.center,
                        children: [
                          const Text('🌙', style: TextStyle(fontSize: 40)),
                          const SizedBox(height: 12),
                          Text('No events scheduled for tonight',
                              style: text.titleMedium),
                          const SizedBox(height: 4),
                          Text('Check back soon for beach parties and live music!',
                              style: text.bodyMedium
                                  ?.copyWith(color: SpottoColors.muted),
                              textAlign: TextAlign.center),
                        ],
                      ),
                    ),
                  )
                : ListView.separated(
                    padding: const EdgeInsets.all(20),
                    itemCount: events.length,
                    separatorBuilder: (_, __) => const SizedBox(height: 12),
                    itemBuilder: (context, index) {
                      final item = events[index];
                      return InkWell(
                        onTap: () {
                          Navigator.pop(context);
                          onSelectPlace(item);
                        },
                        child: Container(
                          padding: const EdgeInsets.all(16),
                          decoration: BoxDecoration(
                            color: Colors.white,
                            borderRadius: BorderRadius.circular(16),
                            border: Border.all(color: SpottoColors.line),
                          ),
                          child: Column(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              Row(
                                children: [
                                  Expanded(
                                    child: Text(
                                      item.name,
                                      style: text.titleMedium
                                          ?.copyWith(fontWeight: FontWeight.bold),
                                    ),
                                  ),
                                  Container(
                                    padding: const EdgeInsets.symmetric(
                                        horizontal: 8, vertical: 4),
                                    decoration: BoxDecoration(
                                      color: SpottoColors.accentTint,
                                      borderRadius: BorderRadius.circular(8),
                                    ),
                                    child: Text(
                                      item.region.isNotEmpty ? item.region : item.type,
                                      style: text.bodySmall?.copyWith(
                                        color: SpottoColors.accent,
                                        fontWeight: FontWeight.bold,
                                      ),
                                    ),
                                  ),
                                ],
                              ),
                              if (item.deal.active && item.deal.label != null) ...[
                                const SizedBox(height: 8),
                                Text(
                                  item.deal.label!,
                                  style: text.bodyMedium?.copyWith(
                                    color: SpottoColors.ink,
                                    fontWeight: FontWeight.w600,
                                  ),
                                ),
                              ],
                            ],
                          ),
                        ),
                      );
                    },
                  ),
          ),
        ],
      ),
    );
  }
}
